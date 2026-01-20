package instance

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/valksor/go-assern/internal/config"
)

// DetectTimeout is the timeout for connecting to an existing instance.
// 500ms allows for slower systems while still being responsive.
const DetectTimeout = 500 * time.Millisecond

// Detector checks for running assern instances.
type Detector struct {
	logger *slog.Logger
}

// NewDetector creates a new instance detector.
func NewDetector(logger *slog.Logger) *Detector {
	return &Detector{logger: logger}
}

// DetectRunning checks if an assern instance is already running.
// Returns the instance info if found, nil if no instance is running.
// A nil return with nil error means no instance was detected (not an error condition).
//
//nolint:nilnil // Returning (nil, nil) is intentional - it means "no instance found"
func (d *Detector) DetectRunning() (*Info, error) {
	if !SharingEnabled() {
		d.logger.Debug("instance sharing disabled via environment")

		return nil, nil
	}

	socketPath, err := config.SocketPath()
	if err != nil {
		return nil, err
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
