# Concepts

Understanding how Assern works under the hood.

## Architecture Overview

Assern acts as an aggregation layer between MCP clients and multiple backend servers.

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│   MCP Client (Claude, other AI assistants)                      │
│                                                                  │
│   Single stdio connection ──────────────────────────────────┐    │
│                                                             │    │
│   ┌─────────────────────────────────────────────────────┐   │    │
│   │              Assern Aggregator                      │   │    │
│   │                                                     │   │    │
│   │   1. Configuration Resolution                       │   │    │
│   │   2. Project Detection                              │   │    │
│   │   3. Server Lifecycle Management                    │   │    │
│   │   4. Tool Registration & Prefixing                  │   │    │
│   │   5. Request Routing                                │   │    │
│   └─────────────────────────────────────────────────────┘   │    │
│                                                             │    │
│   ┌──────────┬──────────┬──────────┬──────────┐           │    │
│   │          │          │          │          │           │    │
│   ▼          ▼          ▼          ▼          │           │    │
│  ┌─────┐  ┌─────┐   ┌─────┐   ┌─────┐        │           │    │
│  │ git │  │ gh  │   │jira │   │ slack│ ...   │           │    │
│  │     │  │     │   │     │   │      │        │           │    │
│  └─────┘  └─────┘   └─────┘   └─────┘        │           │    │
│                                            ▼            ▼    │
│                                         stdio out     │       │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

1. **Configuration Resolver** - Merges global, project, and local configs
2. **Project Detector** - Determines active project from directory
3. **Server Manager** - Spawns and manages MCP server processes
4. **Tool Registry** - Maintains prefixed tool names and routing
5. **Resource Registry** - Maintains prefixed resource URIs and routing
6. **Prompt Registry** - Maintains prefixed prompt names and routing
7. **Request Router** - Directs tool calls, resource reads, and prompt requests to appropriate backend

## MCP Protocol Support

Assern aggregates all three major MCP capabilities from backend servers:

| Capability | Description | Prefixing |
|------------|-------------|-----------|
| **Tools** | Callable functions | `{server}_{tool}` |
| **Resources** | Readable data sources | `assern://{server}/{uri}` |
| **Prompts** | Reusable prompt templates | `{server}_{prompt}` |

### How It Works

When a backend server exposes tools, resources, or prompts, Assern:
1. Discovers them during server initialization
2. Prefixes them to prevent naming conflicts
3. Registers handlers that route requests to the original backend
4. Exposes the aggregated capabilities through a single MCP interface

## Tool Prefixing

All tools from backend servers are prefixed with the server name to prevent naming conflicts.

### Why Prefixing?

Without prefixing, two servers exposing tools named `get` would collide. Prefixing creates unique namespaced tools.

### Prefixing Rules

| Server Name | Original Tool | Prefixed Tool |
|-------------|--------------|---------------|
| `github` | `search_repositories` | `github_search_repositories` |
| `jira` | `get_ticket` | `jira_get_ticket` |
| `my-server` | `my_tool` | `my_server_my_tool` |

- Server name used as-is (lowercase recommended)
- Dashes converted to underscores: `my-server` → `my_server`
- Original tool name preserved after the underscore

### Filtering Tools

Use the `allowed` field to expose only specific tools:

```yaml
servers:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
    allowed:
      - read_file
      - list_directory
      # write_file, delete_file, etc. NOT exposed
```

## Resource Prefixing

Resources from backend servers are prefixed with a custom URI scheme to prevent conflicts.

### URI Format

```
assern://{server}/{original-uri}
```

### Examples

| Server | Original URI | Prefixed URI |
|--------|--------------|--------------|
| `github` | `file:///repo/README.md` | `assern://github/file:///repo/README.md` |
| `filesystem` | `file:///home/user/doc.txt` | `assern://filesystem/file:///home/user/doc.txt` |

When a client reads a resource using the prefixed URI, Assern:
1. Parses the server name from the URI
2. Extracts the original URI
3. Routes the read request to the correct backend

## Prompt Prefixing

Prompts from backend servers are prefixed using the same pattern as tools.

### Examples

| Server | Original Prompt | Prefixed Prompt |
|--------|-----------------|-----------------|
| `assistant` | `code-review` | `assistant_code_review` |
| `templates` | `generate-code` | `templates_generate_code` |

Prompt arguments are preserved during aggregation.

## Configuration Resolution Order

Assern resolves configuration in layers, from lowest to highest priority:

```
1. System Environment Variables
       │
2. Global .env (~/.valksor/assern/.env)
       │
3. Global Config (~/.valksor/assern/config.yaml)
       │
4. Project Definition (from global config's projects.*)
       │
5. Local .env (.assern/.env)
       │
6. Local Config (.assern/config.yaml)
       │
▶ Final Configuration
```

### Example

Given:

```yaml
# Global config
servers:
  github:
    env:
      GITHUB_TOKEN: "${DEFAULT_TOKEN}"

projects:
  work:
    env:
      GITHUB_TOKEN: "${WORK_TOKEN}"
```

And a local `.assern/.env` with:

```bash
GITHUB_TOKEN=repo-specific-token
```

Resolution order:
1. `DEFAULT_TOKEN` from system env
2. Overridden by `WORK_TOKEN` from project env
3. Overridden by `repo-specific-token` from local .env

## Project Detection Flow

Assern determines which project configuration to use based on your current directory.

```
Current Directory: /home/user/work/acme/my-repo
                        │
                        ▼
        ┌───────────────────────────────┐
        │ Check .assern/config.yaml     │
        │ in current or parent dirs?    │
        └───────────────────────────────┘
                    │
           ┌────────┴────────┐
           │ Yes             │ No
           ▼                 ▼
    Use local config    Match against
       settings         projects.*.directories
                              │
                    ┌─────────┴─────────┐
                    │ Match found?      │
                    └───────────────────┘
                         │
                ┌────────┴────────┐
                │ Yes             │ No
                ▼                 ▼
         Use project       Check --project flag
           config              │
                              ▼
                        ┌─────────────┐
                        │ Flag provided? │
                        └─────────────┘
                             │
                    ┌────────┴────────┐
                    │ Yes             │ No
                    ▼                 ▼
             Use specified      Error: No project
                project          detected
```

### Directory Pattern Matching

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `~/work/myproject` | `/home/user/work/myproject` | `/home/user/work/other` |
| `~/work/*` | `/home/user/work/project1` | `/home/user/work/org/project` |
| `~/work/**` | `/home/user/work/project`<br>`/home/user/work/org/project`<br>`/home/user/work/org/team/project` | (matches all) |

## Environment Variable Expansion

Use `${VAR}` or `$VAR` syntax for variable expansion in configuration files.

### Expansion Sources

Variables are resolved from:
1. System environment
2. `.env` files (in resolution order)

### Example

```yaml
# Global config
servers:
  github:
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"           # From system env or .env
      API_URL: "https://${GITHUB_DOMAIN}/api"   # Interpolation
```

## Merge Modes

When a project defines environment variables, they merge with server environment variables.

### Overlay Mode (Default)

Project values override, but non-overridden server values are preserved.

```yaml
servers:
  github:
    merge_mode: overlay
    env:
      TOKEN: "global"
      OTHER: "value"

projects:
  work:
    env:
      TOKEN: "project"

# Result: TOKEN="project", OTHER="value"
```

### Replace Mode

Project environment completely replaces server environment.

```yaml
servers:
  github:
    merge_mode: replace
    env:
      TOKEN: "global"
      OTHER: "value"

projects:
  work:
    env:
      TOKEN: "project"

# Result: TOKEN="project" (OTHER is NOT included)
```

## Storage Locations

Assern keeps configuration and data separate from your project directories.

### Configuration

| Location | Purpose |
|----------|---------|
| `~/.valksor/assern/config.yaml` | Global configuration |
| `~/.valksor/assern/.env` | Global environment variables |
| `.assern/config.yaml` | Local project overrides |
| `.assern/.env` | Local environment variables |

### No Project Data Storage

Unlike some tools, Assern does not store task data or state. It is a stateless aggregator that only:
- Reads configuration files
- Spawns server processes
- Routes tool calls

All state is managed by the backend MCP servers themselves.

## Instance Sharing

When LLMs use MCP servers that themselves call other LLMs (creating nested chains), each nested invocation would normally spawn a new Assern instance with all its MCP servers. This creates a cascade of redundant instances.

Assern solves this with **instance sharing** - a single global instance serves all invocations.

### How It Works

```
Primary Instance (first invocation):
┌─────────────┐
│ LLM (Claude)│
└──────┬──────┘
       │ stdio
┌──────▼──────┐
│ Assern      │
│ - ServeStdio│
│ - SocketSrv │◄───┐
└──────┬──────┘    │
       │           │ Unix Socket
┌──────▼──────┐    │
│ Aggregator  │    │
│ (MCP Servers)    │
└─────────────┘    │
                   │
Secondary Instance:│
┌─────────────┐    │
│ Nested LLM  │    │
└──────┬──────┘    │
       │ stdio     │
┌──────▼──────┐    │
│ Assern      │────┘
│ (proxy mode)│
└─────────────┘
```

1. **First invocation** → becomes primary instance
   - Starts aggregator and all MCP servers normally
   - Creates Unix socket at `~/.valksor/assern/assern.sock`
   - Serves stdio (backward compatible) AND listens on socket

2. **Subsequent invocations** → detect socket, become proxy
   - Connect to primary via Unix socket
   - Bridge stdio ↔ socket (transparent to the calling LLM)
   - Share the single aggregator instance

### Socket Location

| File | Purpose |
|------|---------|
| `~/.valksor/assern/assern.sock` | Unix socket for instance communication |

### Disabling Instance Sharing

Set the environment variable to run isolated instances:

```bash
export ASSERN_NO_INSTANCE_SHARING=1
assern serve
```

This is useful for:
- Debugging instance-specific issues
- Testing configuration changes in isolation
- Running multiple independent aggregators intentionally

### Stale Socket Handling

If Assern crashes or is killed without cleanup, the socket file may remain. On next startup:
1. Assern attempts to connect to the socket
2. If connection fails (no process listening), the stale socket is removed
3. The new instance becomes primary

### CLI Commands Using Instance Sharing

The `assern list` command also leverages instance sharing for faster tool discovery:

```bash
# If assern serve is running, this returns instantly
assern list

# Force fresh discovery (ignores running instance)
assern list --fresh
```

When a running instance is detected:
1. `assern list` queries tools via the socket (instant response)
2. Output shows "(from running instance)" to indicate the source
3. If no instance is running, falls back to starting a fresh aggregator

This is useful when you want to quickly check available tools while working with an LLM that already has Assern running.
