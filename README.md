# Valksor Assern

[![valksor](https://badgen.net/static/org/valksor/green)](https://github.com/valksor)
[![BSD-3-Clause](https://img.shields.io/badge/BSD--3--Clause-green?style=flat)](https://github.com/valksor/go-assern/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/valksor/go-assern.svg?style=flat)](https://github.com/valksor/go-assern/releases/latest)
[![GitHub last commit](https://img.shields.io/github/last-commit/valksor/go-assern.svg?style=flat)]()

[![Coverage Status](https://coveralls.io/repos/github/valksor/go-assern/badge.svg?branch=master)](https://coveralls.io/github/valksor/go-assern?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/valksor/go-assern)](https://goreportcard.com/report/github.com/valksor/go-assern)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/valksor/go-assern)

---

A Go-based MCP (Model Context Protocol) aggregator with project-level configuration support. Assern combines multiple MCP servers into a unified interface while allowing different configurations per project.

## Why Assern?

Assern solves the problem of managing multiple MCP servers across different projects and contexts.

**Key benefits:**

- **Unified Interface** – Single MCP client connection to all your servers
- **Full MCP Protocol** – Aggregates tools, resources, and prompts from all backends
- **Tool Prefixing** – All tools are namespaced by server name (`github_search`, `jira_get_ticket`) preventing conflicts
- **Resource Prefixing** – Resources use custom URI scheme (`assern://server/original-uri`)
- **Prompt Prefixing** – Prompts are namespaced like tools (`assistant_code_review`)
- **Project Contexts** – Different configurations per project (tokens, env vars, enabled servers)
- **TOON Format** – Token-optimized output format for LLM consumption (40-60% token reduction)
- **Directory Matching** - Auto-detect projects based on directory patterns
- **Environment Merging** - Configurable overlay or replace modes for env variables
- **Tool Filtering** – Expose only allowed tools per server for security and simplicity
- **Instance Sharing** – Prevents cascade spawning when LLMs call nested LLMs via shared Unix socket

## Client Compatibility

Assern speaks standard stdio MCP protocol, so it works with **any MCP-compatible client**:

| Client | Status |
|--------|--------|
| Claude Code | ✅ Primary target |
| Claude Desktop | ✅ Supported |
| Cursor | ✅ Compatible |
| Windsurf | ✅ Compatible |
| Cline | ✅ Compatible |
| Continue.dev | ✅ Compatible |
| GitHub Copilot | ✅ Compatible |
| Roo Code | ✅ Compatible |
| Mintlify | ✅ Compatible |

> **Note:** If your client speaks stdio MCP, it works with Assern – no special integration needed.

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  MCP Client (Claude, etc.)                                  │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────────────────────────────────────┐                │
│  │           Assern Aggregator             │                │
│  │  • Tool/Resource/Prompt prefixing       │                │
│  │  • Project detection                    │                │
│  │  • Config merging                       │                │
│  └─────────────────────────────────────────┘                │
│       │                                                     │
│       ├─────────────────┬─────────────┬───────────────┐     │
│       ▼                 ▼             ▼               ▼     │
│  ┌─────────┐      ┌─────────┐   ┌─────────┐    ┌─────────┐  │
│  │ GitHub  │      │ Jira    │   │  Slack  │    │    ...  │  │
│  │ Server  │      │ Server  │   │ Server  │    │ Server  │  │
│  └─────────┘      └─────────┘   └─────────┘    └─────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

1. **Configure** – Define MCP servers and projects in `~/.valksor/assern/config.yaml`
2. **Detect** – Assern auto-detects your project from the current directory
3. **Aggregate** – All enabled servers are spawned; tools, resources, and prompts are prefixed
4. **Route** – Requests are routed to the appropriate backend server

## Requirements

- **Go 1.25+** (only if building from source – not needed for pre-built binaries)

> **Note:** Individual MCP servers have their own requirements (npx/Node.js, API tokens, etc.). These are your responsibility to configure – Assern simply spawns the commands you define.

## Features

- **MCP Aggregation**: Combine multiple MCP servers into one unified interface
- **Full MCP Protocol**: Aggregates tools, resources, and prompts from backend servers
- **Multi-Transport**: Support for stdio (local), HTTP, SSE, and OAuth-authenticated (remote) MCP servers
- **Authentication**: HTTP headers (API keys, Bearer tokens) and OAuth 2.0 with PKCE support
- **Tool Prefixing**: All tools are prefixed with server name (`github_search`, `jira_get_ticket`)
- **Resource Prefixing**: Resources use custom URI scheme (`assern://github/file:///repo/README.md`)
- **Prompt Prefixing**: Prompts are prefixed like tools (`assistant_code_review`)
- **Project Contexts**: Different configurations per project (tokens, env vars, servers)
- **Directory Matching**: Auto-detect projects based on directory patterns
- **Environment Merging**: Configurable overlay or replace modes for env variables
- **Tool Filtering**: Expose only allowed tools per server
- **Instance Sharing**: Prevents cascade spawning when nested LLMs launch assern

## Instance Sharing

When LLMs use MCP servers that themselves call LLMs (creating nested chains), each nested invocation would normally spawn a new assern instance with all its MCP servers. This creates a cascade of redundant instances.

Assern solves this with **instance sharing**:

```
Primary Instance (first invocation):
┌─────────────┐
│ LLM (Claude)│
└──────┬──────┘
       │ stdio
┌──────▼──────┐
│assern       │
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
│assern       │────┘
│ (proxy mode)│
└─────────────┘
```

- **First invocation** → becomes primary instance (starts aggregator + socket server)
- **Subsequent invocations** → detect socket, run as proxy to primary instance
- **Socket location**: `~/.valksor/assern/assern.sock`
- **Disable**: Set `ASSERN_NO_INSTANCE_SHARING=1` environment variable

## Installation

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/master/install.sh | bash
```

The install script automatically:
- Detects your OS and architecture (Linux/macOS, AMD64/ARM64)
- Finds the best install location (`~/.local/bin`, `~/bin`, or `/usr/local/bin`)
- Verifies checksums (and Cosign signatures if available)
- Provides shell-specific PATH instructions

**Install options:**

```bash
# Install specific version
curl -fsSL ... | bash -s -- -v v1.0.0

# Install nightly build
curl -fsSL ... | bash -s -- --nightly

# Show help
bash install.sh --help
```

### Go Install

```bash
go install github.com/valksor/go-assern/cmd/assern@latest
```

### Docker

Build and run with Docker Compose:

```bash
# Create config directory
mkdir -p ~/.valksor/assern

# Add your mcp.json configuration
cat > ~/.valksor/assern/mcp.json << 'EOF'
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
EOF

# Start with docker-compose
docker-compose up -d
```

Or build and run manually:

```bash
docker build -t assern .
docker run -d --name assern-mcp \
  -v ~/.valksor/assern:/home/assern/.valksor/assern:ro \
  assern
```

See [docs/docker.md](docs/docker.md) for complete Docker documentation.

### Manual Download

Download the latest release for your platform:

| Platform | Architecture          | Binary Name           |
|----------|-----------------------|-----------------------|
| Linux    | AMD64                 | `assern-linux-amd64`  |
| Linux    | ARM64                 | `assern-linux-arm64`  |
| macOS    | AMD64 (Intel)         | `assern-darwin-amd64` |
| macOS    | ARM64 (Apple Silicon) | `assern-darwin-arm64` |

Download from [GitHub Releases](https://github.com/valksor/go-assern/releases):

```bash
# Download and install (example for macOS ARM64)
curl -L https://github.com/valksor/go-assern/releases/latest/download/assern-darwin-arm64 -o assern
chmod +x assern
sudo mv assern /usr/local/bin/

# Verify
assern version
```

## Quick Start

1. Initialize configuration:

```bash
assern config init
```

2. Add MCP servers interactively OR edit config files manually:

**Option A: Interactive CLI (Recommended)**
```bash
# Add a server with guided prompts
assern mcp add

# List all configured servers
assern mcp list

# Edit an existing server
assern mcp edit github

# Delete servers
assern mcp delete
```

**Option B: Manual Configuration**

Add servers to `~/.valksor/assern/mcp.json` (local or remote):

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"]
    },
    "context7": {
      "url": "https://mcp.context7.com/mcp"
    },
    "api-with-key": {
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}"
      }
    }
  }
}
```

3. Configure projects in `~/.valksor/assern/config.yaml`:

```yaml
projects:
  work:
    directories:
      - ~/work/*
    env:
      GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"
    servers:
      filesystem:
        allowed:
          - read_file
          - list_directory

settings:
  log_level: info
  timeout: 60s
```

4. Start the aggregator:

```bash
cd ~/work/myproject
assern serve
```

## Documentation

Full documentation available at [assern.valksor.com/docs](https://assern.valksor.com/docs)

- [Quick Start](https://assern.valksor.com/docs/#/quickstart)
- [Docker Setup](docs/docker.md) - Run Assern in containers
- [Integration](https://assern.valksor.com/docs/#/integration) - Claude Desktop, Claude Code, IDEs
- [Configuration](https://assern.valksor.com/docs/#/configuration)
- [Projects](https://assern.valksor.com/docs/#/projects)
- [Servers](https://assern.valksor.com/docs/#/servers)
- [Concepts](https://assern.valksor.com/docs/#/concepts)
- [Troubleshooting](https://assern.valksor.com/docs/#/troubleshooting)

## CLI Commands

| Command                      | Description                                              |
|------------------------------|----------------------------------------------------------|
| `assern serve`               | Start MCP aggregator on stdio (default command)          |
| `assern list`                | List available servers and tools (uses running instance if available) |
| `assern list --fresh`        | List tools with fresh discovery (ignores running instance) |
| `assern mcp add`             | Interactively add a new MCP server configuration          |
| `assern mcp edit [name]`     | Interactively edit an existing MCP server                 |
| `assern mcp delete [name]`   | Interactively delete MCP server(s)                        |
| `assern mcp list`            | List all configured MCP servers                          |
| `assern config init`         | Create ~/.valksor/assern/ with mcp.json and config.yaml  |
| `assern config init --force` | Reinitialize configuration (overwrites existing files)   |
| `assern config validate`     | Validate configuration syntax                            |
| `assern version`             | Show version information                                 |

> **Note:** All commands support **colon notation** for faster typing (e.g., `mcp:add`, `config:init`, `list:servers`).

### Global Flags

| Flag                    | Description                                                     |
|-------------------------|-----------------------------------------------------------------|
| `--output-format`       | Output format for tool results: `json` or `toon`                |
| `--project`             | Explicit project name (overrides auto-detection)                |
| `--config`              | Path to config.yaml (default: ~/.valksor/assern/config.yaml)    |
| `-v, --verbose`         | Enable debug logging                                            |
| `-q, --quiet`           | Suppress progress and info messages                             |

## Configuration

### Global MCP Servers (`~/.valksor/assern/mcp.json`)

Standard MCP format - copy-paste from Claude Desktop or any MCP example.

### Global Config (`~/.valksor/assern/config.yaml`)

Defines project registry with directory patterns, settings, and server overrides.

### Local MCP Servers (`.assern/mcp.json`)

Optional project-specific MCP servers.

### Local Config (`.assern/config.yaml`)

Optional project-level overrides in any directory.

### Environment Variables

- Global: `~/.valksor/assern/.env`
- Project: `.assern/.env`

Variables can use `${VAR}` syntax for expansion.

## Project Detection

1. Check for `.assern/config.yaml` in current or parent directories
2. Match against `projects[*].directories` patterns in global config
3. Auto-detect from directory name (e.g., `my-repo` from `/path/to/my-repo`)
4. Use explicit `--project` flag to override

> Works in any directory without configuration - the directory name becomes the project name.

## Development

```bash
make build        # Build binary to ./build/assern
make install      # Install to $GOPATH/bin
make test         # Run tests with coverage
make coverage     # Generate coverage report
make quality      # Run golangci-lint + govulncheck
make fmt          # Format code (gofmt, goimports, gofumpt)
make tidy         # Tidy dependencies
make hooks        # Enable versioned git hooks
make lefthook     # Install pre-commit hooks (auto-format + lint)
```

**CI/CD**: PRs trigger lint/test/build via GitHub Actions. Releases use GoReleaser.

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

MIT License – see [LICENSE](LICENSE) for details.
