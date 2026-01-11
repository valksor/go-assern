package project

import (
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/valksor/go-assern/internal/config"
)

// envVarPattern matches ${VAR} or $VAR patterns.
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// EnvLoader handles loading and expanding environment variables.
type EnvLoader struct {
	// Base environment (typically os.Environ())
	base map[string]string
	// Global .env file variables
	global map[string]string
	// Project .env file variables
	project map[string]string
}

// NewEnvLoader creates a new environment loader.
func NewEnvLoader() *EnvLoader {
	return &EnvLoader{
		base:    envToMap(os.Environ()),
		global:  make(map[string]string),
		project: make(map[string]string),
	}
}

// LoadGlobalEnv loads the global .env file.
func (e *EnvLoader) LoadGlobalEnv() error {
	path, err := config.GlobalEnvPath()
	if err != nil {
		return err
	}

	if !config.FileExists(path) {
		return nil
	}

	vars, err := godotenv.Read(path)
	if err != nil {
		return err
	}

	e.global = vars

	return nil
}

// ApplyProjectEnv sets project-level environment variables from config.
// Note: Assern does not read .env files from project directories.
// Project env vars are defined in config.yaml and applied here.
func (e *EnvLoader) ApplyProjectEnv(env map[string]string) {
	if env == nil {
		return
	}

	for k, v := range env {
		e.project[k] = v
	}
}

// Expand expands environment variable references in a string.
// Supports ${VAR} and $VAR syntax.
// Resolution order: project env → global env → system env.
func (e *EnvLoader) Expand(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		var varName string

		if strings.HasPrefix(match, "${") {
			varName = strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		} else {
			varName = strings.TrimPrefix(match, "$")
		}

		// Resolution order: project → global → base (system)
		if val, ok := e.project[varName]; ok {
			return val
		}

		if val, ok := e.global[varName]; ok {
			return val
		}

		if val, ok := e.base[varName]; ok {
			return val
		}

		// Return original if not found
		return match
	})
}

// ExpandMap expands all values in a map.
func (e *EnvLoader) ExpandMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = e.Expand(v)
	}

	return result
}

// BuildServerEnv builds the final environment for a server.
// Combines expanded server env with project context.
func (e *EnvLoader) BuildServerEnv(serverEnv map[string]string, projectName string) []string {
	// Start with expanded server env
	expanded := e.ExpandMap(serverEnv)

	// Add ASSERN_PROJECT if we have a project context
	if projectName != "" {
		if expanded == nil {
			expanded = make(map[string]string)
		}

		expanded["ASSERN_PROJECT"] = projectName
	}

	// Convert to slice format for exec.Cmd
	return mapToEnv(expanded)
}

// GetCombinedEnv returns a combined view of all environment layers.
func (e *EnvLoader) GetCombinedEnv() map[string]string {
	result := make(map[string]string)

	// Layer in order: base → global → project
	for k, v := range e.base {
		result[k] = v
	}

	for k, v := range e.global {
		result[k] = v
	}

	for k, v := range e.project {
		result[k] = v
	}

	return result
}

// SetProjectEnv sets a project-level environment variable.
func (e *EnvLoader) SetProjectEnv(key, value string) {
	if e.project == nil {
		e.project = make(map[string]string)
	}

	e.project[key] = value
}

// SetGlobalEnv sets a global-level environment variable.
func (e *EnvLoader) SetGlobalEnv(key, value string) {
	if e.global == nil {
		e.global = make(map[string]string)
	}

	e.global[key] = value
}

// envToMap converts os.Environ() format (KEY=value) to a map.
func envToMap(environ []string) map[string]string {
	result := make(map[string]string, len(environ))

	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result
}

// mapToEnv converts a map to os.Environ() format (KEY=value).
func mapToEnv(m map[string]string) []string {
	result := make([]string, 0, len(m))

	for k, v := range m {
		result = append(result, k+"="+v)
	}

	return result
}

// MergeEnvSlices merges two environment slices, with later values taking precedence.
func MergeEnvSlices(base, override []string) []string {
	baseMap := envToMap(base)

	for _, env := range override {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			baseMap[parts[0]] = parts[1]
		}
	}

	return mapToEnv(baseMap)
}
