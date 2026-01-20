// Package instance provides instance detection and sharing for assern.
// It allows multiple assern invocations to share a single aggregator instance
// via Unix domain sockets, preventing cascade spawning when LLMs call other LLMs.
package instance

import (
	"os"
	"time"
)

// EnvNoSharing is the environment variable to disable instance sharing.
const EnvNoSharing = "ASSERN_NO_INSTANCE_SHARING"

// Info contains information about a running assern instance.
type Info struct {
	PID        int       `json:"pid"`
	SocketPath string    `json:"socket_path"`
	StartTime  time.Time `json:"start_time"`
	WorkDir    string    `json:"work_dir"`
}

// SharingEnabled returns true if instance sharing is enabled.
// Sharing is disabled when ASSERN_NO_INSTANCE_SHARING is set to any non-empty value.
func SharingEnabled() bool {
	return os.Getenv(EnvNoSharing) == ""
}
