# Configuration

Assern uses a two-file configuration system for maximum flexibility:

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
