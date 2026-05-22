package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// stdioSession is a tool-capable ClientSession for the primary instance's own
// stdio client. mcp-go's built-in stdio session does not implement
// server.SessionWithTools, so discovery mode drives stdio with this session to
// scope per-session tool loading the same way socket-proxied clients do.
type stdioSession struct {
	notifications      chan mcp.JSONRPCNotification
	initialized        atomic.Bool
	loggingLevel       atomic.Value
	clientInfo         atomic.Value
	clientCapabilities atomic.Value

	toolsMu      sync.RWMutex
	sessionTools map[string]server.ServerTool
}

func newStdioSession() *stdioSession {
	return &stdioSession{
		notifications: make(chan mcp.JSONRPCNotification, 100),
		sessionTools:  make(map[string]server.ServerTool),
	}
}

func (s *stdioSession) SessionID() string { return "stdio" }

func (s *stdioSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notifications
}

func (s *stdioSession) Initialize() {
	s.loggingLevel.Store(mcp.LoggingLevelError)
	s.initialized.Store(true)
}

func (s *stdioSession) Initialized() bool { return s.initialized.Load() }

func (s *stdioSession) GetClientInfo() mcp.Implementation {
	if v := s.clientInfo.Load(); v != nil {
		if ci, ok := v.(mcp.Implementation); ok {
			return ci
		}
	}

	return mcp.Implementation{}
}

func (s *stdioSession) SetClientInfo(clientInfo mcp.Implementation) {
	s.clientInfo.Store(clientInfo)
}

func (s *stdioSession) GetClientCapabilities() mcp.ClientCapabilities {
	if v := s.clientCapabilities.Load(); v != nil {
		if cc, ok := v.(mcp.ClientCapabilities); ok {
			return cc
		}
	}

	return mcp.ClientCapabilities{}
}

func (s *stdioSession) SetClientCapabilities(clientCapabilities mcp.ClientCapabilities) {
	s.clientCapabilities.Store(clientCapabilities)
}

func (s *stdioSession) SetLogLevel(level mcp.LoggingLevel) {
	s.loggingLevel.Store(level)
}

func (s *stdioSession) GetLogLevel() mcp.LoggingLevel {
	if v := s.loggingLevel.Load(); v != nil {
		if l, ok := v.(mcp.LoggingLevel); ok {
			return l
		}
	}

	return mcp.LoggingLevelError
}

func (s *stdioSession) GetSessionTools() map[string]server.ServerTool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()

	return s.sessionTools
}

func (s *stdioSession) SetSessionTools(tools map[string]server.ServerTool) {
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()

	s.sessionTools = tools
}

var (
	_ server.ClientSession         = (*stdioSession)(nil)
	_ server.SessionWithTools      = (*stdioSession)(nil)
	_ server.SessionWithLogging    = (*stdioSession)(nil)
	_ server.SessionWithClientInfo = (*stdioSession)(nil)
)

// lockedWriter serializes newline-delimited JSON-RPC writes so that responses
// and notifications never interleave on the same stream.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) writeMessage(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.w.Write(data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// serveStdioWithDiscovery serves the MCP server over stdio using a tool-capable
// session, enabling per-session progressive tool disclosure. It mirrors the
// socket serve loop used for proxied clients.
func serveStdioWithDiscovery(ctx context.Context, mcpServer *server.MCPServer, logger *slog.Logger) error {
	return runSessionLoop(ctx, mcpServer, newStdioSession(), os.Stdin, os.Stdout, logger)
}

// runSessionLoop registers session on mcpServer, then reads newline-delimited
// JSON-RPC messages from r, dispatches them, and writes responses and
// notifications to w. It returns when r reaches EOF or ctx is cancelled.
func runSessionLoop(ctx context.Context, mcpServer *server.MCPServer, session *stdioSession, r io.Reader, w io.Writer, logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := &lockedWriter{w: w}

	if err := mcpServer.RegisterSession(ctx, session); err != nil {
		return fmt.Errorf("registering stdio session: %w", err)
	}
	defer mcpServer.UnregisterSession(ctx, session.SessionID())

	ctx = mcpServer.WithContext(ctx, session)

	go forwardNotifications(ctx, session.notifications, out, logger)

	reader := bufio.NewReader(r)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("reading stdin: %w", err)
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		var rawMsg json.RawMessage
		if err := json.Unmarshal([]byte(line), &rawMsg); err != nil {
			logger.Debug("invalid JSON message on stdin", "error", err)

			continue
		}

		response := mcpServer.HandleMessage(ctx, rawMsg)
		if response == nil {
			continue
		}

		if err := out.writeMessage(response); err != nil {
			return err
		}
	}
}

// forwardNotifications relays server-to-client notifications to the output
// stream until the context is cancelled or the channel closes.
func forwardNotifications(ctx context.Context, notes <-chan mcp.JSONRPCNotification, out *lockedWriter, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case note, ok := <-notes:
			if !ok {
				return
			}

			if err := out.writeMessage(note); err != nil {
				logger.Debug("failed to write notification", "error", err)

				return
			}
		}
	}
}
