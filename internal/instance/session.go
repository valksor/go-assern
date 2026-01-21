package instance

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// socketSession implements server.ClientSession for socket connections.
// Each socket connection gets its own unique session to avoid conflicts
// with the stdio session used by the primary instance.
type socketSession struct {
	id                 string
	notifications      chan mcp.JSONRPCNotification
	initialized        atomic.Bool
	loggingLevel       atomic.Value
	clientInfo         atomic.Value
	clientCapabilities atomic.Value
}

// newSocketSession creates a new session with a unique ID for a socket connection.
func newSocketSession() *socketSession {
	return &socketSession{
		id:            fmt.Sprintf("socket-%d", time.Now().UnixNano()),
		notifications: make(chan mcp.JSONRPCNotification, 100),
	}
}

func (s *socketSession) SessionID() string {
	return s.id
}

func (s *socketSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notifications
}

func (s *socketSession) Initialize() {
	s.loggingLevel.Store(mcp.LoggingLevelError)
	s.initialized.Store(true)
}

func (s *socketSession) Initialized() bool {
	return s.initialized.Load()
}

func (s *socketSession) GetClientInfo() mcp.Implementation {
	if value := s.clientInfo.Load(); value != nil {
		if clientInfo, ok := value.(mcp.Implementation); ok {
			return clientInfo
		}
	}

	return mcp.Implementation{}
}

func (s *socketSession) SetClientInfo(clientInfo mcp.Implementation) {
	s.clientInfo.Store(clientInfo)
}

func (s *socketSession) GetClientCapabilities() mcp.ClientCapabilities {
	if value := s.clientCapabilities.Load(); value != nil {
		if clientCapabilities, ok := value.(mcp.ClientCapabilities); ok {
			return clientCapabilities
		}
	}

	return mcp.ClientCapabilities{}
}

func (s *socketSession) SetClientCapabilities(clientCapabilities mcp.ClientCapabilities) {
	s.clientCapabilities.Store(clientCapabilities)
}

func (s *socketSession) SetLogLevel(level mcp.LoggingLevel) {
	s.loggingLevel.Store(level)
}

func (s *socketSession) GetLogLevel() mcp.LoggingLevel {
	level := s.loggingLevel.Load()
	if level == nil {
		return mcp.LoggingLevelError
	}

	if l, ok := level.(mcp.LoggingLevel); ok {
		return l
	}

	return mcp.LoggingLevelError
}

// close closes the notification channel.
func (s *socketSession) close() {
	close(s.notifications)
}
