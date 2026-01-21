package instance

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewSocketSession(t *testing.T) {
	t.Parallel()

	session := newSocketSession()

	if session == nil {
		t.Fatal("newSocketSession() returned nil")
	}

	if session.id == "" {
		t.Error("newSocketSession() id should not be empty")
	}

	if !strings.HasPrefix(session.id, "socket-") {
		t.Errorf("newSocketSession() id = %s, should start with 'socket-'", session.id)
	}

	if session.notifications == nil {
		t.Error("newSocketSession() notifications channel should not be nil")
	}
}

func TestSocketSession_UniqueIDs(t *testing.T) {
	t.Parallel()

	// Create multiple sessions and verify they have unique IDs
	sessions := make([]*socketSession, 10)
	ids := make(map[string]bool)

	for i := range sessions {
		sessions[i] = newSocketSession()
		id := sessions[i].SessionID()

		if ids[id] {
			t.Errorf("Duplicate session ID: %s", id)
		}

		ids[id] = true
	}
}

func TestSocketSession_SessionID(t *testing.T) {
	t.Parallel()

	session := newSocketSession()
	id := session.SessionID()

	if id != session.id {
		t.Errorf("SessionID() = %s, want %s", id, session.id)
	}
}

func TestSocketSession_NotificationChannel(t *testing.T) {
	t.Parallel()

	session := newSocketSession()
	ch := session.NotificationChannel()

	if ch == nil {
		t.Fatal("NotificationChannel() returned nil")
	}

	// Verify we can send to the channel
	notification := mcp.JSONRPCNotification{
		JSONRPC: "2.0",
		Notification: mcp.Notification{
			Method: "test/notification",
		},
	}

	select {
	case ch <- notification:
		// Success
	default:
		t.Error("NotificationChannel() should accept notifications")
	}
}

func TestSocketSession_Initialize(t *testing.T) {
	t.Parallel()

	session := newSocketSession()

	if session.Initialized() {
		t.Error("Initialized() should be false before Initialize()")
	}

	session.Initialize()

	if !session.Initialized() {
		t.Error("Initialized() should be true after Initialize()")
	}

	// Verify default log level is set
	level := session.GetLogLevel()
	if level != mcp.LoggingLevelError {
		t.Errorf("GetLogLevel() = %v, want %v after Initialize()", level, mcp.LoggingLevelError)
	}
}

func TestSocketSession_LogLevel(t *testing.T) {
	t.Parallel()

	session := newSocketSession()

	// Default should be error level
	if level := session.GetLogLevel(); level != mcp.LoggingLevelError {
		t.Errorf("GetLogLevel() default = %v, want %v", level, mcp.LoggingLevelError)
	}

	// Set and get different levels
	testLevels := []mcp.LoggingLevel{
		mcp.LoggingLevelDebug,
		mcp.LoggingLevelInfo,
		mcp.LoggingLevelWarning,
		mcp.LoggingLevelError,
	}

	for _, level := range testLevels {
		session.SetLogLevel(level)

		if got := session.GetLogLevel(); got != level {
			t.Errorf("GetLogLevel() = %v, want %v", got, level)
		}
	}
}

func TestSocketSession_ClientInfo(t *testing.T) {
	t.Parallel()

	session := newSocketSession()

	// Default should be empty
	info := session.GetClientInfo()
	if info.Name != "" || info.Version != "" {
		t.Errorf("GetClientInfo() default = %+v, want empty", info)
	}

	// Set and get client info
	expected := mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
	session.SetClientInfo(expected)

	got := session.GetClientInfo()
	if got.Name != expected.Name || got.Version != expected.Version {
		t.Errorf("GetClientInfo() = %+v, want %+v", got, expected)
	}
}

func TestSocketSession_ClientCapabilities(t *testing.T) {
	t.Parallel()

	session := newSocketSession()

	// Default should be empty
	caps := session.GetClientCapabilities()
	if caps.Roots != nil {
		t.Errorf("GetClientCapabilities() default should be empty, got %+v", caps)
	}

	// Set and get capabilities - just verify round-trip works
	expected := mcp.ClientCapabilities{}
	session.SetClientCapabilities(expected)

	got := session.GetClientCapabilities()
	// Verify we can get back what we set (empty capabilities)
	if got.Roots != nil {
		t.Error("GetClientCapabilities() Roots should be nil")
	}
}

func TestSocketSession_Close(t *testing.T) {
	t.Parallel()

	session := newSocketSession()

	// Close should close the notification channel
	session.close()

	// Sending to closed channel should panic, so we test by receiving
	_, ok := <-session.notifications
	if ok {
		t.Error("notifications channel should be closed after close()")
	}
}
