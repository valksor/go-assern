# Servers

Servers are the MCP backends that Assern aggregates into a unified interface.

## Server Definition

Define servers in your global config (`~/.valksor/assern/config.yaml`):

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
```

## Configuration Options

### command (required)

The command to spawn the server process:

```yaml
servers:
  github:
    command: npx
```

### args (optional)

Arguments passed to the command:

```yaml
servers:
  github:
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
```

### env (optional)

Environment variables for the server:

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
      GITHUB_API_URL: "https://api.github.com"
```

### allowed (optional)

Whitelist of tools to expose. If empty, all tools are exposed:

```yaml
servers:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
    allowed:
      - read_file
      - list_directory
      # write_file, delete_file, etc. are NOT exposed
```

### disabled (optional)

Temporarily disable a server:

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    disabled: true  # Server won't be started
```

### merge_mode (optional)

How project environment merges with server environment:

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    merge_mode: overlay  # or "replace"
```

See [Configuration - Merge Modes](configuration.md#merge-modes) for details.

## Tool Prefixing

All tools from backend servers are prefixed with the server name:

| Server | Original Tool | Exposed Tool |
|--------|--------------|--------------|
| github | search_repositories | github_search_repositories |
| github | get_repository | github_get_repository |
| jira | get_ticket | jira_get_ticket |
| filesystem | read_file | filesystem_read_file |

This prevents naming conflicts when multiple servers have similar tools.

### Naming Rules

- Server name: used as-is (lowercase recommended)
- Dashes converted to underscores: `my-server` + `my-tool` = `my_server_my_tool`
- Underscores preserved: `my_server` + `my_tool` = `my_server_my_tool`

## Listing Servers and Tools

```bash
assern list
```

Output:
```
Project: work

Servers:
  github (5 tools)
    - github_search_repositories
    - github_get_repository
    - github_list_issues
    - github_create_issue
    - github_get_user

  filesystem (2 tools)
    - filesystem_read_file
    - filesystem_list_directory
```

## Server Lifecycle

1. **Startup**: When `assern serve` runs, all enabled servers are spawned
2. **Discovery**: Assern lists tools from each server
3. **Registration**: Tools are registered with prefixed names
4. **Routing**: Tool calls are routed to the appropriate backend server
5. **Shutdown**: On exit, all servers are gracefully terminated

## Error Handling

### Server Spawn Failure

If a server fails to spawn, Assern logs the error and continues with other servers:

```
ERROR Failed to start server: github: exec: "npx": not found
INFO  Started server: filesystem (2 tools)
```

### Tool Call Failure

Tool call errors are returned to the MCP client:

```json
{
  "error": {
    "code": -32603,
    "message": "server 'github' returned error: rate limit exceeded"
  }
}
```

## Remote Servers

Remote MCP servers use HTTP or SSE transports instead of spawning local processes.

### HTTP Transport

```json
{
  "mcpServers": {
    "remote-api": {
      "url": "https://api.example.com/mcp"
    }
  }
}
```

### SSE Transport (Legacy)

```json
{
  "mcpServers": {
    "legacy-server": {
      "url": "https://old-api.example.com/sse",
      "transport": "sse"
    }
  }
}
```

### HTTP Headers Authentication

For servers requiring API keys or Bearer tokens:

```json
{
  "mcpServers": {
    "api-server": {
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}",
        "X-API-Key": "${API_KEY}"
      }
    }
  }
}
```

### OAuth 2.0 Authentication

For servers requiring OAuth authentication:

```json
{
  "mcpServers": {
    "enterprise": {
      "url": "https://api.enterprise.com/mcp",
      "oauth": {
        "clientId": "${CLIENT_ID}",
        "clientSecret": "${CLIENT_SECRET}",
        "authServerMetadataUrl": "https://auth.enterprise.com/.well-known/oauth-authorization-server",
        "scopes": ["mcp:read", "mcp:write"]
      }
    }
  }
}
```

For public clients using PKCE:

```json
{
  "mcpServers": {
    "public-client": {
      "url": "https://api.example.com/mcp",
      "transport": "oauth-http",
      "oauth": {
        "clientId": "${CLIENT_ID}",
        "redirectUri": "http://localhost:8080/callback",
        "authServerMetadataUrl": "https://auth.example.com/.well-known/oauth-authorization-server",
        "pkceEnabled": true
      }
    }
  }
}
```

See [Configuration - Transport Types](configuration.md#transport-types) for full details.

## Common Servers

### GitHub

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

### Filesystem

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"]
    }
  }
}
```

### Slack

```json
{
  "mcpServers": {
    "slack": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-server-slack"],
      "env": {
        "SLACK_TOKEN": "${SLACK_TOKEN}"
      }
    }
  }
}
```

### Context7 (Remote)

```json
{
  "mcpServers": {
    "context7": {
      "url": "https://mcp.context7.com/mcp"
    }
  }
}
```

### Custom Server

```json
{
  "mcpServers": {
    "myserver": {
      "command": "/path/to/my-mcp-server",
      "args": ["--config", "/path/to/config.json"],
      "env": {
        "MY_API_KEY": "${MY_API_KEY}"
      }
    }
  }
}
```

## Project Server Overrides

Override server settings per project:

```yaml
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
    servers:
      github:
        # Override just the env
        env:
          GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"
```

The `command` and `args` from the global definition are preserved; only `env` is overridden.

## Adding Project-Only Servers

Add servers that only exist for specific projects:

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]

projects:
  acme:
    directories:
      - ~/work/acme/*
    servers:
      jira:
        command: npx
        args: ["-y", "mcp-server-jira"]
        env:
          JIRA_TOKEN: "${ACME_JIRA_TOKEN}"
          JIRA_URL: "https://acme.atlassian.net"
```

The `jira` server only appears when in an acme project directory.

## Disabling Servers Per Project

Disable a server for specific projects:

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]

projects:
  restricted:
    directories:
      - ~/restricted/*
    servers:
      github:
        disabled: true
```
