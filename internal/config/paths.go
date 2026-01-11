package config

import (
	"os"
	"path/filepath"
)

// homeDirFunc is used to get the home directory. Can be overridden in tests.
var homeDirFunc = os.UserHomeDir

const (
	// GlobalConfigDir is the directory name for global Assern configuration.
	GlobalConfigDir = "assern"
	// GlobalConfigVendor is the vendor directory under home.
	GlobalConfigVendor = ".valksor"
	// GlobalConfigFile is the name of the global configuration file.
	GlobalConfigFile = "config.yaml"
	// GlobalMCPFile is the name of the global MCP servers file.
	GlobalMCPFile = "mcp.json"
	// GlobalEnvFile is the name of the global environment file.
	GlobalEnvFile = ".env"

	// LocalConfigDir is the directory name for project-local configuration.
	LocalConfigDir = ".assern"
	// LocalConfigFile is the name of the local configuration file.
	LocalConfigFile = "config.yaml"
	// LocalMCPFile is the name of the local MCP servers file.
	LocalMCPFile = "mcp.json"
)

// GlobalDir returns the path to the global Assern configuration directory.
// Default: ~/.valksor/assern/.
func GlobalDir() (string, error) {
	home, err := homeDirFunc()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, GlobalConfigVendor, GlobalConfigDir), nil
}

// GlobalConfigPath returns the path to the global configuration file.
// Default: ~/.valksor/assern/config.yaml.
func GlobalConfigPath() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, GlobalConfigFile), nil
}

// GlobalEnvPath returns the path to the global environment file.
// Default: ~/.valksor/assern/.env.
func GlobalEnvPath() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, GlobalEnvFile), nil
}

// GlobalMCPPath returns the path to the global MCP servers file.
// Default: ~/.valksor/assern/mcp.json.
func GlobalMCPPath() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, GlobalMCPFile), nil
}

// FindLocalConfigDir searches for a .assern directory starting from the given
// directory and walking up to the filesystem root.
// Returns the path to the .assern directory if found, empty string otherwise.
func FindLocalConfigDir(startDir string) string {
	dir := startDir

	for {
		candidate := filepath.Join(dir, LocalConfigDir)
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

// LocalConfigPath returns the path to the local config file within a .assern directory.
func LocalConfigPath(assernDir string) string {
	return filepath.Join(assernDir, LocalConfigFile)
}

// LocalMCPPath returns the path to the local MCP servers file within a .assern directory.
func LocalMCPPath(assernDir string) string {
	return filepath.Join(assernDir, LocalMCPFile)
}

// EnsureGlobalDir creates the global configuration directory if it doesn't exist.
func EnsureGlobalDir() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return dir, nil
}

// EnsureLocalDir creates the local .assern directory in the given path.
func EnsureLocalDir(baseDir string) (string, error) {
	dir := filepath.Join(baseDir, LocalConfigDir)

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
