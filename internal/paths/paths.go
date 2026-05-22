// Package paths provides path resolution utilities for Valksor tools.
//
// This package handles XDG directory support, home directory expansion,
// and provides a configurable way to manage global and local configuration paths.
//
// Basic usage:
//
//	cfg := &paths.Config{
//	    Vendor:   "valksor",
//	    ToolName: "mehrhof",
//	    LocalDir: ".mehrhof",
//	}
//	globalPath, _ := cfg.GlobalConfigPath()  // ~/.valksor/mehrhof/config.yaml
package paths

import (
	"os"
	"path/filepath"
)

// homeDirFunc is used to get the home directory. Can be overridden in tests.
var homeDirFunc = os.UserHomeDir

// SetHomeDirForTesting overrides the home directory function for testing.
// Returns a restore function that should be deferred.
func SetHomeDirForTesting(dir string) func() {
	original := homeDirFunc
	homeDirFunc = func() (string, error) { return dir, nil }

	return func() { homeDirFunc = original }
}

// Config holds the configuration for path resolution.
type Config struct {
	// Vendor is the vendor directory name (e.g., "valksor").
	Vendor string
	// ToolName is the tool name (e.g., "mehrhof", "assern").
	ToolName string
	// LocalDir is the local config directory name (e.g., ".mehrhof", ".assern").
	LocalDir string
}

// GlobalDir returns the path to the global configuration directory.
// Example: ~/.valksor/mehrhof/.
func (c *Config) GlobalDir() (string, error) {
	home, err := homeDirFunc()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, c.Vendor, c.ToolName), nil
}

// GlobalConfigPath returns the path to the global configuration file.
// Example: ~/.valksor/mehrhof/config.yaml.
func (c *Config) GlobalConfigPath() (string, error) {
	dir, err := c.GlobalDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.yaml"), nil
}

// FindLocalConfigDir searches for a local config directory (e.g., .mehrhof)
// starting from the given directory and walking up to the filesystem root.
// Returns the path to the directory if found, empty string otherwise.
func (c *Config) FindLocalConfigDir(startDir string) string {
	dir := startDir

	for {
		candidate := filepath.Join(dir, c.LocalDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}

		dir = parent
	}
}

// LocalConfigPath returns the path to the local config file within a config directory.
// Example: /path/to/project/.mehrhof/config.yaml.
func (c *Config) LocalConfigPath(localDir string) string {
	return filepath.Join(localDir, "config.yaml")
}

// GlobalFilePath returns the path to a file in the global configuration directory.
// Example: cfg.GlobalFilePath("mcp.json") returns ~/.valksor/mehrhof/mcp.json.
func (c *Config) GlobalFilePath(filename string) (string, error) {
	dir, err := c.GlobalDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, filename), nil
}

// LocalFilePath returns the path to a file in a local config directory.
// Example: cfg.LocalFilePath(localDir, "mcp.json") returns /path/to/project/.mehrhof/mcp.json.
func (c *Config) LocalFilePath(localDir, filename string) string {
	return filepath.Join(localDir, filename)
}

// EnsureGlobalDir creates the global configuration directory if it doesn't exist.
func (c *Config) EnsureGlobalDir() (string, error) {
	dir, err := c.GlobalDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return dir, nil
}

// EnsureLocalDir creates the local config directory in the given path.
func (c *Config) EnsureLocalDir(baseDir string) (string, error) {
	dir := filepath.Join(baseDir, c.LocalDir)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return dir, nil
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if len(path) == 0 {
		return path
	}

	if path[0] != '~' {
		return path
	}

	home, err := homeDirFunc()
	if err != nil {
		return path
	}

	if len(path) == 1 {
		return home
	}

	if path[1] == '/' || path[1] == filepath.Separator {
		return filepath.Join(home, path[2:])
	}

	return path
}
