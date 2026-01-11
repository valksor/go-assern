# Guides

Step-by-step tutorials for common Assern workflows.

## Multi-Organization Setup

Configure different GitHub tokens and servers for multiple organizations.

### Scenario

You work on:
- **Personal projects** using your personal GitHub account
- **Company A** projects using their GitHub Enterprise instance
- **Company B** projects using their own setup

### Configuration

```yaml
# ~/.valksor/assern/config.yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${PERSONAL_GITHUB_TOKEN}"
      GITHUB_API_URL: "https://api.github.com"

projects:
  company-a:
    directories:
      - ~/work/company-a/*
    env:
      GITHUB_TOKEN: "${COMPANY_A_GITHUB_TOKEN}"
      GITHUB_API_URL: "https://github.company-a.com/api"
    servers:
      jira:
        command: npx
        args: ["-y", "@modelcontextprotocol/server-jira"]
        env:
          JIRA_URL: "https://company-a.atlassian.net"
          JIRA_TOKEN: "${COMPANY_A_JIRA_TOKEN}"

  company-b:
    directories:
      - ~/work/company-b/*
    env:
      GITHUB_TOKEN: "${COMPANY_B_GITHUB_TOKEN}"
      GITHUB_API_URL: "https://github.company-b.com/api"
```

### Usage

```bash
# Working on personal project
cd ~/repos/my-side-project
assern serve  # Uses personal GitHub

# Working on Company A project
cd ~/work/company-a/backend
assern serve  # Uses Company A GitHub + Jira

# Working on Company B project
cd ~/work/company-b/frontend
assern serve  # Uses Company B GitHub
```

---

## Personal vs Work Projects

Separate your personal and work configurations with automatic detection.

### Setup

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
      - ~/projects/company-*
    env:
      GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"
    servers:
      slack:
        command: npx
        args: ["-y", "@anthropic/mcp-server-slack"]
        env:
          SLACK_TOKEN: "${WORK_SLACK_TOKEN}"

  personal:
    directories:
      - ~/repos/*
      - ~/side-projects/*
    env:
      GITHUB_TOKEN: "${PERSONAL_GITHUB_TOKEN}"
```

### Environment Setup

```bash
# ~/.valksor/assern/.env
GITHUB_TOKEN=ghp_default_personal_token
PERSONAL_GITHUB_TOKEN=ghp_my_personal_token
WORK_GITHUB_TOKEN=ghp_work_provided_token
WORK_SLACK_TOKEN=xoxp-work-slack-token
```

### Result

- Projects in `~/work/*` use work tokens and Slack integration
- Projects in `~/repos/*` and `~/side-projects/*` use personal tokens
- No Slack integration in personal projects

---

## Migrating from Single MCP Servers

Transition from directly configuring MCP servers to using Assern.

### Before (Direct MCP Configuration)

Your Claude Desktop config (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_token"
      }
    },
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"]
    }
  }
}
```

### After (With Assern)

1. **Install and configure Assern:**

```bash
assern config init
```

```yaml
# ~/.valksor/assern/config.yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"

  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
```

2. **Update Claude Desktop config:**

```json
{
  "mcpServers": {
    "assern": {
      "command": "assern",
      "args": ["serve"],
      "env": {
        "GITHUB_TOKEN": "ghp_token"
      }
    }
  }
}
```

3. **Benefits gained:**
   - Tools are now prefixed: `github_search` instead of `search`
   - Can add project-specific configs without touching Claude settings
   - Single entry point for all MCP servers

---

## Setting Up CI/CD with Assern

Use Assern in CI/CD pipelines with project-specific configurations.

### GitHub Actions Example

```yaml
name: CI with MCP

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Assern
        run: |
          curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/main/install.sh | bash

      - name: Setup MCP config
        run: |
          mkdir -p ~/.valksor/assern
          cat > ~/.valksor/assern/config.yaml << 'EOF'
          servers:
            github:
              command: npx
              args: ["-y", "@modelcontextprotocol/server-github"]
              env:
                GITHUB_TOKEN: "${GITHUB_TOKEN}"
          EOF

      - name: Run MCP tools
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          assern list
          # Use your MCP tools here
```

### Dockerfile Example

```dockerfile
FROM golang:1.25 AS builder

# Install Assern
RUN curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/main/install.sh | bash

# Setup config
RUN mkdir -p /root/.valksor/assern
COPY assern-config.yaml /root/.valksor/assern/config.yaml

# Your application...
```

---

## Selective Tool Exposure

Restrict which tools are available for security or simplicity.

### Read-Only Filesystem Access

```yaml
servers:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
    allowed:
      - read_file
      - list_directory
      - get_file_info
      # write_file, create_directory, delete_file NOT exposed
```

### Limited GitHub Access

```yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    allowed:
      - search_repositories
      - get_repository
      - get_file_contents
      # create_issue, push_files NOT exposed
```

---

## Local Development Server

Add a local development server that only exists in a specific project.

```yaml
# ~/.valksor/assern/config.yaml
servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"

projects:
  my-api-project:
    directories:
      - ~/projects/my-api/*
    servers:
      local-db:
        command: ./scripts/mcp-db-server
        env:
          DB_PATH: ./data/dev.db
```

Now `local-db_*` tools are only available when working in `~/projects/my-api/*`.

---

## Testing Configuration Changes

Safely test configuration changes without affecting your workflow.

1. **Create a test project:**

```yaml
# ~/.valksor/assern/config.yaml
projects:
  test:
    directories:
      - ~/tmp/test-assern/*
```

2. **Create test directory:**

```bash
mkdir -p ~/tmp/test-assern
cd ~/tmp/test-assern
```

3. **Validate configuration:**

```bash
assern config validate
```

4. **List what would be available:**

```bash
assern list
```

5. **Once satisfied, use in your actual project directory.**
