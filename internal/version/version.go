// Package version provides build version information.
//
// The version variables are set via ldflags at build time.
// This package provides functions to format and display version information.
package version

import (
	"fmt"
	"runtime"
)

// Build-time variables. These are set via ldflags:
//
//	-ldflags "-X github.com/valksor/go-assern/internal/version.Version=1.0.0 \
//	         -X github.com/valksor/go-assern/internal/version.Commit=abc123 \
//	         -X github.com/valksor/go-assern/internal/version.BuildTime=2024-01-15T12:00:00Z"
var (
	// Version is the application version (set via ldflags).
	Version = "dev"
	// Commit is the git commit hash (set via ldflags).
	Commit = "none"
	// BuildTime is the build timestamp (set via ldflags).
	BuildTime = "unknown"
)

// Info returns formatted version information for the given application name.
//
// Example output:
//
//	assern 1.0.0
//	  Commit: abc123
//	  Built:  2024-01-15T12:00:00Z
//	  Go:     go1.21.5
func Info(appName string) string {
	return fmt.Sprintf("%s %s\n  Commit: %s\n  Built:  %s\n  Go:     %s\n  by Valksor",
		appName, Version, Commit, BuildTime, runtime.Version())
}

// Short returns just the version string.
func Short() string {
	return Version
}

// Set allows setting version information (primarily for testing).
func Set(v, c, bt string) {
	Version = v
	Commit = c
	BuildTime = bt
}
