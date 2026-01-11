# Build stage
FROM golang:1.25-trixie AS builder

# Set build arguments (TARGETOS/TARGETARCH provided by buildx)
ARG VERSION=dev
ARG BUILD_TIME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (supports multi-platform via buildx)
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT:-} \
    GOAMD64=${TARGETVARIANT:-} \
    go build -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o assern ./cmd/assern

# Runtime stage
FROM debian:trixie-slim

# Install Node.js (for npx support with MCP servers)
# Use official Node.js installation script
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl; \
    curl -fsSL https://deb.nodesource.com/setup_24.x | bash -; \
    apt-get install -y --no-install-recommends nodejs; \
    node --version; \
    npm --version; \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1000 -s /bin/bash assern

# Set working directory
WORKDIR /home/assern

# Copy binary from builder
COPY --from=builder /build/assern /usr/local/bin/assern

# Create config directory
RUN mkdir -p /home/assern/.valksor/assern && \
    chown -R assern:assern /home/assern

# Switch to non-root user
USER assern

# Set default config directory as volume
VOLUME ["/home/assern/.valksor/assern"]

# Default command
ENTRYPOINT ["/usr/local/bin/assern"]
CMD ["serve"]
