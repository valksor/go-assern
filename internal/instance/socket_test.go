package instance

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	socketPath := "/tmp/test.sock"
	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}

	if srv.socketPath != socketPath {
		t.Errorf("NewServer() socketPath = %s, want %s", srv.socketPath, socketPath)
	}

	if srv.mcpServer != mcpServer {
		t.Error("NewServer() mcpServer not set correctly")
	}

	if srv.logger != logger {
		t.Error("NewServer() logger not set correctly")
	}

	if srv.info == nil {
		t.Fatal("NewServer() info is nil")
	}

	if srv.info.PID != os.Getpid() {
		t.Errorf("NewServer() info.PID = %d, want %d", srv.info.PID, os.Getpid())
	}

	if srv.info.SocketPath != socketPath {
		t.Errorf("NewServer() info.SocketPath = %s, want %s", srv.info.SocketPath, socketPath)
	}

	if srv.clients == nil {
		t.Error("NewServer() clients map is nil")
	}

	if srv.done == nil {
		t.Error("NewServer() done channel is nil")
	}
}

func TestServer_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	// Start the server
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify socket file exists
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("Socket file not created: %v", err)
	}

	// Check socket file permissions (should be 0600)
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("Socket file permissions = %o, want 0600", perm)
	}

	// Stop the server
	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify socket file is cleaned up
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after Stop()")
	}
}

func TestServer_IsStopped(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	// Before start, should not be stopped (done channel not closed)
	if srv.isStopped() {
		t.Error("isStopped() should return false before Stop() is called")
	}

	// Start the server
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Still not stopped
	if srv.isStopped() {
		t.Error("isStopped() should return false while server is running")
	}

	// Stop the server
	_ = srv.Stop()

	// Now should be stopped
	if !srv.isStopped() {
		t.Error("isStopped() should return true after Stop() is called")
	}
}

func TestExtractJSONMessage_Complete(t *testing.T) {
	t.Parallel()

	buf := []byte(`{"method":"test"}` + "\n" + `remaining`)

	msg, remaining, ok := extractJSONMessage(buf)
	if !ok {
		t.Fatal("extractJSONMessage() should return ok=true for complete message")
	}

	if string(msg) != `{"method":"test"}` {
		t.Errorf("extractJSONMessage() msg = %s, want %s", msg, `{"method":"test"}`)
	}

	if string(remaining) != "remaining" {
		t.Errorf("extractJSONMessage() remaining = %s, want %s", remaining, "remaining")
	}
}

func TestExtractJSONMessage_Incomplete(t *testing.T) {
	t.Parallel()

	buf := []byte(`{"method":"test"`)

	msg, remaining, ok := extractJSONMessage(buf)
	if ok {
		t.Fatal("extractJSONMessage() should return ok=false for incomplete message")
	}

	if msg != nil {
		t.Errorf("extractJSONMessage() msg = %v, want nil", msg)
	}

	if string(remaining) != string(buf) {
		t.Errorf("extractJSONMessage() remaining = %s, want %s", remaining, buf)
	}
}

func TestExtractJSONMessage_Empty(t *testing.T) {
	t.Parallel()

	buf := []byte{}

	_, remaining, ok := extractJSONMessage(buf)
	if ok {
		t.Fatal("extractJSONMessage() should return ok=false for empty buffer")
	}

	if len(remaining) != 0 {
		t.Errorf("extractJSONMessage() remaining should be empty")
	}
}

func TestExtractJSONMessage_JustNewline(t *testing.T) {
	t.Parallel()

	buf := []byte("\n")

	msg, remaining, ok := extractJSONMessage(buf)
	if !ok {
		t.Fatal("extractJSONMessage() should return ok=true")
	}

	if len(msg) != 0 {
		t.Errorf("extractJSONMessage() msg should be empty")
	}

	if len(remaining) != 0 {
		t.Errorf("extractJSONMessage() remaining should be empty")
	}
}

func TestServer_PingPong(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	srv := NewServer(socketPath, mcpServer, logger)

	// Start the server
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect to the socket
	var dialer net.Dialer
	conn, err := dialer.DialContext(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send ping request
	pingReq := `{"jsonrpc":"2.0","id":1,"method":"assern/ping"}` + "\n"
	if _, err := conn.Write([]byte(pingReq)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Parse response
	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  *Info  `json:"result"`
	}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v, response = %s", err, buf[:n])
	}

	// Verify response
	if resp.JSONRPC != "2.0" {
		t.Errorf("Response jsonrpc = %s, want 2.0", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("Response id = %d, want 1", resp.ID)
	}

	if resp.Result == nil {
		t.Fatal("Response result is nil")
	}

	if resp.Result.PID != os.Getpid() {
		t.Errorf("Response result.PID = %d, want %d", resp.Result.PID, os.Getpid())
	}

	if resp.Result.SocketPath != socketPath {
		t.Errorf("Response result.SocketPath = %s, want %s", resp.Result.SocketPath, socketPath)
	}
}

func TestServer_InfoCommand(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	var dialer net.Dialer
	conn, err := dialer.DialContext(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send info request (alternative to ping)
	infoReq := `{"jsonrpc":"2.0","id":42,"method":"assern/info"}` + "\n"
	if _, err := conn.Write([]byte(infoReq)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	var resp struct {
		ID     int   `json:"id"`
		Result *Info `json:"result"`
	}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.ID != 42 {
		t.Errorf("Response id = %d, want 42", resp.ID)
	}

	if resp.Result == nil {
		t.Fatal("Response result is nil")
	}
}

func TestServer_MultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect multiple clients and send pings concurrently
	const numClients = 5
	errCh := make(chan error, numClients)
	ctx := t.Context()

	for i := range numClients {
		go func(clientID int) {
			var dialer net.Dialer
			conn, err := dialer.DialContext(ctx, "unix", socketPath)
			if err != nil {
				errCh <- fmt.Errorf("client %d: Dial() error = %w", clientID, err)

				return
			}
			defer func() { _ = conn.Close() }()

			pingReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"assern/ping"}`+"\n", clientID)
			if _, err := conn.Write([]byte(pingReq)); err != nil {
				errCh <- fmt.Errorf("client %d: Write() error = %w", clientID, err)

				return
			}

			_ = conn.SetReadDeadline(time.Now().Add(time.Second))

			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- fmt.Errorf("client %d: Read() error = %w", clientID, err)

				return
			}

			var resp struct {
				ID     int   `json:"id"`
				Result *Info `json:"result"`
			}
			if err := json.Unmarshal(buf[:n], &resp); err != nil {
				errCh <- fmt.Errorf("client %d: Unmarshal() error = %w", clientID, err)

				return
			}

			if resp.ID != clientID {
				errCh <- fmt.Errorf("client %d: got id %d", clientID, resp.ID)

				return
			}

			errCh <- nil
		}(i)
	}

	// Wait for all clients
	for range numClients {
		if err := <-errCh; err != nil {
			t.Error(err)
		}
	}
}

func TestServer_HandshakeTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect but don't send anything - should timeout gracefully
	var dialer net.Dialer
	conn, err := dialer.DialContext(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}

	// Wait for handshake timeout
	time.Sleep(handshakeTimeout + 50*time.Millisecond)

	// Connection should still be usable (server proceeds with MCP after timeout)
	// Close the connection
	_ = conn.Close()
}

func TestHandshakeTimeout_Value(t *testing.T) {
	t.Parallel()

	if handshakeTimeout <= 0 {
		t.Errorf("handshakeTimeout = %v, should be positive", handshakeTimeout)
	}

	if handshakeTimeout > time.Second {
		t.Errorf("handshakeTimeout = %v, should be <= 1 second for responsiveness", handshakeTimeout)
	}
}
