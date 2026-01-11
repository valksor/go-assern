package project

import (
	"path/filepath"
	"strings"

	"github.com/valksor/go-assern/internal/config"
)

// RegistryMatch represents a matched project from the registry.
type RegistryMatch struct {
	Name    string
	Pattern string
	Config  *config.ProjectConfig
}

// Registry handles project matching against the global project registry.
type Registry struct {
	projects map[string]*config.ProjectConfig
}

// NewRegistry creates a new project registry from the global config.
func NewRegistry(cfg *config.Config) *Registry {
	if cfg == nil {
		return &Registry{
			projects: make(map[string]*config.ProjectConfig),
		}
	}

	return &Registry{
		projects: cfg.Projects,
	}
}

// Match attempts to match a directory against registered projects.
// Returns the first matching project or nil if no match found.
func (r *Registry) Match(dir string) *RegistryMatch {
	if r.projects == nil {
		return nil
	}

	// Normalize the input directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}

	for name, proj := range r.projects {
		for _, pattern := range proj.Directories {
			if matchDirectory(absDir, pattern) {
				return &RegistryMatch{
					Name:    name,
					Pattern: pattern,
					Config:  proj,
				}
			}
		}
	}

	return nil
}

// List returns all registered project names.
func (r *Registry) List() []string {
	if r.projects == nil {
		return nil
	}

	names := make([]string, 0, len(r.projects))
	for name := range r.projects {
		names = append(names, name)
	}

	return names
}

// Get retrieves a project configuration by name.
func (r *Registry) Get(name string) *config.ProjectConfig {
	if r.projects == nil {
		return nil
	}

	return r.projects[name]
}

// matchDirectory checks if a directory matches a pattern.
// Supports:
// - Exact paths (after expansion): ~/work/project
// - Glob patterns: ~/work/acme/*
// - Double-star patterns: ~/repos/**.
func matchDirectory(dir, pattern string) bool {
	// Expand ~ in pattern
	expandedPattern := config.ExpandPath(pattern)

	// Handle glob patterns
	if strings.Contains(expandedPattern, "*") {
		return matchGlob(dir, expandedPattern)
	}

	// Exact match
	return dir == expandedPattern
}

// matchGlob matches a directory against a glob pattern.
func matchGlob(dir, pattern string) bool {
	// Handle ** (match any depth)
	if strings.Contains(pattern, "**") {
		return matchDoublestar(dir, pattern)
	}

	// Handle single * (match one level)
	if strings.HasSuffix(pattern, "/*") {
		// Pattern like ~/work/acme/* should match ~/work/acme/repo1
		basePattern := strings.TrimSuffix(pattern, "/*")

		// Check if dir is a direct child of basePattern
		dirParent := filepath.Dir(dir)

		return dirParent == basePattern
	}

	// Try standard glob matching
	matched, err := filepath.Match(pattern, dir)
	if err != nil {
		return false
	}

	return matched
}

// matchDoublestar matches a directory against a ** pattern.
func matchDoublestar(dir, pattern string) bool {
	// Split pattern at **
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		return false
	}

	prefix := parts[0]
	suffix := parts[1]

	// Remove trailing slash from prefix
	prefix = strings.TrimSuffix(prefix, "/")
	prefix = strings.TrimSuffix(prefix, string(filepath.Separator))

	// Check if dir starts with prefix
	if !strings.HasPrefix(dir, prefix) {
		return false
	}

	// If no suffix, match anything under prefix
	if suffix == "" || suffix == "/" {
		return true
	}

	// Check suffix match
	remainder := strings.TrimPrefix(dir, prefix)

	return strings.HasSuffix(remainder, suffix)
}

// FindProjectByDirectory searches for a project that contains the given directory.
func (r *Registry) FindProjectByDirectory(dir string) (string, *config.ProjectConfig) {
	match := r.Match(dir)
	if match == nil {
		return "", nil
	}

	return match.Name, match.Config
}

// AddDirectory adds a directory to a project's directory list.
func (r *Registry) AddDirectory(projectName, directory string) bool {
	proj, exists := r.projects[projectName]
	if !exists {
		return false
	}

	// Check if already exists
	for _, d := range proj.Directories {
		if config.ExpandPath(d) == directory {
			return false
		}
	}

	proj.Directories = append(proj.Directories, directory)

	return true
}
