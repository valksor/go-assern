package instance

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// handshakeTimeout is the time to wait for the first message to determine
// if this is an internal command (ping) or an MCP client connection.
const handshakeTimeout = 100 * time.Millisecond

// Server manages the Unix socket for instance sharing.
type Server struct {
	socketPath string
	mcpServer  *server.MCPServer
	logger     *slog.Logger
	info       *Info

	listener net.Listener
	clients  map[net.Conn]struct{}
	mu       sync.Mutex
	wg       sync.WaitGroup
	done     chan struct{}
}

// NewServer creates a new instance sharing server.
func NewServer(socketPath string, mcpServer *server.MCPServer, logger *slog.Logger) *Server {
	cwd, _ := os.Getwd()

	return &Server{
		socketPath: socketPath,
		mcpServer:  mcpServer,
		logger:     logger,
		info: &Info{
			PID:        os.Getpid(),
			SocketPath: socketPath,
			StartTime:  time.Now(),
			WorkDir:    cwd,
		},
		clients: make(map[net.Conn]struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins listening on the Unix socket.
func (s *Server) Start() error {
	// Remove stale socket if exists
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		s.logger.Debug("failed to remove existing socket", "error", err)
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "unix", s.socketPath)
	if err != nil {
		return err
	}

	// Set socket permissions to owner only
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		_ = listener.Close()

		return err
	}

	s.listener = listener
	s.logger.Info("instance sharing socket listening", "path", s.socketPath)

	// Accept connections
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop closes the socket and all client connections.
func (s *Server) Stop() error {
	close(s.done)

	if s.listener != nil {
		_ = s.listener.Close()
	}

	// Close all client connections
	s.mu.Lock()
	for conn := range s.clients {
		_ = conn.Close()
	}
	s.clients = make(map[net.Conn]struct{})
	s.mu.Unlock()

	s.wg.Wait()

	// Clean up socket file
	_ = os.Remove(s.socketPath)

	return nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.logger.Debug("accept error", "error", err)

				continue
			}
		}

		s.mu.Lock()
		s.clients[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		_ = conn.Close()
	}()

	s.logger.Debug("client connected", "remote", conn.RemoteAddr())

	// Check for internal handshake command (ping/info) before starting MCP
	// This allows the detector to quickly check if an instance is running
	reader, handled := s.tryHandleInternalCommand(conn)
	if handled {
		s.logger.Debug("handled internal command, closing connection")

		return
	}

	// Not an internal command - proceed with MCP protocol
	// reader may contain buffered data from the handshake check
	s.serveMCP(conn, reader)
}

// tryHandleInternalCommand checks if the first message is an internal command.
// Returns the reader to use for subsequent reads and whether the command was handled.
// If handled is true, the connection should be closed.
// If handled is false, the returned reader should be used for MCP serving.
func (s *Server) tryHandleInternalCommand(conn net.Conn) (io.Reader, bool) {
	// Set deadline for reading first message
	if err := conn.SetReadDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		s.logger.Debug("failed to set read deadline", "error", err)

		return conn, false
	}

	// Read first line (newline-delimited JSON)
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')

	// Clear deadline for subsequent operations
	_ = conn.SetReadDeadline(time.Time{})

	if err != nil {
		// Timeout or error - not an internal command, use buffered reader
		if reader.Buffered() > 0 {
			// Some data was read but no newline yet - prepend it
			return io.MultiReader(reader, conn), false
		}

		return conn, false
	}

	// Try to parse as internal command
	var req struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Method  string `json:"method"`
	}

	if err := json.Unmarshal(line, &req); err != nil {
		// Not valid JSON - prepend the line and continue with MCP
		return io.MultiReader(bytes.NewReader(line), reader, conn), false
	}

	// Check if it's an internal command
	switch req.Method {
	case "assern/ping", "assern/info":
		s.sendInternalResponse(conn, req.ID, s.info)

		return nil, true
	}

	// Not an internal command - prepend the message for MCP to process
	return io.MultiReader(bytes.NewReader(line), reader, conn), false
}

func (s *Server) sendInternalResponse(conn net.Conn, id any, result any) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Debug("failed to marshal response", "error", err)

		return
	}

	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		s.logger.Debug("failed to write response", "error", err)
	}
}

func (s *Server) serveMCP(conn net.Conn, reader io.Reader) {
	// Create a context that cancels when server stops
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-s.done
		cancel()
	}()

	// Serve MCP protocol over this connection
	stdioServer := server.NewStdioServer(s.mcpServer)
	if err := stdioServer.Listen(ctx, reader, conn); err != nil {
		if !errors.Is(err, io.EOF) && !s.isStopped() {
			s.logger.Debug("client connection error", "error", err)
		}
	}

	s.logger.Debug("client disconnected")
}

func (s *Server) isStopped() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// extractJSONMessage attempts to extract a complete JSON message from buffer.
// Used by tests.
func extractJSONMessage(buf []byte) ([]byte, []byte, bool) {
	// Look for newline delimiter (JSON-RPC messages are newline-delimited)
	for i, b := range buf {
		if b == '\n' {
			return buf[:i], buf[i+1:], true
		}
	}

	return nil, buf, false
}
