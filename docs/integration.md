# Integration

Configure Assern with MCP clients like Claude Desktop, Claude Code, and other IDEs.

## Claude Desktop

### macOS Location

`~/Library/Application Support/Claude/claude_desktop_config.json`

### Windows Location

`%APPDATA%\Claude\claude_desktop_config.json`

### Configuration

```json
{
  "mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"]
    }
  }
}
```

Assern will auto-detect your project based on the current working directory of your Claude session.

### With Environment Variables

If your MCP servers need tokens:

```json
{
  "mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here",
        "SLACK_TOKEN": "xoxp_your_token_here"
      }
    }
  }
}
```

> **Tip:** For security, prefer storing tokens in `~/.valksor/assern/.env` instead of the Claude config file.

---

## Claude Code (CLI)

Claude Code can use MCP servers through environment configuration.

### Option 1: Global MCP Config

Create or edit `~/.config/claude-code/mcp_servers.json`:

```json
{
  "mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"]
    }
  }
}
```

### Option 2: Project-Specific Config

Create `.claude-code/mcp_servers.json` in your project:

```json
{
  "mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"]
    }
  }
}
```

### Verify Connection

Start Claude Code in a directory with an Assern project:

```bash
cd ~/work/myproject
claude  # Assern tools will be available
```

---

## VS Code with Claude Extension

If using VS Code with the Claude extension:

### Settings Location

**VS Code Settings** → **Extensions** → **Claude** → **MCP Servers**

Or add to `.vscode/settings.json`:

```json
{
  "claude.mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"]
    }
  }
}
```

---

## JetBrains IDEs (IntelliJ, PyCharm, etc.)

For JetBrains IDEs with Claude integration:

### Plugin Configuration

1. Open **Settings/Preferences**
2. Navigate to **Tools** → **Claude** → **MCP Servers**
3. Add server configuration:

| Field | Value |
|-------|-------|
| Name | `assern` |
| Command | `assern` |
| Arguments | `serve` |

---

## Cursor Editor

Cursor supports MCP servers through configuration files.

### Config Location

`~/.cursor/mcp_config.json` or project-specific `.cursor/mcp_config.json`

```json
{
  "mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"]
    }
  }
}
```

---

## Project Detection with IDEs

Assern detects your project based on the IDE's working directory.

### How It Works

```
IDE opens: /home/user/work/acme/my-repo
                │
                ▼
        Assern detects:
        - .assern/config.yaml in repo? → Use local config
        - Matches ~/work/acme/*? → Use "acme" project
        - Neither? → Use global config only
```

### Verifying Detection

When Assern starts, it logs which project is detected:

```
INFO  Detected project: work (from directory pattern ~/work/*)
INFO  Started server: github (5 tools)
INFO  Started server: jira (3 tools)
```

Run `assern list` in your project directory to verify:

```bash
cd ~/work/myproject
assern list

# Output:
# Project: work (detected from directory match)
#
# Servers:
#   github (5 tools)
#   jira (3 tools)
```

---

## Common Issues

### "Assern command not found"

**Cause:** The `assern` binary is not in your system `$PATH`.

**Solution:**
- Ensure `~/.local/bin` is in your `$PATH`
- Or use full path in config: `/home/user/.local/bin/assern`

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"
```

### Wrong project detected

**Cause:** Directory pattern matches multiple projects.

**Solution:** Create `.assern/config.yaml` in your project to explicitly set the project:

```yaml
# .assern/config.yaml
project: work  # Force this project
```

### Tools not appearing in IDE

**Cause:** Missing `type` field, Assern failed to start, or servers failed to spawn.

**Solution:**

1. **Ensure `"type": "stdio"` is set** in your MCP server config:
   ```json
   "assern": {
     "type": "stdio",
     "command": "assern",
     "args": ["serve"]
   }
   ```
   Without this, the client may not know how to communicate with Assern.

2. Check Assern works directly: `assern list`

3. Enable debug logging in your config:
   ```yaml
   settings:
     log_level: debug
   ```

4. Check IDE logs for MCP server errors

---

## Example: Complete Setup

Here's a complete example from scratch:

### 1. Install Assern

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/main/install.sh | bash
```

### 2. Configure Assern

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

projects:
  work:
    directories:
      - ~/work/*
    env:
      GITHUB_TOKEN: "${WORK_GITHUB_TOKEN}"
```

### 3. Configure Claude Desktop

```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "assern": {
      "type": "stdio",
      "command": "assern",
      "args": ["serve"]
    }
  }
}
```

### 4. Restart Claude Desktop

Claude will now have access to `github_search_repositories`, `github_get_repository`, etc., with the correct token based on which directory you're working in.
