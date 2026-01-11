# Quick Start

Get Assern up and running in 5 minutes.

> **Note:** Assern itself requires no runtime dependencies - just the downloaded binary. Individual MCP servers may have their own requirements (npx, API tokens, etc.) that you'll need to configure separately.

## Installation

### Option 1: Install Script (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/main/install.sh | bash
```

The script automatically:
- Detects your OS and architecture
- Downloads the correct binary
- Verifies checksums
- Installs to `~/.local/bin`

### Option 2: Go Install

```bash
go install github.com/valksor/go-assern/cmd/assern@latest
```

### Option 3: Manual Download

Download from [GitHub Releases](https://github.com/valksor/go-assern/releases) and place in your PATH.

## Verify Installation

```bash
assern version
```

## Initialize Configuration

```bash
assern config init
```

This creates:
- `~/.valksor/assern/mcp.json` - MCP server definitions (copy-paste from Claude Desktop)
- `~/.valksor/assern/config.yaml` - Projects, settings, and server overrides
- `~/.valksor/assern/.env` - Environment variables (optional)

## Configure Your First Server

Add servers to `~/.valksor/assern/mcp.json` (standard MCP format, copy-paste friendly):

```json
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
```

Optionally configure settings in `~/.valksor/assern/config.yaml`:

```yaml
settings:
  log_level: info
  timeout: 60s
```

Set your GitHub token:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

Or add it to `~/.valksor/assern/.env`:

```bash
GITHUB_TOKEN=ghp_your_token_here
```

## Verify Configuration

```bash
assern config validate
```

## List Available Tools

```bash
assern list
```

Output:
```
Servers:
  github (1 tools)
    - github_search_repositories

Tools:
  github_search_repositories (github)
    Search GitHub repositories
```

## Start the Server

```bash
assern serve
```

Assern is now running as an MCP server on stdio, ready to accept connections.

## Add a Project

Add a project section to `~/.valksor/assern/config.yaml` to use different tokens per project:

```yaml
projects:
  work:
    directories:
      - ~/work/*
    env:
      GITHUB_TOKEN: "${WORK_TOKEN}"

settings:
  log_level: info
  timeout: 60s
```

Now when you run `assern serve` from any directory matching `~/work/*`, it uses `WORK_TOKEN` instead of `GITHUB_TOKEN`.

## Next Steps

- [Integration Guide](integration.md) - Connect to Claude Desktop, Claude Code, or your IDE
- [Configuration Reference](configuration.md) - Full configuration options
- [Projects](projects.md) - Project detection and registry
- [Servers](servers.md) - Server definitions and tool filtering
