# Docker Setup

Assern can run in a Docker container with full support for stdio-based MCP servers (via npx) and HTTP/SSE servers.

## Quick Start

### Using Docker Compose (Recommended)

1. **Create your config directory:**

```bash
mkdir -p ~/.valksor/assern
```

2. **Create `~/.valksor/assern/mcp.json`:**

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
      "args": ["-y", "@modelcontextprotocol/server-filesystem"],
      "env": {
        "FS_ALLOWED_DIRS": "${HOME}"
      }
    }
  }
}
```

3. **Start the container:**

```bash
docker-compose up -d
```

### Using Docker Directly

1. **Build the image:**

```bash
docker build -t assern .
```

2. **Run the container:**

```bash
docker run -d \
  --name assern-mcp \
  -v ~/.valksor/assern:/home/assern/.valksor/assern:ro \
  assern
```

3. **Run interactively (for testing):**

```bash
docker run -it --rm \
  -v ~/.valksor/assern:/home/assern/.valksor/assern:ro \
  assern list
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ASSERN_LOG_LEVEL` | `info` | Log level: error, info, debug, trace |
| `ASSERN_OUTPUT_FORMAT` | `json` | Output format: json or toon |

Pass them in `docker-compose.yml` or via `docker run -e`:

```bash
docker run -e ASSERN_LOG_LEVEL=debug -e ASSERN_OUTPUT_FORMAT=toon assern
```

### Using with MCP Clients

The Docker image runs Assern in stdio mode by default, compatible with any MCP client.

**Example Claude Desktop config:**

```json
{
  "mcpServers": {
    "assern": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "~/.valksor/assern:/home/assern/.valksor/assern:ro",
        "valksor/assern:latest"
      ]
    }
  }
}
```

## Image Details

- **Base Image:** `debian:trixie-slim`
- **Go Version:** 1.25
- **Node.js Version:** 24 LTS (for npx support)
- **User:** Non-root user (`assern:1000`)
- **Working Directory:** `/home/assern`

## Building from Source

### Single Platform

To build with version tags:

```bash
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t assern:local .
```

### Multi-Platform (AMD64 + ARM64)

Build for multiple platforms using buildx:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t assern:local \
  --load .
```

Or push to a registry:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t your-registry/assern:latest \
  --push .
```

## Troubleshooting

### Permission Issues

The container runs as user `1000`. If your config directory has restrictive permissions:

```bash
chmod -R 755 ~/.valksor/assern
```

### npx Not Found

The image includes Node.js 24 LTS. Verify:

```bash
docker run --rm --entrypoint node assern --version
docker run --rm --entrypoint npx assern --version
```

### Volume Mount Issues on macOS

If the volume mount doesn't work, ensure Docker Desktop has access to your home directory in Settings → Resources → File Sharing.

### Debug Logging

```bash
docker run -it --rm \
  -v ~/.valksor/assern:/home/assern/.valksor/assern:ro \
  -e ASSERN_LOG_LEVEL=trace \
  assern serve --verbose
```
