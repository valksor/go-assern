# Valksor Assern

> MCP Aggregator with Project-Level Configuration

Assern is a Go-based MCP (Model Context Protocol) aggregator that combines multiple MCP servers into a unified interface. Unlike simple aggregators, Assern supports project-level configuration, allowing different tokens, environment variables, and server setups per project.

## Key Features

- **MCP Aggregation**: Combine multiple MCP servers into one unified interface
- **Full MCP Support**: Aggregates tools, resources, and prompts from backend servers
- **Tool Prefixing**: All tools are prefixed with server name (`github_search`, `jira_get_ticket`)
- **Resource Prefixing**: Resources use custom URI scheme (`assern://github/file:///repo/README.md`)
- **Prompt Prefixing**: Prompts are prefixed like tools (`assistant_code_review`)
- **Project Contexts**: Different configurations per project (tokens, env vars, servers)
- **Directory Matching**: Auto-detect projects based on directory patterns with glob support
- **Environment Merging**: Configurable overlay or replace modes for environment variables
- **Tool Filtering**: Expose only allowed tools per server

## Why Assern?

When working with multiple projects that use different credentials or MCP server configurations, you typically need to:
- Manually switch environment variables
- Maintain separate configuration files
- Restart your MCP client

Assern solves this by automatically detecting which project you're in and applying the appropriate configuration.

## How It Works

```
Your MCP Client (Claude, etc.)
         │
         ▼
    ┌─────────┐
    │ Assern  │ ◄── Detects project from CWD
    └────┬────┘
         │
    ┌────┴────┬────────┬────────┐
    ▼         ▼        ▼        ▼
 GitHub    Jira   Filesystem   ...
 Server   Server    Server
```

1. Assern detects your project based on current directory
2. Loads project-specific configuration (env vars, allowed tools)
3. Spawns configured MCP servers with merged environment
4. Exposes all tools with server-prefixed names

## Quick Example

```yaml
# ~/.valksor/assern/config.yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"

projects:
  work:
    directories:
      - ~/work/*
    env:
      GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"

  personal:
    directories:
      - ~/repos/*
    env:
      GITHUB_TOKEN: "${PERSONAL_GITHUB_TOKEN}"
```

When you `cd ~/work/myproject` and run `assern serve`, it automatically uses `WORK_GITHUB_TOKEN`.

## Next Steps

- [Quick Start](quickstart.md) - Get up and running in 5 minutes
- [Configuration](configuration.md) - Full configuration reference
- [Projects](projects.md) - Project detection and registry
- [Servers](servers.md) - Server definitions and options
