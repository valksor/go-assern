package config

import (
	"github.com/valksor/go-toolkit/paths"
)

// Path configuration for assern.
var pathsConfig = &paths.Config{
	Vendor:   ".valksor",
	ToolName: "assern",
	LocalDir: ".assern",
}

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

// SetHomeDirForTesting overrides the home directory function for testing.
// Returns a restore function that should be deferred.
func SetHomeDirForTesting(dir string) func() {
	return paths.SetHomeDirForTesting(dir)
}

// GlobalDir returns the path to the global Assern configuration directory.
// Default: ~/.valksor/assern/.
func GlobalDir() (string, error) {
	return pathsConfig.GlobalDir()
}

// GlobalConfigPath returns the path to the global configuration file.
// Default: ~/.valksor/assern/config.yaml.
func GlobalConfigPath() (string, error) {
	return pathsConfig.GlobalConfigPath()
}

// GlobalEnvPath returns the path to the global environment file.
// Default: ~/.valksor/assern/.env.
func GlobalEnvPath() (string, error) {
	return pathsConfig.GlobalFilePath(GlobalEnvFile)
}

// GlobalMCPPath returns the path to the global MCP servers file.
// Default: ~/.valksor/assern/mcp.json.
func GlobalMCPPath() (string, error) {
	return pathsConfig.GlobalFilePath(GlobalMCPFile)
}

// FindLocalConfigDir searches for a .assern directory starting from the given
// directory and walking up to the filesystem root.
// Returns the path to the .assern directory if found, empty string otherwise.
func FindLocalConfigDir(startDir string) string {
	return pathsConfig.FindLocalConfigDir(startDir)
}

// LocalConfigPath returns the path to the local config file within a .assern directory.
func LocalConfigPath(assernDir string) string {
	return pathsConfig.LocalConfigPath(assernDir)
}

// LocalMCPPath returns the path to the local MCP servers file within a .assern directory.
func LocalMCPPath(assernDir string) string {
	return pathsConfig.LocalFilePath(assernDir, LocalMCPFile)
}

// EnsureGlobalDir creates the global configuration directory if it doesn't exist.
func EnsureGlobalDir() (string, error) {
	return pathsConfig.EnsureGlobalDir()
}

// EnsureLocalDir creates the local .assern directory in the given path.
func EnsureLocalDir(baseDir string) (string, error) {
	return pathsConfig.EnsureLocalDir(baseDir)
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	return paths.FileExists(path)
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	return paths.DirExists(path)
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	return paths.ExpandPath(path)
}
