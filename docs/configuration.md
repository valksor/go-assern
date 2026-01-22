# Configuration

Assern provides two ways to configure your MCP servers:

## Interactive CLI (Recommended)

The easiest way to manage MCP servers is through the interactive CLI:

```bash
assern mcp add              # Add a new server
assern mcp list             # List all servers
assern mcp edit <name>      # Edit existing server
assern mcp delete <name>    # Delete server(s)
```

The interactive prompts guide you through all configuration options and validate your inputs.

> **Note:** Commands also support **colon notation** for faster typing: `mcp:add`, `mcp:list`, etc.

## Manual Configuration

For advanced users or automation, you can directly edit configuration files. Assern uses a two-file configuration system for maximum flexibility:

1. **`mcp.json`** - Standard MCP server definitions (copy-paste from Claude Desktop)
2. **`config.yaml`** - Projects, settings, and server overrides

## Configuration Files

### Global Configuration

Location: `~/.valksor/assern/`

| File | Purpose |
|------|---------|
| `mcp.json` | MCP server definitions (standard format) |
| `config.yaml` | Projects, settings, server overrides |
| `.env` | Environment variables (optional) |

### Local Configuration

Location: `.assern/` (in any project directory)

| File | Purpose |
|------|---------|
| `mcp.json` | Project-specific MCP servers (optional) |
| `config.yaml` | Project-level overrides (optional) |

## MCP Server Configuration (`mcp.json`)

Standard MCP format - copy-paste from Claude Desktop or any MCP example:

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

## Transport Types

Assern supports multiple MCP transport types:

### Stdio Transport (Local Servers)

For local MCP servers that run as subprocesses:

```json
{
  "mcpServers": {
    "local-server": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"],
      "env": {
        "TOKEN": "${TOKEN}"
      }
    }
  }
}
```

### HTTP Transport (Remote Servers)

For remote MCP servers using the modern Streamable HTTP transport:

```json
{
  "mcpServers": {
    "context7": {
      "url": "https://mcp.context7.com/mcp"
    },
    "remote-api": {
      "url": "https://api.example.com/mcp",
      "transport": "http"
    }
  }
}
```

### SSE Transport (Legacy Remote Servers)

For older remote servers using Server-Sent Events:

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

For remote servers requiring API keys or Bearer tokens:

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

Headers can be used with both HTTP and SSE transports:

```json
{
  "mcpServers": {
    "sse-with-auth": {
      "url": "https://api.example.com/sse",
      "transport": "sse",
      "headers": {
        "Authorization": "Bearer ${TOKEN}"
      }
    }
  }
}
```

### OAuth HTTP Transport

For servers requiring OAuth 2.0 authentication with modern Streamable HTTP:

```json
{
  "mcpServers": {
    "enterprise": {
      "url": "https://api.enterprise.com/mcp",
      "transport": "oauth-http",
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

### OAuth SSE Transport

For servers requiring OAuth 2.0 with Server-Sent Events:

```json
{
  "mcpServers": {
    "oauth-sse-server": {
      "url": "https://api.example.com/sse",
      "transport": "oauth-sse",
      "oauth": {
        "clientId": "${CLIENT_ID}",
        "clientSecret": "${CLIENT_SECRET}",
        "authServerMetadataUrl": "https://auth.example.com/.well-known/oauth-authorization-server",
        "scopes": ["read", "write"]
      }
    }
  }
}
```

### OAuth with PKCE (Public Clients)

For public clients that cannot securely store client secrets, use PKCE (Proof Key for Code Exchange):

```json
{
  "mcpServers": {
    "public-oauth": {
      "url": "https://api.example.com/mcp",
      "transport": "oauth-http",
      "oauth": {
        "clientId": "${CLIENT_ID}",
        "redirectUri": "http://localhost:8080/callback",
        "authServerMetadataUrl": "https://auth.example.com/.well-known/oauth-authorization-server",
        "scopes": ["mcp:read"],
        "pkceEnabled": true
      }
    }
  }
}
```

### Working Directory for Stdio Servers

Specify a working directory for local subprocess servers:

```json
{
  "mcpServers": {
    "local-server": {
      "command": "node",
      "args": ["./server.js"],
      "workDir": "/home/user/mcp-server"
    }
  }
}
```

### Transport Detection

Assern automatically detects the transport type:

| Config | Transport |
|--------|-----------|
| `command` field present | stdio |
| `url` + `oauth` fields present | oauth-http (auto-detected) |
| `url` field present | http (default for remote) |
| `transport: "stdio"` explicit | stdio |
| `transport: "sse"` explicit | sse |
| `transport: "http"` explicit | http |
| `transport: "oauth-sse"` explicit | oauth-sse |
| `transport: "oauth-http"` explicit | oauth-http |

**OAuth Transport Fields:**

| Field | Description |
|-------|-------------|
| `clientId` | OAuth 2.0 client identifier |
| `clientSecret` | OAuth 2.0 client secret (optional for public clients with PKCE) |
| `redirectUri` | Redirect URI for OAuth flow |
| `scopes` | Array of requested OAuth scopes |
| `authServerMetadataUrl` | URL to OAuth 2.0 server metadata (RFC 9728) |
| `pkceEnabled` | Enable PKCE for public clients (boolean) |

### Mixed Configuration Example

Combine local, remote, and authenticated servers:

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
    "context7": {
      "url": "https://mcp.context7.com/mcp"
    },
    "api-with-key": {
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}"
      }
    },
    "enterprise-oauth": {
      "url": "https://enterprise.example.com/mcp",
      "oauth": {
        "clientId": "${OAUTH_CLIENT_ID}",
        "clientSecret": "${OAUTH_CLIENT_SECRET}",
        "authServerMetadataUrl": "https://auth.enterprise.com/.well-known/oauth-authorization-server",
        "scopes": ["mcp:read", "mcp:write"]
      }
    },
    "sequential-thinking": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-sequential-thinking"]
    }
  }
}
```

## Assern Configuration (`config.yaml`)

Contains project registry, settings, and server overrides:

```yaml
# Project registry
projects:
  # Project name
  work:
    # Directories that belong to this project (supports globs)
    directories:
      - ~/work/*
      - ~/projects/work-*

    # Environment variables for this project
    env:
      GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"

    # Server overrides for this project
    servers:
      github:
        # How project env merges with server env
        # "overlay" (default): project values override, others preserved
        # "replace": project values completely replace server env
        merge_mode: overlay
        env:
          GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"

      # Restrict tools for this project
      filesystem:
        allowed:
          - read_file
          - list_directory

      # Disable a server for this project
      slack:
        disabled: true

  personal:
    directories:
      - ~/repos/*
      - ~/side-projects/*
    env:
      GITHUB_TOKEN: "${PERSONAL_GITHUB_TOKEN}"

# Global settings
settings:
  # Log level: debug, info, warn, error
  log_level: info

  # Server connection timeout
  timeout: 60s

  # Output format for tool results: json or toon
  # TOON format reduces token usage by 40-60% for LLM consumption
  output_format: json
```

## Local Configuration (`.assern/config.yaml`)

```yaml
# Reference to global project (optional)
project: work

# Local environment overrides
env:
  GITHUB_TOKEN: "${REPO_SPECIFIC_TOKEN}"

# Local server overrides
servers:
  github:
    env:
      GITHUB_TOKEN: "${REPO_SPECIFIC_TOKEN}"
```

## Local MCP Servers (`.assern/mcp.json`)

Add project-specific servers:

```json
{
  "mcpServers": {
    "jira": {
      "command": "npx",
      "args": ["-y", "mcp-server-jira"],
      "env": {
        "JIRA_TOKEN": "${ACME_JIRA_TOKEN}"
      }
    }
  }
}
```

## Configuration Resolution

Assern resolves configuration in this order (highest priority first):

1. Local MCP servers (`.assern/mcp.json`)
2. Local config overrides (`.assern/config.yaml`)
3. Project definition in global `config.yaml`
4. Global MCP servers (`~/.valksor/assern/mcp.json`)
5. Global env (`~/.valksor/assern/.env`)
6. System environment variables

> **Note:** Assern does NOT read `.env` files from project directories. All environment variables come from the global `.env` file or system environment.

## Environment Variable Expansion

Use `${VAR}` or `$VAR` syntax for variable expansion:

```json
{
  "env": {
    "TOKEN": "${MY_TOKEN}"
  }
}
```

```yaml
env:
  TOKEN: "${MY_TOKEN}"
  API_URL: "$API_URL"
```

## Merge Modes

### Overlay Mode (Default)

Project environment values override server values, but non-overridden server values are preserved.

```yaml
# config.yaml
projects:
  work:
    servers:
      github:
        merge_mode: overlay
        env:
          TOKEN: "project"

# If mcp.json has: TOKEN: "global", OTHER: "value"
# Result for work project:
# TOKEN: "project"
# OTHER: "value"
```

### Replace Mode

Project environment completely replaces server environment.

```yaml
# config.yaml
projects:
  work:
    servers:
      github:
        merge_mode: replace
        env:
          TOKEN: "project"

# If mcp.json has: TOKEN: "global", OTHER: "value"
# Result for work project:
# TOKEN: "project"
# (OTHER is not included)
```

## Output Format (TOON)

Assern supports **TOON** (Token-Oriented Object Notation) format for tool results, which reduces token usage by 40-60% when communicating with LLMs.

### Enabling TOON Format

TOON format can be enabled via:

**1. Configuration file:**
```yaml
settings:
  output_format: toon
```

**2. Environment variable:**
```bash
export ASSERN_OUTPUT_FORMAT=toon
assern serve
```

**3. CLI flag:**
```bash
assern --output-format toon serve
```

### Priority Order

CLI flag > Environment variable > Config file > Default (JSON)

### When to Use TOON

- **Use TOON** when tool results are consumed by LLMs directly (reduces token costs)
- **Use JSON** for compatibility with Claude Desktop and other MCP clients

### TOON Format Example

**JSON output:**
```json
{
  "content": [
    {"type": "text", "text": "First result"},
    {"type": "text", "text": "Second result"}
  ],
  "metadata": {"format": "json", "contentCount": 2}
}
```

**TOON output:**
```toon
content[2]{type,text}:
text,First result
text,Second result
metadata:
format:json
contentCount:2
```

## Environment Variables

Assern supports several environment variables:

| Variable | Description |
|----------|-------------|
| `ASSERN_OUTPUT_FORMAT` | Output format: `json` or `toon` |
| `GITHUB_TOKEN` | Example token for GitHub MCP server |
| `SLACK_TOKEN` | Example token for Slack MCP server |

Environment variables can be defined in:
- Global: `~/.valksor/assern/.env`
- System environment (for shell expansion in config)

## Validation

Validate your configuration:

```bash
assern config validate
```

This checks:
- JSON/YAML syntax
- Required fields
- Server command existence
- Directory patterns

## Example Configurations

### Single Server

**mcp.json:**
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

**config.yaml:**
```yaml
settings:
  log_level: info
```

### Multiple Servers

**mcp.json:**
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

**config.yaml:**
```yaml
settings:
  log_level: info
  timeout: 120s
```

### Multi-Project Setup

**mcp.json:**
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

**config.yaml:**
```yaml
projects:
  acme-corp:
    directories:
      - ~/work/acme/*
    env:
      GITHUB_TOKEN: "${ACME_TOKEN}"
    servers:
      filesystem:
        allowed:
          - read_file
          - list_directory

  personal:
    directories:
      - ~/repos/*
    env:
      GITHUB_TOKEN: "${PERSONAL_TOKEN}"

settings:
  log_level: info
```

**acme-project/.assern/mcp.json:** (optional project-specific server)
```json
{
  "mcpServers": {
    "jira": {
      "command": "npx",
      "args": ["-y", "mcp-server-jira"],
      "env": {
        "JIRA_TOKEN": "${ACME_JIRA_TOKEN}"
      }
    }
  }
}
```

### Enterprise OAuth Setup

**mcp.json:**
```json
{
  "mcpServers": {
    "enterprise-api": {
      "url": "https://api.enterprise.com/mcp",
      "oauth": {
        "clientId": "${ENTERPRISE_CLIENT_ID}",
        "clientSecret": "${ENTERPRISE_CLIENT_SECRET}",
        "authServerMetadataUrl": "https://auth.enterprise.com/.well-known/oauth-authorization-server",
        "scopes": ["mcp:read", "mcp:write", "mcp:admin"]
      }
    },
    "internal-service": {
      "url": "https://internal.enterprise.com/mcp",
      "headers": {
        "Authorization": "Bearer ${INTERNAL_API_TOKEN}",
        "X-Tenant-ID": "${TENANT_ID}"
      }
    }
  }
}
```

**config.yaml:**
```yaml
projects:
  enterprise:
    directories:
      - ~/work/enterprise/*
    servers:
      enterprise-api:
        merge_mode: overlay
      internal-service:
        headers:
          X-Environment: "production"

settings:
  log_level: info
  timeout: 120s
```

### Public OAuth Client with PKCE

For browser-based or mobile applications that cannot securely store secrets:

**mcp.json:**
```json
{
  "mcpServers": {
    "public-api": {
      "url": "https://api.example.com/mcp",
      "transport": "oauth-http",
      "oauth": {
        "clientId": "${PUBLIC_CLIENT_ID}",
        "redirectUri": "http://localhost:8080/callback",
        "authServerMetadataUrl": "https://auth.example.com/.well-known/oauth-authorization-server",
        "scopes": ["read", "write"],
        "pkceEnabled": true
      }
    }
  }
}
```
