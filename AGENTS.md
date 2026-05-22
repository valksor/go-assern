# CLAUDE.md

# IT IS YEAR 2026 !!! Please use 2026 in web searches!!!

Guidance for Claude Code when working with go-assern.

## Project Overview

Assern is a **Go CLI + library** for aggregating multiple MCP (Model Context Protocol) servers into a unified interface. It combines tools, resources, and prompts from multiple MCP servers with automatic prefixing for namespace isolation.

**Key constraint**: All aggregated tools/resources/prompts must maintain MCP protocol compliance.

---

## Critical Rules

### 1. go-toolkit: Import Directly

Use `github.com/valksor/go-toolkit` packages directly. **NO type aliases, NO wrapper functions, NO re-exports.**

```go
// GOOD - Direct import
import "github.com/valksor/go-toolkit/cli"
disambiguate.Parse(args)

// BAD - Type alias or wrapper
type Parser = disambiguate.Parser  // Don't do this
```

**When to add to go-toolkit**: Generic utilities with no assern dependencies.
**When to add to go-assern**: Domain-specific MCP aggregation logic.

### 2. Tests & Docs Required

Every feature MUST include:

| Requirement | Location | Target |
|-------------|----------|--------|
| Unit tests | `*_test.go` next to source | 80%+ coverage |
| Integration tests | `internal/testutil/` | Critical paths |
| CLI docs | `docs/` | Usage + examples |

Write tests FIRST (TDD). Use table-driven tests. Run `make test` before committing code.

### 3. Quality Checks by Scope

Run checks **only for code you changed**:

| Changed | Command |
|---------|---------|
| `cmd/`, `internal/`, `*.go` | `make quality && make test` |
| `docs/**`, `*.md` | None |

If tests fail, fix them first. No exceptions for "not my code."

### 4. Use Make Commands

Always use `make` commands, not direct `go` commands:

| Operation | Command              |
|-----------|----------------------|
| Build     | `make build`         |
| Test      | `make test`          |
| Race      | `make race`          |
| Quality   | `make quality`       |
| Format    | `make fmt`           |
| Coverage  | `make coverage-html` |
| Install   | `make install`       |

`make quality` runs: golangci-lint, gofmt, goimports, gofumpt, govulncheck, check-alias.

### 5. No nolint Abuse

**`//nolint` is a LAST RESORT.** Never disable linters in `.golangci.yml`.

**Acceptable**:
- `//nolint:unparam // Required by interface`
- `//nolint:nilnil // No server found is not an error`
- `//nolint:errcheck // String builder WriteString won't fail`

**Never acceptable**:
- `//nolint:errcheck` without justification
- `//nolint:gosec` (fix the security issue)
- `//nolint:all` (never suppress all linters)

Always: specify linter name, include justification, place on specific line.

### 6. File Size < 500 Lines

Keep all Go files under 500 lines. Split by feature or responsibility:

```go
// Split aggregator.go (1000 lines) into:
aggregator_core.go    // Core aggregation logic
aggregator_tools.go   // Tool handling
aggregator_health.go  // Health tracking
```

**Exceptions**: Generated code, single-responsibility modules.

### 7. Git Command Policy

All git commands are classified into three tiers. **No exceptions, no force flags, no overrides.** Tier 2 and 3 commands are never used autonomously — the agent must have explicit user instruction before running any write operation on the repository.

#### Tier 1 — Always Allowed

Safe read-only commands, available anytime:

`git status`, `git diff`, `git log`, `git show`, `git blame`, `git grep`, `git branch` (read-only), `git remote -v` (read-only), `git fetch`, `git reflog`, `git shortlog`, `git describe`, `git checkout`, `git switch`, `git restore`

#### Tier 2 — User-Requested Only

**Only use when the user explicitly asks.** Never run these commands autonomously — not for convenience, not as part of a workflow, not "to be helpful." If the task seems to need one of these commands but the user hasn't asked, ask first.

`git add`, `git commit`, `git rm`, `git mv`, `git apply`, `git am`

#### Tier 3 — Always Blocked

**NEVER use these commands.** No time window, no override, no exceptions:

`git push`, `git pull`, `git merge`, `git rebase`, `git reset`, `git revert`, `git cherry-pick`, `git tag`, `git stash` (all subcommands), `git worktree` (all subcommands), `git clean`, `git bisect`, `git notes`, `git submodule` (write operations)

Do not suggest, recommend, or implement workflows that rely on any Tier 3 command. If a task seems to need one, use a Tier 1 or Tier 2 alternative, or ask the user to perform the operation manually.

**⛔ `git worktree` — absolute prohibition.** No `git worktree add`, `remove`, `list`, `prune`, or any other worktree subcommand. Do not suggest, recommend, or implement any workflow that involves worktrees. No force flag, no override, no exceptions — ever. If a task seems to benefit from worktrees, use separate clones or branches instead.

---

## Commands

### Build & Development

```bash
make build | install | test | coverage | quality | fmt | tidy | hooks | race
```

All targets: `build`, `test`, `race`, `coverage`, `coverage-html`, `quality`, `fmt`, `install`, `clean`, `run`, `tidy`, `deps`, `version`, `hooks`, `lefthook`, `help`.

### CLI Commands

```bash
assern serve              # Start aggregator (default)
assern list               # List servers/tools/resources/prompts
assern reload             # Hot-reload configuration
assern mcp add|edit|delete|list  # MCP server management
assern config init|validate      # Configuration management
assern version            # Version info
```

**Colon notation supported**: `mcp:add`, `config:init`, `list:servers`

---

## Architecture

### Entry Point

`cmd/assern/main.go` → Cobra commands → package handlers

### Core Packages

| Package | Responsibility |
|---------|----------------|
| `internal/aggregator/` | Core MCP aggregation - combines servers, prefixes tools/resources/prompts, health tracking |
| `internal/cli/` | CLI components - MCP CRUD, prompts, formatters, validators |
| `internal/config/` | Configuration - loading, merging, validation, path resolution |
| `internal/instance/` | Instance sharing - Unix socket detection to prevent cascade spawning |
| `internal/transport/` | Transport layer - stdio handling, logging, signal handling |

### Key Patterns

**Aggregation**: Multiple MCP servers → single unified interface with namespaced tools.

**Prefixing**: Tools become `server_tool`, resources become `assern://server/uri`, prompts become `server_prompt`.

**Instance Sharing**: Unix socket at `~/.valksor/assern/assern.sock` prevents nested LLM cascade spawning.

**Hot-Reload**: Configuration updates via `assern reload` or SIGHUP signal without restart.

### Configuration Hierarchy

1. Global MCP: `~/.valksor/assern/mcp.json`
2. Global Config: `~/.valksor/assern/config.yaml`
3. Local MCP: `.assern/mcp.json` (project-specific)
4. Local Config: `.assern/config.yaml` (project-specific)
5. Environment: `.env` files

---

## Code Style

- **Imports**: stdlib → third-party → local (alphabetical within groups)
- **Naming**: PascalCase exported, camelCase unexported
- **Errors**: `fmt.Errorf("prefix: %w", err)`; `errors.Join(errs...)`
- **Logging**: `log/slog`
- **Formatting**: `make fmt` (gofmt, goimports, gofumpt)
- **Quality**: `make quality`

### Modern Go (1.25+)

- Use `slices.Contains()`, `maps.Clone()` instead of manual loops
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done() }()`
- Always pass `context.Context` for cancelable operations

---

## Testing

- Run: `make test`
- Coverage: `make coverage-html` (output: `.coverage/coverage.html`)
- Style: Table-driven with `tests := []struct{...}{...}`
- Utilities: `internal/testutil/` (mocks, fixtures)
- Target: 80%+ coverage
- Race detector: `make race`

---

## See Also

- [README.md](README.md) - Installation, quick start
- [docs/quickstart.md](docs/quickstart.md) - Getting started guide
- [docs/configuration.md](docs/configuration.md) - Configuration reference
- [docs/concepts.md](docs/concepts.md) - Design concepts
