# Tool Discovery

When you aggregate many MCP servers, every tool definition is sent to the LLM's
context on connection. With dozens of servers this can be thousands of tokens
before the model does anything — and a long tool list also degrades the model's
tool-selection accuracy.

**Tool discovery** (progressive disclosure) solves this. Instead of exposing
every aggregated tool up front, Assern exposes a small set of `assern_*`
meta-tools. The client searches the catalog and loads only the tools it needs,
at runtime, scoped to its own session.

Discovery is **opt-in**. With it off (the default), Assern behaves exactly as
before — every tool is exposed at startup.

## Enabling discovery

```yaml
# ~/.valksor/assern/config.yaml
settings:
  discovery:
    enabled: true
    pinned:            # optional: always-on tools, exposed without a search
      - github_search_repos
    max_results: 10    # default number of matches assern_search returns
    max_loaded: 30     # per-session ceiling; oldest tools evicted past it
```

> **`max_loaded` values:** a positive number is the ceiling. **Unset or `0`**
> falls back to the default (`30`). **`-1`** means **unlimited** (no eviction).

## How it works

With discovery enabled, a connecting client initially sees only:

| Tool | Purpose |
|------|---------|
| `assern_search` | Search the catalog by keyword. Returns matching tool names, descriptions, and estimated token cost. **Does not** load them. |
| `assern_load` | Make one or more tools (by prefixed name) callable in this session. |
| `assern_forget` | Unload tools to free context. |
| *pinned tools* | Any tools listed in `discovery.pinned`. |

A typical agent flow:

1. `assern_search({"query": "create github issue"})` → returns `github_create_issue` (+ schema info).
2. `assern_load({"names": ["github_create_issue"]})` → the tool becomes callable; Assern emits `notifications/tools/list_changed`.
3. The client re-fetches `tools/list` and now sees `github_create_issue` as a **native, schema-validated** tool, and calls it normally.
4. (optional) `assern_forget({"names": ["github_create_issue"]})` when done.

Because tools are loaded with MCP **per-session** semantics, one client's
discovery activity never changes the tool list another client sees — important
when several clients share one Assern instance via the instance-sharing socket.

### Eviction

When a session reaches `max_loaded`, loading another tool evicts the
least-recently loaded one (it is removed from that session and announced via
`tools/list_changed`). `max_loaded: -1` disables the ceiling entirely; `0` or
unset uses the default of `30`.

## Measuring the benefit

`assern list` reports the estimated token cost of exposed tool definitions:

```
Total: 76 tools, ~12.4k tokens
By server:
  - github               ~4.1k tokens
  - linear               ~3.0k tokens
  ...
```

The estimate is a relative heuristic (≈ characters / 4), useful for comparing
servers and for seeing the before/after impact of enabling discovery — not an
exact tokenizer count.

## Client compatibility

Discovery relies on the MCP `tools/list_changed` notification, which Assern emits
when tools are loaded or evicted. Clients that honor `list_changed` (such as
Claude) pick up loaded tools automatically. The `assern_*` meta-tools themselves
work with any MCP client.

## Notes

- The reserved `assern_` prefix is used for meta-tools so they never collide with
  aggregated `server_tool` names. Avoid naming a backend server `assern`.
- Resources and prompts are always exposed in full; discovery applies to tools.
- `pinned` entries use the **prefixed** name (e.g. `github_search_repos`), the
  same name shown by `assern list` and `assern_search`.
