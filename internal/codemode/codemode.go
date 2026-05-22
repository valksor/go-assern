// Package codemode runs short, sandboxed Starlark scripts that orchestrate
// several aggregated MCP tools in a single call. This "code mode" cuts the
// round-trips and intermediate context of calling tools one at a time.
//
// Starlark is a deterministic Python dialect with no file, network, or module
// access by default, so a script can only do what the host explicitly exposes:
// here, search the tool catalog and call tools. Execution is further bounded by
// a wall-clock timeout, a tool-call cap, an output-size cap, and a step limit.
package codemode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// scriptOptions enables a friendlier dialect for orchestration scripts:
// top-level if/for/while and reassignment, plus the set builtin. Recursion
// stays disabled (the default) to bound execution.
var scriptOptions = &syntax.FileOptions{
	Set:             true,
	While:           true,
	TopLevelControl: true,
	GlobalReassign:  true,
}

// Execution-limit defaults, used when the corresponding option is non-positive.
const (
	DefaultTimeout   = 30 * time.Second
	DefaultMaxCalls  = 50
	DefaultMaxOutput = 64 * 1024

	// maxSteps guards against runaway loops independent of wall-clock time.
	maxSteps = 50_000_000
)

// ErrNoToolCaller indicates the executor was built without a tool caller.
var ErrNoToolCaller = errors.New("codemode: no tool caller configured")

// ToolMatch is a catalog search result surfaced to scripts.
type ToolMatch struct {
	Name        string
	Server      string
	Description string
}

// ToolCaller routes a tool call to the backend and returns its textual result.
type ToolCaller func(ctx context.Context, name string, args map[string]any) (string, error)

// ToolSearcher ranks catalog tools for a query.
type ToolSearcher func(query string, limit int) []ToolMatch

// Options configures an Executor.
type Options struct {
	Call      ToolCaller
	Search    ToolSearcher
	Timeout   time.Duration
	MaxCalls  int
	MaxOutput int
}

// Executor runs Starlark scripts with tool-orchestration builtins.
type Executor struct {
	call      ToolCaller
	search    ToolSearcher
	timeout   time.Duration
	maxCalls  int
	maxOutput int
}

// New creates an Executor, applying defaults for any unset limits.
func New(opts Options) *Executor {
	e := &Executor{
		call:      opts.Call,
		search:    opts.Search,
		timeout:   opts.Timeout,
		maxCalls:  opts.MaxCalls,
		maxOutput: opts.MaxOutput,
	}

	if e.timeout <= 0 {
		e.timeout = DefaultTimeout
	}

	if e.maxCalls <= 0 {
		e.maxCalls = DefaultMaxCalls
	}

	if e.maxOutput <= 0 {
		e.maxOutput = DefaultMaxOutput
	}

	return e
}

// Result is the outcome of running a script.
type Result struct {
	// Output is the script's captured print() output (possibly truncated).
	Output string
	// Calls is the number of tool calls the script made.
	Calls int
}

// Run executes code and returns its captured output. Scripts call tools via
// call(name, args) and discover them via search(query[, limit]). On a script
// error, the partial output is still returned alongside the error.
func (e *Executor) Run(ctx context.Context, code string) (Result, error) {
	if e.call == nil {
		return Result{}, ErrNoToolCaller
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	out := &boundedBuffer{limit: e.maxOutput}
	callCount := 0

	thread := &starlark.Thread{
		Name: "assern_execute",
		Print: func(_ *starlark.Thread, msg string) {
			out.writeLine(msg)
		},
		// Explicitly disable module loading so the sandbox can never gain
		// import capability, even if a future change wires up a Load function.
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, fmt.Errorf("load is disabled: module %q is not available in code mode", module)
		},
	}
	thread.SetMaxExecutionSteps(maxSteps)

	// Cancel the script promptly when the context is done (timeout or caller
	// cancellation). Cancel is safe to call from another goroutine.
	stop := make(chan struct{})
	defer close(stop)

	go func() {
		select {
		case <-ctx.Done():
			thread.Cancel(ctx.Err().Error())
		case <-stop:
		}
	}()

	predeclared := starlark.StringDict{
		"call":   e.callBuiltin(ctx, &callCount),
		"search": e.searchBuiltin(),
	}

	_, execErr := starlark.ExecFileOptions(scriptOptions, thread, "assern_execute.star", code, predeclared)

	result := Result{Output: out.string(), Calls: callCount}
	if execErr != nil {
		return result, fmt.Errorf("codemode: %w", execErr)
	}

	return result, nil
}

// callBuiltin returns the `call(name, args={})` builtin, enforcing the call cap.
func (e *Executor) callBuiltin(ctx context.Context, count *int) *starlark.Builtin {
	return starlark.NewBuiltin("call", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			name    string
			argsVal starlark.Value
		)

		if err := starlark.UnpackArgs("call", args, kwargs, "name", &name, "args?", &argsVal); err != nil {
			return nil, err
		}

		if *count >= e.maxCalls {
			return nil, fmt.Errorf("tool-call limit reached (%d); reduce the number of call() invocations", e.maxCalls)
		}

		goArgs, err := valueToArgs(argsVal)
		if err != nil {
			return nil, fmt.Errorf("call %q: %w", name, err)
		}

		// Count attempted calls (including ones that fail) so a script cannot
		// bypass the cap by triggering fast-failing calls in a loop.
		*count++

		text, err := e.call(ctx, name, goArgs)
		if err != nil {
			return nil, fmt.Errorf("call %q: %w", name, err)
		}

		return starlark.String(text), nil
	})
}

// searchBuiltin returns the `search(query, limit=0)` builtin.
func (e *Executor) searchBuiltin() *starlark.Builtin {
	return starlark.NewBuiltin("search", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			query string
			limit int
		)

		if err := starlark.UnpackArgs("search", args, kwargs, "query", &query, "limit?", &limit); err != nil {
			return nil, err
		}

		if e.search == nil {
			return starlark.NewList(nil), nil
		}

		matches := e.search(query, limit)

		items := make([]starlark.Value, 0, len(matches))
		for _, m := range matches {
			dict := starlark.NewDict(3)
			if err := setStrings(dict, map[string]string{
				"name":        m.Name,
				"server":      m.Server,
				"description": m.Description,
			}); err != nil {
				return nil, err
			}

			items = append(items, dict)
		}

		return starlark.NewList(items), nil
	})
}

// setStrings sets string key/value pairs on a Starlark dict.
func setStrings(dict *starlark.Dict, pairs map[string]string) error {
	for k, v := range pairs {
		if err := dict.SetKey(starlark.String(k), starlark.String(v)); err != nil {
			return fmt.Errorf("building search result: %w", err)
		}
	}

	return nil
}
