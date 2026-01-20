# Troubleshooting

Common issues and solutions when using Assern.

## Debug Commands

Use these commands to diagnose issues:

```bash
# Validate configuration syntax
assern config validate

# List detected project and available tools
assern list

# Enable debug logging
assern serve --verbose
```

---

## Server Startup Failures

### Error: `command not found`

**Symptom:** Server fails to start with `exec: "npx": not found`

**Cause:** The command specified for an MCP server is not available on your system.

**Solution:** Install the required software for your MCP servers. For example:
- `npx` servers require Node.js (install via `nvm` or system package manager)
- Binary servers require the binary to be in your `$PATH`

```bash
# Verify npx is available
which npx

# Install Node.js if needed (macOS)
brew install node

# Or use nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install node
```

### Error: `permission denied`

**Symptom:** Server binary cannot be executed.

**Solution:** Make the server executable:

```bash
chmod +x /path/to/server-binary
```

---

## Server Initialization Failures

### Exit Status 127

**Symptom**: `error closing client after init failure" error="exit status 127"` or `server sequential-thinking: command '/path/to/command' not found`

**Meaning**: Command not found - the specified command executable doesn't exist or isn't accessible.

**Solutions**:
1. **Verify command exists**:
   ```bash
   # Check if the command exists
   ls -la /path/to/command
   which npx  # or your command
   ```

2. **Use absolute path in config**:
   ```json
   {
     "mcpServers": {
       "sequential-thinking": {
         "command": "/absolute/path/to/npx",
         "args": ["-y", "@modelcontextprotocol/server-sequential-thinking"]
       }
     }
   }
   ```

3. **Install missing package** (e.g., Node.js):
   ```bash
   # macOS
   brew install node

   # Or use nvm
   curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
   nvm install node

   # Verify installation
   which npx
   npx --version
   ```

### Context Deadline Exceeded

**Symptom**: `server sequential-thinking: initialization timeout after 1m0s` or `initializing connection: transport error: context deadline exceeded`

**Meaning**: Server took too long to respond (default: 60 seconds).

**Possible Causes**:
1. **First-time package download**: npx downloading packages from npm registry
2. **Network issues**: Slow or unreliable internet connection
3. **Resource constraints**: Insufficient CPU/memory
4. **Command hanging**: Process waiting for input or stuck in loop

**Solutions**:
1. **Increase timeout** in `~/.valksor/assern/config.yaml`:
   ```yaml
   settings:
     timeout: 5m0s  # 5 minutes for first-time package downloads
   ```

2. **Pre-download packages** to avoid timeout during startup:
   ```bash
   npx -y @modelcontextprotocol/server-sequential-thinking --version
   ```

3. **Check system resources**:
   ```bash
   top  # check CPU/memory usage
   df -h  # check disk space
   ```

4. **Test command manually** to see what's happening:
   ```bash
   /path/to/npx -y @modelcontextprotocol/server-sequential-thinking
   ```

5. **Enable debug logging** to see timing information:
   ```bash
   assern serve --verbose
   ```

### Intermittent Failures

If failures are intermittent (sometimes work, sometimes fail):

1. **Check for race conditions**: Multiple servers starting simultaneously
2. **Review system logs**:
   ```bash
   journalctl -f  # Linux
   # or
   tail -f /var/log/syslog
   ```
3. **Enable debug logging** to see timing information:
   ```bash
   assern serve --verbose
   ```

---

## Configuration Issues

### Error: `invalid YAML syntax`

**Symptom:** Configuration validation fails with YAML syntax error.

**Solution:** Check your YAML for common issues:
- Use spaces, not tabs (2 spaces for indentation)
- Ensure colons have a space after them
- Quote strings with special characters

```bash
# Validate your config
assern config validate
```

### Project auto-detected but need specific config

**Symptom:** Assern auto-detects the project name from directory (e.g., `my-repo`), but you need project-specific environment variables or server overrides.

**Solution:** Either:
1. Create `.assern/config.yaml` in your project directory with `project: work` to link to a global project
2. Add directory pattern to global config's `projects.*.directories`
3. Use `--project` flag to explicitly specify project

```bash
# Check which project is detected
assern list

# Explicitly specify project
assern serve --project work

# Or create local config to link to existing project
mkdir -p .assern && echo "project: work" > .assern/config.yaml
```

> **Note:** Assern auto-detects the project name from the directory basename when no configuration is found, so you can always run `assern serve` in any directory.

### Error: `environment variable not found`

**Symptom:** `${VAR}` expansion fails, variable is empty.

**Solution:** Ensure the variable is set in one of:
1. System environment
2. `~/.valksor/assern/.env`
3. `.assern/.env`

```bash
# Check if variable is set
echo $MY_TOKEN

# Set in .env file
echo 'MY_TOKEN=value' >> ~/.valksor/assern/.env
```

---

## Project Detection Problems

### Wrong project detected

**Symptom:** `assern list` shows a different project than expected.

**Cause:** Directory pattern matches multiple project definitions.

**Solution:** Check your directory patterns:
- More specific patterns take precedence
- Use `**` for nested directories vs `*` for single level
- Create `.assern/config.yaml` to explicitly set project

```yaml
# In .assern/config.yaml
project: work  # Force this project
```

### Changes to config not taking effect

**Symptom:** Updated configuration but Assern uses old values.

**Solution:** Restart Assern - configuration is read on startup only.

```bash
# Stop running instance and restart
assern serve
```

---

## IDE Integration Issues

### Tools not appearing in IDE/Claude

**Symptom:** `assern list` shows tools, but Claude Desktop/Code doesn't see them.

**Cause:** Missing `"type": "stdio"` in MCP server configuration.

**Solution:** Ensure your MCP config includes the `type` field:

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

The `type` field tells the MCP client how to communicate with the server. Without it, the client may not establish proper communication.

---

## Tool Call Issues

### Tool not found

**Symptom:** MCP client reports tool not available.

**Solution:** Use `assern list` to verify the tool exists:
- Check server is enabled (not `disabled: true`)
- Check tool is in `allowed` list if configured
- Check tool name includes server prefix

```bash
assern list | grep github
# Output: github_search_repositories, github_get_repository, etc.
```

### Server returns error

**Symptom:** Tool call fails with server-specific error.

**Solution:** This is a backend server issue, not Assern. Check:
- API tokens are valid and not expired
- Server has required permissions
- Rate limits not exceeded

```bash
# Enable debug logging to see server communication
assern serve --verbose
```

---

## Permission Issues

### Error: `permission denied` reading config

**Symptom:** Cannot read `~/.valksor/assern/config.yaml`.

**Solution:** Fix file permissions:

```bash
chmod 644 ~/.valksor/assern/config.yaml
chmod 600 ~/.valksor/assern/.env  # .env should be restricted
```

### Error: `cannot create directory`

**Symptom:** Cannot initialize configuration.

**Solution:** Ensure parent directory exists and is writable:

```bash
mkdir -p ~/.valksor/assern
chmod 755 ~/.valksor/assern
```

---

## Performance Issues

### Slow startup

**Symptom:** Assern takes a long time to start serving.

**Possible causes:**
- Many servers to spawn
- Servers with slow initialization
- Network latency for API-based servers

**Solutions:**
- Disable unused servers with `disabled: true`
- Use `allowed` to reduce tool discovery overhead
- Increase `timeout` in settings if needed

```yaml
settings:
  timeout: 120s  # Increase from default 60s
```

---

## Instance Sharing Issues

### Secondary instance not connecting to primary

**Symptom:** Multiple Assern instances running with separate MCP servers.

**Cause:** Instance detection failed or was disabled.

**Solution:** Check:
1. Socket file exists: `ls -la ~/.valksor/assern/assern.sock`
2. Environment variable not set: `echo $ASSERN_NO_INSTANCE_SHARING`
3. Primary instance is running: `ps aux | grep assern`

```bash
# Verify socket exists and has correct permissions
ls -la ~/.valksor/assern/assern.sock
# Should show: srw------- ... assern.sock
```

### Stale socket preventing startup

**Symptom:** New instance fails to start or behaves unexpectedly.

**Cause:** Previous instance crashed without cleaning up socket.

**Solution:** Assern automatically cleans stale sockets, but if issues persist:

```bash
# Remove stale socket manually
rm ~/.valksor/assern/assern.sock

# Restart Assern
assern serve
```

### Need isolated instance for testing

**Symptom:** Want to test configuration changes without affecting running instance.

**Solution:** Disable instance sharing temporarily:

```bash
ASSERN_NO_INSTANCE_SHARING=1 assern serve
```

### Primary instance exit disconnects proxies

**Symptom:** Secondary instances lose connection when primary exits.

**Expected behavior:** This is normal. When the primary instance stops, all proxy instances lose their connection.

**Solution:** Restart Assern - the next invocation becomes the new primary.

---

## Getting Help

If you're still stuck:

1. **Enable debug logging:**
   ```bash
   assern serve --verbose
   ```

2. **Validate configuration:**
   ```bash
   assern config validate
   ```

3. **Check existing issues:**
   [GitHub Issues](https://github.com/valksor/go-assern/issues)

4. **Create a new issue:**
   Include your debug output and configuration (with sensitive values redacted).
