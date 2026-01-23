package instance

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/valksor/go-assern/internal/config"
)

func TestNewDetector(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	detector := NewDetector(logger)

	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}

	if detector.logger != logger {
		t.Error("NewDetector() did not set logger correctly")
	}
}

func TestDetectRunning_NoSocket(t *testing.T) {
	// Set up mock home directory
	tmpDir := t.TempDir()
	restore := config.SetHomeDirForTesting(tmpDir)
	t.Cleanup(restore)

	// Ensure global dir exists
	_, err := config.EnsureGlobalDir()
	if err != nil {
		t.Fatalf("EnsureGlobalDir() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	detector := NewDetector(logger)

	info, err := detector.DetectRunning()
	if err != nil {
		t.Fatalf("DetectRunning() error = %v", err)
	}

	if info != nil {
		t.Errorf("DetectRunning() = %v, want nil when no socket exists", info)
	}
}

func TestDetectRunning_SharingDisabled(t *testing.T) {
	t.Setenv(EnvNoSharing, "1")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	detector := NewDetector(logger)

	info, err := detector.DetectRunning()
	if err != nil {
		t.Fatalf("DetectRunning() error = %v", err)
	}

	if info != nil {
		t.Errorf("DetectRunning() = %v, want nil when sharing is disabled", info)
	}
}

func TestDetectRunning_StaleSocket(t *testing.T) {
	// Set up mock home directory
	tmpDir := t.TempDir()
	restore := config.SetHomeDirForTesting(tmpDir)
	t.Cleanup(restore)

	// Ensure global dir exists
	globalDir, err := config.EnsureGlobalDir()
	if err != nil {
		t.Fatalf("EnsureGlobalDir() error = %v", err)
	}

	// Create a stale socket file (just a regular file, not an actual socket)
	socketPath := filepath.Join(globalDir, config.SocketFile)
	if err := os.WriteFile(socketPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	detector := NewDetector(logger)

	info, err := detector.DetectRunning()
	if err != nil {
		t.Fatalf("DetectRunning() error = %v", err)
	}

	if info != nil {
		t.Errorf("DetectRunning() = %v, want nil for stale socket", info)
	}

	// Stale socket should be cleaned up
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Stale socket should have been removed")
	}
}

func TestDetectTimeout_Value(t *testing.T) {
	t.Parallel()

	if DetectTimeout <= 0 {
		t.Errorf("DetectTimeout = %v, should be positive", DetectTimeout)
	}

	// Should be reasonable for responsiveness
	if DetectTimeout > 2*time.Second {
		t.Errorf("DetectTimeout = %v, should be <= 2 seconds", DetectTimeout)
	}
}

func TestDetectRunning_WithRunningServer(t *testing.T) {
	// Use /tmp directly to keep socket path short (macOS has 108 char limit)
	// t.TempDir() creates paths too long for Unix sockets
	tmpDir, err := os.MkdirTemp("/tmp", "assern-test-") //nolint:usetesting
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	restore := config.SetHomeDirForTesting(tmpDir)
	t.Cleanup(restore)

	// Create short socket path directly
	socketPath := filepath.Join(tmpDir, "s.sock")
	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	srv := NewServer(socketPath, mcpServer, nil, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("Server.Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Create detector that uses our socket path
	// We need to override the socket path for testing
	detector := &Detector{logger: logger}
	info, err := detectWithSocketPath(detector, socketPath)
	if err != nil {
		t.Fatalf("detectWithSocketPath() error = %v", err)
	}

	if info == nil {
		t.Fatal("detectWithSocketPath() returned nil, expected instance info")
	}

	if info.PID != os.Getpid() {
		t.Errorf("detectWithSocketPath() info.PID = %d, want %d", info.PID, os.Getpid())
	}

	if info.SocketPath != socketPath {
		t.Errorf("detectWithSocketPath() info.SocketPath = %s, want %s", info.SocketPath, socketPath)
	}
}

func TestDetectRunning_ServerStoppedMidDetection(t *testing.T) {
	// Use /tmp directly to keep socket path short (macOS has 108 char limit)
	tmpDir, err := os.MkdirTemp("/tmp", "assern-test-") //nolint:usetesting
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	socketPath := filepath.Join(tmpDir, "s.sock")
	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, nil, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("Server.Start() error = %v", err)
	}

	// Stop the server immediately (simulates crash)
	_ = srv.Stop()

	// Detection should return nil (stale socket cleaned up)
	detector := &Detector{logger: logger}
	info, err := detectWithSocketPath(detector, socketPath)
	if err != nil {
		t.Fatalf("detectWithSocketPath() error = %v", err)
	}

	if info != nil {
		t.Errorf("detectWithSocketPath() = %v, want nil for stopped server", info)
	}
}

func TestDetectRunning_MultipleDetections(t *testing.T) {
	// Use /tmp directly to keep socket path short (macOS has 108 char limit)
	tmpDir, err := os.MkdirTemp("/tmp", "assern-test-") //nolint:usetesting
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	socketPath := filepath.Join(tmpDir, "s.sock")
	mcpServer := server.NewMCPServer("test", "1.0.0")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := NewServer(socketPath, mcpServer, nil, logger)
	if err := srv.Start(); err != nil {
		t.Fatalf("Server.Start() error = %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Run multiple detections concurrently
	const numDetections = 10
	errCh := make(chan error, numDetections)

	for range numDetections {
		go func() {
			detector := &Detector{logger: logger}
			info, err := detectWithSocketPath(detector, socketPath)
			if err != nil {
				errCh <- err

				return
			}
			if info == nil {
				errCh <- errors.New("detectWithSocketPath() returned nil")

				return
			}
			errCh <- nil
		}()
	}

	for range numDetections {
		if err := <-errCh; err != nil {
			t.Error(err)
		}
	}
}

// detectWithSocketPath is a test helper that performs detection with a specific socket path.
// This allows testing without relying on config.SocketPath() which can create paths too long for Unix sockets.
//
//nolint:nilnil // Returning (nil, nil) is intentional - it means "no instance found"
func detectWithSocketPath(d *Detector, socketPath string) (*Info, error) {
	if !SharingEnabled() {
		d.logger.Debug("instance sharing disabled via environment")

		return nil, nil
	}

	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		d.logger.Debug("no socket file found", "path", socketPath)

		return nil, nil
	}

	// Try to connect to verify it's alive
	ctx, cancel := context.WithTimeout(context.Background(), DetectTimeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		// Socket exists but can't connect - likely stale
		d.logger.Debug("socket exists but connection failed, cleaning up", "path", socketPath, "error", err)
		if removeErr := os.Remove(socketPath); removeErr != nil {
			d.logger.Debug("failed to remove stale socket", "error", removeErr)
		}

		return nil, nil
	}
	defer func() { _ = conn.Close() }()

	// Send ping request
	pingReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "assern/ping",
	}
	if err := json.NewEncoder(conn).Encode(pingReq); err != nil {
		d.logger.Debug("failed to send ping", "error", err)

		return nil, nil
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(DetectTimeout)); err != nil {
		d.logger.Debug("failed to set read deadline", "error", err)

		return nil, nil
	}

	// Read response
	var resp struct {
		Result *Info `json:"result"`
	}
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		d.logger.Debug("failed to read ping response", "error", err)

		return nil, nil
	}

	if resp.Result == nil {
		d.logger.Debug("empty ping response")

		return nil, nil
	}

	d.logger.Debug("found running instance",
		"pid", resp.Result.PID,
		"socket", resp.Result.SocketPath,
	)

	return resp.Result, nil
}
