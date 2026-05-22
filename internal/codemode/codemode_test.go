package codemode_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-assern/internal/codemode"
)

func echoCaller(_ context.Context, name string, _ map[string]any) (string, error) {
	return "result:" + name, nil
}

func TestRunCallsToolAndPrints(t *testing.T) {
	t.Parallel()

	exec := codemode.New(codemode.Options{Call: echoCaller})

	res, err := exec.Run(context.Background(), `
out = call("github_search", {"query": "x"})
print(out)
`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(res.Output, "result:github_search") {
		t.Errorf("output missing tool result: %q", res.Output)
	}

	if res.Calls != 1 {
		t.Errorf("Calls = %d, want 1", res.Calls)
	}
}

func TestRunComposesMultipleTools(t *testing.T) {
	t.Parallel()

	var seen []string

	caller := func(_ context.Context, name string, _ map[string]any) (string, error) {
		seen = append(seen, name)

		return name + "-ok", nil
	}

	exec := codemode.New(codemode.Options{Call: caller})

	res, err := exec.Run(context.Background(), `
a = call("alpha", {})
b = call("beta", {})
print(a + "|" + b)
`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if res.Calls != 2 {
		t.Errorf("Calls = %d, want 2", res.Calls)
	}

	if !strings.Contains(res.Output, "alpha-ok|beta-ok") {
		t.Errorf("composed output unexpected: %q", res.Output)
	}

	if len(seen) != 2 || seen[0] != "alpha" || seen[1] != "beta" {
		t.Errorf("backend received %v, want [alpha beta]", seen)
	}
}

func TestRunSearchBuiltin(t *testing.T) {
	t.Parallel()

	search := func(_ string, _ int) []codemode.ToolMatch {
		return []codemode.ToolMatch{
			{Name: "github_search", Server: "github", Description: "Search repos"},
			{Name: "linear_search", Server: "linear", Description: "Search tickets"},
		}
	}

	exec := codemode.New(codemode.Options{Call: echoCaller, Search: search})

	res, err := exec.Run(context.Background(), `
for tool in search("search"):
    print(tool["name"])
`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, want := range []string{"github_search", "linear_search"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("search output missing %q: %q", want, res.Output)
		}
	}
}

func TestRunEnforcesCallLimit(t *testing.T) {
	t.Parallel()

	exec := codemode.New(codemode.Options{Call: echoCaller, MaxCalls: 2})

	_, err := exec.Run(context.Background(), `
call("a", {})
call("b", {})
call("c", {})
`)
	if err == nil {
		t.Fatal("expected error when exceeding call limit")
	}

	if !strings.Contains(err.Error(), "limit") {
		t.Errorf("error = %v, want a call-limit message", err)
	}
}

func TestRunTruncatesOutput(t *testing.T) {
	t.Parallel()

	exec := codemode.New(codemode.Options{Call: echoCaller, MaxOutput: 40})

	res, err := exec.Run(context.Background(), `
for i in range(100):
    print("a fairly long line of output text")
`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(res.Output, "truncated") {
		t.Errorf("expected truncation marker, got %q", res.Output)
	}

	// Allow some slack for the truncation marker itself.
	if len(res.Output) > 80 {
		t.Errorf("output not bounded: %d bytes", len(res.Output))
	}
}

func TestRunMalformedCode(t *testing.T) {
	t.Parallel()

	exec := codemode.New(codemode.Options{Call: echoCaller})

	if _, err := exec.Run(context.Background(), `this is ((not valid`); err == nil {
		t.Fatal("expected error for malformed script")
	}
}

func TestRunSurfacesCallerError(t *testing.T) {
	t.Parallel()

	failing := func(_ context.Context, _ string, _ map[string]any) (string, error) {
		return "", errors.New("backend boom")
	}

	exec := codemode.New(codemode.Options{Call: failing})

	_, err := exec.Run(context.Background(), `call("x", {})`)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want it to surface backend boom", err)
	}
}

func TestRunTimesOut(t *testing.T) {
	t.Parallel()

	// A caller that blocks until the context is cancelled lets us assert the
	// timeout fires deterministically without a long-running loop.
	blocking := func(ctx context.Context, _ string, _ map[string]any) (string, error) {
		<-ctx.Done()

		return "", ctx.Err()
	}

	exec := codemode.New(codemode.Options{Call: blocking, Timeout: 50 * time.Millisecond})

	start := time.Now()
	_, err := exec.Run(context.Background(), `call("slow", {})`)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	// The error must reflect the deadline, not some unrelated failure.
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %v, want a context-deadline message", err)
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestRunRejectsDeeplyNestedArgs(t *testing.T) {
	t.Parallel()

	var received map[string]any

	caller := func(_ context.Context, _ string, args map[string]any) (string, error) {
		received = args

		return "ok", nil
	}

	exec := codemode.New(codemode.Options{Call: caller})

	// Build a structure deeper than the conversion limit (64) using a loop.
	code := `
x = "leaf"
for i in range(200):
    x = [x]
call("t", {"deep": x})
`

	_, err := exec.Run(context.Background(), code)
	if err == nil {
		t.Fatalf("expected error for deeply nested args, got nil (received=%v)", received)
	}

	if !strings.Contains(err.Error(), "too deep") {
		t.Errorf("error = %v, want a nesting-depth message", err)
	}
}

func TestRunRequiresCaller(t *testing.T) {
	t.Parallel()

	exec := codemode.New(codemode.Options{})

	if _, err := exec.Run(context.Background(), `print("hi")`); !errors.Is(err, codemode.ErrNoToolCaller) {
		t.Fatalf("error = %v, want ErrNoToolCaller", err)
	}
}
