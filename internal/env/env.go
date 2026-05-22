// Package env provides environment variable expansion utilities.
//
// This package supports ${VAR} and $VAR syntax for environment variable
// expansion in strings and maps, layered environment loading, and .env
// file loading support.
//
// Basic usage:
//
//	expanded := env.ExpandEnv("${HOME}/.config")
//	envMap := env.ExpandEnvInMap(map[string]string{"path": "${HOME}/docs"})
//
// Advanced usage with layered environment:
//
//	loader := env.NewLoader()
//	loader.LoadDotenv("/path/to/.env")
//	loader.SetLayer("project", map[string]string{"API_KEY": "xxx"})
//	expanded := loader.Expand("${API_KEY}/endpoint")
package env

import (
	"maps"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// ExpandEnv expands environment variable references in a string.
// It supports both ${VAR} and $VAR syntax.
// If a referenced variable is not set, it will be replaced with an empty string.
func ExpandEnv(s string) string {
	return os.ExpandEnv(s)
}

// ExpandEnvInMap expands environment variables in all map values.
// Returns a new map with expanded values; does not modify the input.
// If the input map is nil, returns nil.
func ExpandEnvInMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = os.ExpandEnv(v)
	}

	return result
}

// Getenv returns the value of the environment variable named by the key.
// If the variable is not set, it returns the defaultValue.
func Getenv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

// Loader provides layered environment variable loading and expansion.
// Layers are resolved in the order: base → global → project (highest priority).
type Loader struct {
	base    map[string]string // Typically os.Environ()
	global  map[string]string // Global .env file
	project map[string]string // Project-specific env vars
}

// NewLoader creates a new environment loader with base environment from os.Environ().
func NewLoader() *Loader {
	return &Loader{
		base:    environToMap(os.Environ()),
		global:  make(map[string]string),
		project: make(map[string]string),
	}
}

// LoadDotenv loads environment variables from a .env file into the global layer.
// If the file doesn't exist, returns nil (no error).
// Returns an error only if the file exists but cannot be parsed.
func (l *Loader) LoadDotenv(path string) error {
	vars, err := readDotEnv(path)
	if err != nil {
		return err
	}

	if vars != nil {
		l.global = vars
	}

	return nil
}

// SetLayer sets environment variables for a specific layer.
// Valid layer names: "base", "global", "project".
func (l *Loader) SetLayer(layer string, vars map[string]string) {
	if vars == nil {
		return
	}

	switch layer {
	case "base":
		l.base = vars
	case "global":
		l.global = vars
	case "project":
		l.project = vars
	}
}

// Set sets a single environment variable in the specified layer.
// Valid layer names: "base", "global", "project".
// If layer is empty, defaults to "project".
func (l *Loader) Set(layer, key, value string) {
	if layer == "" {
		layer = "project"
	}

	switch layer {
	case "base":
		if l.base == nil {
			l.base = make(map[string]string)
		}
		l.base[key] = value
	case "global":
		if l.global == nil {
			l.global = make(map[string]string)
		}
		l.global[key] = value
	case "project":
		if l.project == nil {
			l.project = make(map[string]string)
		}
		l.project[key] = value
	}
}

// Get retrieves an environment variable by key.
// Resolution order: project → global → base (highest to lowest priority).
func (l *Loader) Get(key string) string {
	if val, ok := l.project[key]; ok {
		return val
	}
	if val, ok := l.global[key]; ok {
		return val
	}
	if val, ok := l.base[key]; ok {
		return val
	}

	return ""
}

// Expand expands environment variable references in a string.
// Supports ${VAR} and $VAR syntax.
// Resolution order: project → global → base.
func (l *Loader) Expand(s string) string {
	return os.Expand(s, func(key string) string {
		return l.Get(key)
	})
}

// ExpandMap expands all values in a map.
// Returns a new map with expanded values; does not modify the input.
func (l *Loader) ExpandMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = l.Expand(v)
	}

	return result
}

// ToMap returns a combined view of all layers as a single map.
// Later layers take precedence: base → global → project.
func (l *Loader) ToMap() map[string]string {
	result := make(map[string]string)

	maps.Copy(result, l.base)
	maps.Copy(result, l.global)
	maps.Copy(result, l.project)

	return result
}

// ToSlice returns the combined environment as a slice of "KEY=value" strings.
// Suitable for use with os/exec.Cmd.Env.
func (l *Loader) ToSlice() []string {
	m := l.ToMap()
	result := make([]string, 0, len(m))

	for k, v := range m {
		result = append(result, k+"="+v)
	}

	return result
}

// BuildServerEnv builds an environment slice for a server process.
// It merges the base environment with server-specific environment variables.
// The serverEnv map is expanded (variable references like ${VAR} are resolved).
// If projectName is non-empty, it adds an ASSERN_PROJECT variable.
// Returns a slice in "KEY=value" format suitable for os/exec.Cmd.Env.
func (l *Loader) BuildServerEnv(serverEnv map[string]string, projectName string) []string {
	// Start with the base environment from all layers
	result := l.ToSlice()

	// Expand and merge server-specific environment
	if serverEnv != nil {
		// Create a map from the base for easier merging
		baseMap := l.ToMap()

		// Expand server env vars and merge into base
		expandedServerEnv := l.ExpandMap(serverEnv)
		maps.Copy(baseMap, expandedServerEnv)

		// Convert back to slice
		result = mapToEnviron(baseMap)
	}

	// Add project name if specified
	if projectName != "" {
		// Merge the project variable into the result
		result = mergeEnvSlice(result, []string{"ASSERN_PROJECT=" + projectName})
	}

	return result
}

// mergeEnvSlice merges two environment slices, with override taking precedence.
// Both slices should be in "KEY=value" format.
func mergeEnvSlice(base, override []string) []string {
	// Convert base to map for easier merging
	baseMap := environToMap(base)

	// Apply overrides
	for _, env := range override {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			baseMap[parts[0]] = parts[1]
		}
	}

	// Convert back to slice
	return mapToEnviron(baseMap)
}

// environToMap converts os.Environ() format ([]string{"KEY=value"}) to a map.
func environToMap(environ []string) map[string]string {
	result := make(map[string]string, len(environ))

	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result
}

// mapToEnviron converts a map to os.Environ() format ([]string{"KEY=value"}).
func mapToEnviron(m map[string]string) []string {
	result := make([]string, 0, len(m))

	for k, v := range m {
		result = append(result, k+"="+v)
	}

	return result
}

// readDotEnv reads a .env file and returns a map of key-value pairs.
// Returns empty map if the file doesn't exist (no error).
// Returns an error if the file exists but cannot be parsed.
func readDotEnv(path string) (map[string]string, error) {
	// Check if file exists first
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	// Use godotenv to read the file
	return godotenv.Read(path)
}
