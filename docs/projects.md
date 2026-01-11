# Projects

Projects allow you to maintain different MCP configurations based on the directory you're working in.

## How Project Detection Works

When you run `assern serve`, Assern detects your project in this order:

1. **Local Config**: Check for `.assern/config.yaml` in current or parent directories
2. **Registry Match**: Match current directory against `projects[*].directories` patterns
3. **Auto-Detect**: Use directory basename as project name (e.g., `go-myproject` from `/home/user/repos/go-myproject`)
4. **Explicit Flag**: Use `--project` flag to override any detection

```
/home/user/work/acme/my-repo/
         │
         ├── Check: .assern/config.yaml exists? → Use local config
         │
         └── No local config:
             ├── Match ~/work/acme/* in global config? → Use "acme" project
             └── No match → Auto-detect as "my-repo" from directory name
```

> **Note:** Auto-detection means you can use Assern in any directory without configuration. The directory name becomes the project name.

## Project Registry

Define projects in your global config (`~/.valksor/assern/config.yaml`):

```yaml
projects:
  work:
    directories:
      - ~/work/*
      - ~/projects/work-*
    env:
      GITHUB_TOKEN: "${WORK_TOKEN}"

  personal:
    directories:
      - ~/repos/*
      - ~/side-projects/*
    env:
      GITHUB_TOKEN: "${PERSONAL_TOKEN}"
```

## Directory Patterns

### Exact Path

```yaml
directories:
  - ~/work/myproject
```

Matches only `/home/user/work/myproject`.

### Single Wildcard (`*`)

```yaml
directories:
  - ~/work/*
```

Matches `/home/user/work/project1`, `/home/user/work/project2`, etc.
Does NOT match `/home/user/work/org/project`.

### Double Wildcard (`**`)

```yaml
directories:
  - ~/work/**
```

Matches any depth under `~/work/`:
- `~/work/project`
- `~/work/org/project`
- `~/work/org/team/project`

### Tilde Expansion

`~` is expanded to your home directory:

```yaml
directories:
  - ~/work/*        # /home/user/work/*
  - ~/.config/*     # /home/user/.config/*
```

## Local Project Configuration

Create `.assern/config.yaml` in any directory for local overrides:

```yaml
# Link to global project definition
project: work

# Override environment variables
env:
  GITHUB_TOKEN: "${REPO_SPECIFIC_TOKEN}"

# Override servers
servers:
  github:
    env:
      GITHUB_TOKEN: "${REPO_SPECIFIC_TOKEN}"
```

### Without Project Reference

If you don't reference a global project, only local settings apply:

```yaml
# No project reference - uses only global server definitions
env:
  GITHUB_TOKEN: "${MY_TOKEN}"
```

### Local-Only Servers

Add servers that only exist for this directory:

```yaml
project: work

servers:
  local-db:
    command: my-local-db-server
    env:
      DB_PATH: ./data/local.db
```

## Auto-Registration

When Assern detects a new `.assern/config.yaml`, it can auto-register the directory to your global config for future detection without the local config file.

This happens when:
1. Local config exists with a `project` reference
2. The directory isn't already in the global project registry

## Project Environment

Each project can define environment variables that override global settings:

```yaml
projects:
  acme:
    directories:
      - ~/work/acme/*
    env:
      GITHUB_TOKEN: "${ACME_TOKEN}"
      ORG_NAME: "acme-corp"
```

### Environment Inheritance

```
System Environment
       │
       ▼
Global .env (~/.valksor/assern/.env)
       │
       ▼
Project env (from projects.*.env)
       │
       ▼
Local .env (.assern/.env)
       │
       ▼
Local config env (.assern/config.yaml env)
       │
       ▼
Final Environment (passed to servers)
```

## Project-Specific Servers

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
      # Override github settings
      github:
        env:
          GITHUB_TOKEN: "${ACME_TOKEN}"

      # Add jira only for acme projects
      jira:
        command: npx
        args: ["-y", "mcp-server-jira"]
        env:
          JIRA_TOKEN: "${ACME_JIRA_TOKEN}"
```

When in an acme project:
- `github_*` tools use `ACME_TOKEN`
- `jira_*` tools are available

When in other projects:
- `github_*` tools use default token
- `jira_*` tools are NOT available

## Debugging Project Detection

Use `assern list` to see which project is detected:

```bash
cd ~/work/acme/my-repo
assern list
```

Output:
```
Project: acme (detected from directory match)

Servers:
  github (5 tools)
  jira (3 tools)
```

## Examples

### Multi-Organization Setup

```yaml
projects:
  acme:
    directories:
      - ~/work/acme/*
    env:
      GITHUB_TOKEN: "${ACME_GITHUB_TOKEN}"
      JIRA_URL: "https://acme.atlassian.net"
    servers:
      jira:
        command: npx
        args: ["-y", "mcp-server-jira"]
        env:
          JIRA_TOKEN: "${ACME_JIRA_TOKEN}"

  globex:
    directories:
      - ~/work/globex/*
    env:
      GITHUB_TOKEN: "${GLOBEX_GITHUB_TOKEN}"
      JIRA_URL: "https://globex.atlassian.net"
    servers:
      jira:
        command: npx
        args: ["-y", "mcp-server-jira"]
        env:
          JIRA_TOKEN: "${GLOBEX_JIRA_TOKEN}"
```

### Personal vs Work

```yaml
projects:
  work:
    directories:
      - ~/work/*
    env:
      GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"

  personal:
    directories:
      - ~/repos/*
      - ~/side-projects/*
    env:
      GITHUB_TOKEN: "${PERSONAL_GITHUB_TOKEN}"
```
