package instance

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

func TestNewProxy(t *testing.T) {
	t.Parallel()

	socketPath := "/tmp/test.sock"
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	proxy := NewProxy(socketPath, logger)

	if proxy == nil {
		t.Fatal("NewProxy() returned nil")
	}

	if proxy.socketPath != socketPath {
		t.Errorf("NewProxy() socketPath = %s, want %s", proxy.socketPath, socketPath)
	}

	if proxy.logger != logger {
		t.Error("NewProxy() logger not set correctly")
	}

	if proxy.conn != nil {
		t.Error("NewProxy() conn should be nil initially")
	}
}

func TestProxy_Connect_NoSocket(t *testing.T) {
	t.Parallel()

	socketPath := "/nonexistent/path/to/socket.sock"
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	proxy := NewProxy(socketPath, logger)

	err := proxy.Connect(t.Context())
	if err == nil {
		t.Fatal("Connect() should fail when socket doesn't exist")
	}
}

func TestProxy_Close_NotConnected(t *testing.T) {
	t.Parallel()

	socketPath := "/tmp/test.sock"
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	proxy := NewProxy(socketPath, logger)

	// Close without connecting should not error
	err := proxy.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil when not connected", err)
	}
}

func TestProxy_Connect_Success(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create a socket server
	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("Server.Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Create proxy and connect
	proxy := NewProxy(socketPath, logger)
	defer func() { _ = proxy.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := proxy.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if proxy.conn == nil {
		t.Error("Connect() should set conn")
	}
}

func TestProxy_Connect_Timeout(t *testing.T) {
	t.Parallel()

	// Use a path that doesn't exist
	socketPath := "/nonexistent/path/test.sock"
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	proxy := NewProxy(socketPath, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := proxy.Connect(ctx)
	if err == nil {
		t.Fatal("Connect() should fail with non-existent socket")
	}
}

func TestProxy_Close_AfterConnect(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("Server.Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	proxy := NewProxy(socketPath, logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := proxy.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Close should succeed
	if err := proxy.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Connection should be closed
	// Note: conn is still set but the underlying connection is closed
}

func TestProxy_MultipleConnections(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("Server.Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Create multiple proxies
	const numProxies = 5
	proxies := make([]*Proxy, numProxies)

	for i := range numProxies {
		proxies[i] = NewProxy(socketPath, logger)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if err := proxies[i].Connect(ctx); err != nil {
			cancel()
			t.Fatalf("Proxy %d Connect() error = %v", i, err)
		}
		cancel()
	}

	// Close all proxies
	for i, proxy := range proxies {
		if err := proxy.Close(); err != nil {
			t.Errorf("Proxy %d Close() error = %v", i, err)
		}
	}
}
