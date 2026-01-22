package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
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

func TestServer_MCPInitialize(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	srv := NewServer(socketPath, mcpServer, logger)

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

	// Wait for handshake timeout so server proceeds to MCP mode
	time.Sleep(handshakeTimeout + 50*time.Millisecond)

	// Send MCP initialize request
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}` + "\n"
	if _, err := conn.Write([]byte(initReq)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
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
		Result  struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v, response = %s", err, buf[:n])
	}

	// Verify response
	if resp.Error != nil {
		t.Fatalf("Initialize returned error: code=%d, message=%s", resp.Error.Code, resp.Error.Message)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Response jsonrpc = %s, want 2.0", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("Response id = %d, want 1", resp.ID)
	}

	if resp.Result.ServerInfo.Name != "test-server" {
		t.Errorf("Response serverInfo.name = %s, want test-server", resp.Result.ServerInfo.Name)
	}
}

func TestServer_MCPListTools(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	// Add a test tool
	mcpServer.AddTool(
		mcp.NewTool("test_tool", mcp.WithDescription("A test tool")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("test"), nil
		},
	)

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

	// Wait for handshake timeout
	time.Sleep(handshakeTimeout + 50*time.Millisecond)

	// Initialize first
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"
	if _, err := conn.Write([]byte(initReq)); err != nil {
		t.Fatalf("Write init error = %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	_, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("Read init response error = %v", err)
	}

	// Send initialized notification
	initializedNotif := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	if _, err := conn.Write([]byte(initializedNotif)); err != nil {
		t.Fatalf("Write initialized error = %v", err)
	}

	// Now list tools
	listToolsReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"
	if _, err := conn.Write([]byte(listToolsReq)); err != nil {
		t.Fatalf("Write list tools error = %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read list tools error = %v", err)
	}

	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v, response = %s", err, buf[:n])
	}

	if resp.ID != 2 {
		t.Errorf("Response id = %d, want 2", resp.ID)
	}

	if len(resp.Result.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(resp.Result.Tools))
	}

	if resp.Result.Tools[0].Name != "test_tool" {
		t.Errorf("Tool name = %s, want test_tool", resp.Result.Tools[0].Name)
	}
}

func TestServer_MultipleMCPClients(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Connect multiple MCP clients concurrently
	const numClients = 3
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

			// Wait for handshake timeout
			time.Sleep(handshakeTimeout + 50*time.Millisecond)

			// Send initialize request with unique ID
			initReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"client-%d","version":"1.0"}}}`, clientID*10+1, clientID) + "\n"
			if _, err := conn.Write([]byte(initReq)); err != nil {
				errCh <- fmt.Errorf("client %d: Write() error = %w", clientID, err)

				return
			}

			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- fmt.Errorf("client %d: Read() error = %w", clientID, err)

				return
			}

			var resp struct {
				ID    int `json:"id"`
				Error *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(buf[:n], &resp); err != nil {
				errCh <- fmt.Errorf("client %d: Unmarshal() error = %w, response = %s", clientID, err, buf[:n])

				return
			}

			if resp.Error != nil {
				errCh <- fmt.Errorf("client %d: Initialize error: code=%d, message=%s", clientID, resp.Error.Code, resp.Error.Message)

				return
			}

			if resp.ID != clientID*10+1 {
				errCh <- fmt.Errorf("client %d: got id %d, want %d", clientID, resp.ID, clientID*10+1)

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

func TestServer_MCPAfterInternal(t *testing.T) {
	// Test that the socket server can handle both internal commands (ping)
	// and MCP protocol on different connections
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "s.sock")

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	ctx := t.Context()
	var dialer net.Dialer

	// First connection: send ping (internal command)
	conn1, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		t.Fatalf("Dial() conn1 error = %v", err)
	}

	pingReq := `{"jsonrpc":"2.0","id":1,"method":"assern/ping"}` + "\n"
	if _, err := conn1.Write([]byte(pingReq)); err != nil {
		t.Fatalf("Write() ping error = %v", err)
	}

	_ = conn1.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 4096)
	n, err := conn1.Read(buf)
	if err != nil {
		t.Fatalf("Read() ping response error = %v", err)
	}

	var pingResp struct {
		ID     int   `json:"id"`
		Result *Info `json:"result"`
	}
	if err := json.Unmarshal(buf[:n], &pingResp); err != nil {
		t.Fatalf("Unmarshal ping response error = %v", err)
	}

	if pingResp.Result == nil {
		t.Fatal("Ping response result is nil")
	}

	_ = conn1.Close()

	// Second connection: send MCP initialize
	conn2, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		t.Fatalf("Dial() conn2 error = %v", err)
	}
	defer func() { _ = conn2.Close() }()

	// Wait for handshake timeout
	time.Sleep(handshakeTimeout + 50*time.Millisecond)

	initReq := `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"
	if _, err := conn2.Write([]byte(initReq)); err != nil {
		t.Fatalf("Write() initialize error = %v", err)
	}

	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn2.Read(buf)
	if err != nil {
		t.Fatalf("Read() initialize response error = %v", err)
	}

	var initResp struct {
		ID    int `json:"id"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(buf[:n], &initResp); err != nil {
		t.Fatalf("Unmarshal initialize response error = %v, response = %s", err, buf[:n])
	}

	if initResp.Error != nil {
		t.Errorf("Initialize returned error after ping: code=%d, message=%s", initResp.Error.Code, initResp.Error.Message)
	}

	if initResp.ID != 2 {
		t.Errorf("Initialize response id = %d, want 2", initResp.ID)
	}
}
