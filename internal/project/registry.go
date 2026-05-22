package project

import (
	"path/filepath"
	"strings"

	"github.com/valksor/go-assern/internal/paths"
)

// RegistryMatch represents a matched project from the registry.
type RegistryMatch struct {
	Name    string
	Pattern string
	Config  any
}

// Registry handles project matching against registered projects.
type Registry struct {
	projects map[string]*RegistryProject
}

// RegistryProject represents a project in the registry.
type RegistryProject struct {
	Name        string
	Directories []string
	Config      any
}

// NewRegistry creates a new project registry.
func NewRegistry() *Registry {
	return &Registry{
		projects: make(map[string]*RegistryProject),
	}
}

// NewRegistryFromMap creates a registry from a map of project configurations.
// The map key is the project name.
func NewRegistryFromMap(projects map[string]*RegistryProject) *Registry {
	return &Registry{
		projects: projects,
	}
}

// Register registers a project with the registry.
func (r *Registry) Register(name string, directories []string, config any) {
	if r.projects == nil {
		r.projects = make(map[string]*RegistryProject)
	}

	r.projects[name] = &RegistryProject{
		Name:        name,
		Directories: directories,
		Config:      config,
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
					Config:  proj.Config,
				}
			}
		}
	}

	return nil
}

// List returns all registered project names.
func (r *Registry) List() []string {
	if len(r.projects) == 0 {
		return nil
	}

	names := make([]string, 0, len(r.projects))
	for name := range r.projects {
		names = append(names, name)
	}

	return names
}

// Get retrieves a project configuration by name.
func (r *Registry) Get(name string) *RegistryProject {
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
	expandedPattern := paths.ExpandPath(pattern)

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
	if before, ok := strings.CutSuffix(pattern, "/*"); ok {
		// Pattern like ~/work/acme/* should match ~/work/acme/repo1
		basePattern := before

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
