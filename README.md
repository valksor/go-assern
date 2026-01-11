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

- **Unified Interface** - Single MCP client connection to all your servers
- **Tool Prefixing** - All tools are namespaced by server name (`github_search`, `jira_get_ticket`) preventing conflicts
- **Project Contexts** - Different configurations per project (tokens, env vars, enabled servers)
- **Directory Matching** - Auto-detect projects based on directory patterns
- **Environment Merging** - Configurable overlay or replace modes for env variables
- **Tool Filtering** - Expose only allowed tools per server for security and simplicity

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  MCP Client (Claude, etc.)                                  │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────────────────────────────────────┐               │
│  │           Assern Aggregator             │               │
│  │  • Tool prefixing                       │               │
│  │  • Project detection                   │               │
│  │  • Config merging                      │               │
│  └─────────────────────────────────────────┘               │
│       │                                                     │
│       ├─────────────────┬─────────────┬───────────────┐    │
│       ▼                 ▼             ▼               ▼    │
│  ┌─────────┐      ┌─────────┐   ┌─────────┐    ┌─────────┐ │
│  │ GitHub  │      │ Jira    │   │  Slack  │    │    ...  │ │
│  │ Server  │      │ Server  │   │ Server  │    │ Server  │ │
│  └─────────┘      └─────────┘   └─────────┘    └─────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

1. **Configure** - Define MCP servers and projects in `~/.valksor/assern/config.yaml`
2. **Detect** - Assern auto-detects your project from current directory
3. **Aggregate** - All enabled servers are spawned and tools are prefixed
4. **Route** - Tool calls are routed to the appropriate backend server

## Requirements

- **Go 1.25+** (only if building from source - not needed for pre-built binaries)

> **Note:** Individual MCP servers have their own requirements (npx/Node.js, API tokens, etc.). These are your responsibility to configure - Assern simply spawns the commands you define.

## Features

- **MCP Aggregation**: Combine multiple MCP servers into one unified interface
- **Tool Prefixing**: All tools are prefixed with server name (`github_search`, `jira_get_ticket`)
- **Project Contexts**: Different configurations per project (tokens, env vars, servers)
- **Directory Matching**: Auto-detect projects based on directory patterns
- **Environment Merging**: Configurable overlay or replace modes for env variables
- **Tool Filtering**: Expose only allowed tools per server

## Installation

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/main/install.sh | bash
```

### Go Install

```bash
go install github.com/valksor/go-assern/cmd/assern@latest
```

### Manual Download

Download the latest release for your platform:

| Platform | Architecture | Binary Name |
|----------|--------------|--------------|
| Linux | AMD64 | `assern-linux-amd64` |
| Linux | ARM64 | `assern-linux-arm64` |
| macOS | AMD64 (Intel) | `assern-darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `assern-darwin-arm64` |

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

2. Add servers to `~/.valksor/assern/mcp.json` (copy-paste from Claude Desktop):

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
- [Integration](https://assern.valksor.com/docs/#/integration) - Claude Desktop, Claude Code, IDEs
- [Configuration](https://assern.valksor.com/docs/#/configuration)
- [Projects](https://assern.valksor.com/docs/#/projects)
- [Servers](https://assern.valksor.com/docs/#/servers)
- [Concepts](https://assern.valksor.com/docs/#/concepts)
- [Troubleshooting](https://assern.valksor.com/docs/#/troubleshooting)

## CLI Commands

| Command | Description |
|---------|-------------|
| `assern serve` | Start MCP server on stdio |
| `assern list` | List available servers and tools |
| `assern config init` | Initialize configuration file |
| `assern config validate` | Validate configuration syntax |
| `assern version` | Show version information |

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
3. Use explicit `--project` flag if needed

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

MIT License - see [LICENSE](LICENSE) for details.
