# Code Mode

Calling tools one at a time is chatty: each call is a separate round-trip, and
every intermediate result lands back in the model's context. **Code mode** lets
the model write a short script that orchestrates several tools in a single call
and returns just the final result.

Code mode exposes one meta-tool, `assern_execute`, which runs a **sandboxed
Starlark script**. Starlark is a deterministic Python dialect with no file,
network, or import access — a script can only do what Assern explicitly exposes:
search the tool catalog and call tools.

Code mode is **opt-in** and adds a code-execution surface, so enable it
deliberately.

## Enabling code mode

```yaml
# ~/.valksor/assern/config.yaml
settings:
  code_mode:
    enabled: true
    timeout: 30s            # wall-clock limit per script
    max_tool_calls: 50      # cap on call() invocations per script
    max_output_bytes: 65536 # cap on captured output
    allowed_tools:          # optional: restrict which tools call() may invoke
      - github_search_repos # (empty = any aggregated tool)
      - github_list_issues
```

Code mode is independent of [tool discovery](discovery.md): you can run it with
discovery on or off.

## The `assern_execute` tool

The script has two builtins plus `print`:

| Builtin | Description |
|---------|-------------|
| `search(query[, limit])` | Returns a list of `{name, server, description}` dicts for matching tools. |
| `call(name, args)` | Invokes a tool by its **prefixed** name (e.g. `github_search_repos`) with an args dict; returns the tool's text result. |
| `print(...)` | Emits output. The script's printed output is what `assern_execute` returns. |

### Example

```python
# Find the open issues for a repo, then summarise their titles.
issues = call("github_list_issues", {"repo": "valksor/go-assern", "state": "open"})

titles = []
for line in issues.split("\n"):
    if line.strip():
        titles.append(line.strip())

print("Open issues: %d" % len(titles))
for t in titles:
    print("- " + t)
```

This runs as **one** `assern_execute` call instead of one `tools/list`-style call
per step, and only the printed summary returns to the model.

## Limits and safety

Every script runs under hard limits:

- **Timeout** (`timeout`) — wall-clock bound; the script is cancelled when it elapses.
- **Tool-call cap** (`max_tool_calls`) — `call()` fails once the limit is hit.
- **Output cap** (`max_output_bytes`) — captured output is truncated past the limit.
- **Step limit** — a built-in guard aborts runaway loops.
- **No ambient authority** — no filesystem, network, or module access; recursion is disabled, and argument nesting depth is bounded.

A script error (or a failing tool call) returns an error result that includes any
output produced before the failure, so partial progress is visible.

> **Trust model:** by default a script may call **any** aggregated tool. If some
> connected servers expose sensitive or destructive tools, restrict code mode
> with `allowed_tools`, and/or limit what each server exposes with its per-server
> `allowed` list. Enable code mode only when you trust the connected servers and
> the client driving Assern.

## When to use it

Reach for code mode when a task naturally chains tools — fan-out/fan-in, filtering
one tool's output into another, or looping over results. For a single tool call,
call the tool directly (load it first if discovery is enabled).
