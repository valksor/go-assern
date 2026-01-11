// Package version provides build version information.
package version

import (
	"fmt"
	"runtime"
)

// Build information set via ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf("assern %s\n  Commit: %s\n  Built:  %s\n  Go:     %s",
		Version, Commit, BuildTime, runtime.Version())
}

// Short returns just the version string.
func Short() string {
	return Version
}
