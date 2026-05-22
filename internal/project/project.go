// Package project provides project detection and context management.
//
// This package allows detecting project context based on directory patterns
// and local configuration directories. It's useful for tools that need to
// understand which project the user is currently working in.
//
// Basic usage:
//
//	detector := project.NewDetector(cfg, ".mehrhof")
//	ctx, err := detector.DetectFromCwd()
//	fmt.Printf("Project: %s\n", ctx.Name)
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Context represents the resolved project context.
type Context struct {
	// Name is the project name.
	Name string
	// Directory is the project root directory.
	Directory string
	// LocalConfigDir is the path to local config directory if it exists.
	LocalConfigDir string
	// Config is the parsed local project configuration (if any).
	Config any
	// Source indicates how the project was detected.
	Source DetectionSource
}

// DetectionSource indicates how the project was detected.
type DetectionSource string

const (
	// SourceLocal means project was detected from local config directory.
	SourceLocal DetectionSource = "local"
	// SourceRegistry means project was matched from registry.
	SourceRegistry DetectionSource = "registry"
	// SourceExplicit means project was explicitly specified via flag.
	SourceExplicit DetectionSource = "explicit"
	// SourceAutoDetect means project name was auto-detected from directory name.
	SourceAutoDetect DetectionSource = "auto"
	// SourceNone means no project context was detected.
	SourceNone DetectionSource = "none"
)

// PathResolver is an interface for resolving config paths.
type PathResolver interface {
	FindLocalConfigDir(startDir string) string
	LocalConfigPath(localDir string) string
	FileExists(path string) bool
}

// ConfigLoader is an interface for loading local project config.
type ConfigLoader func(path string) (any, error)

// Detector handles project detection logic.
type Detector struct {
	pathResolver PathResolver
	configLoader ConfigLoader
	registry     *Registry
	localDirName string
}

// NewDetector creates a new project detector.
func NewDetector(pathResolver PathResolver, localDirName string, registry *Registry) *Detector {
	return &Detector{
		pathResolver: pathResolver,
		localDirName: localDirName,
		registry:     registry,
	}
}

// SetConfigLoader sets a custom config loader function.
func (d *Detector) SetConfigLoader(loader ConfigLoader) {
	d.configLoader = loader
}

// Detect attempts to detect the project context from the given directory.
// Detection priority:
// 1. Local config directory with explicit project name
// 2. Local config directory (use directory name as project name)
// 3. Match directory against global registry.
func (d *Detector) Detect(dir string) (*Context, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving directory path: %w", err)
	}

	ctx := &Context{
		Directory: absDir,
		Source:    SourceNone,
	}

	// Step 1: Look for local config directory
	if d.pathResolver != nil {
		localDir := d.pathResolver.FindLocalConfigDir(absDir)
		if localDir != "" {
			ctx.LocalConfigDir = localDir
			ctx.Directory = filepath.Dir(localDir)

			// Try to load local config if loader is set
			if d.configLoader != nil {
				configPath := d.pathResolver.LocalConfigPath(localDir)
				if d.pathResolver.FileExists(configPath) {
					localCfg, err := d.configLoader(configPath)
					if err != nil {
						return nil, fmt.Errorf("loading local project config: %w", err)
					}

					ctx.Config = localCfg

					// Try to get project name from config
					if name, ok := getProjectName(localCfg); ok && name != "" {
						ctx.Name = name
						ctx.Source = SourceLocal

						return ctx, nil
					}
				}
			}

			// No explicit project name, use directory name
			ctx.Name = filepath.Base(ctx.Directory)
			ctx.Source = SourceLocal

			return ctx, nil
		}
	}

	// Step 2: Try to match against global registry
	if d.registry != nil {
		if match := d.registry.Match(absDir); match != nil {
			ctx.Name = match.Name
			ctx.Config = match.Config
			ctx.Source = SourceRegistry

			return ctx, nil
		}
	}

	// Step 3: Auto-detect from directory basename
	ctx.Name = filepath.Base(absDir)
	ctx.Source = SourceAutoDetect

	return ctx, nil
}

// DetectWithExplicit detects project context, using explicit name if provided.
func (d *Detector) DetectWithExplicit(dir string, explicitProject string) (*Context, error) {
	if explicitProject != "" {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolving directory path: %w", err)
		}

		return &Context{
			Name:      explicitProject,
			Directory: absDir,
			Source:    SourceExplicit,
		}, nil
	}

	return d.Detect(dir)
}

// DetectFromCwd detects project context from the current working directory.
func (d *Detector) DetectFromCwd() (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}

	return d.Detect(cwd)
}

// RequireProject detects project and returns an error if none found.
func (d *Detector) RequireProject(dir string, explicitProject string) (*Context, error) {
	ctx, err := d.DetectWithExplicit(dir, explicitProject)
	if err != nil {
		return nil, err
	}

	if ctx.Source == SourceNone {
		return nil, errors.New("no project context detected")
	}

	return ctx, nil
}

// getProjectName attempts to extract a project name from a config object.
// This is a generic implementation that can be overridden by setting a custom config loader.
func getProjectName(cfg any) (string, bool) {
	// Try to access via interface{} type assertion to common patterns
	// This allows different config types to work without hardcoding

	// This won't match most types, but provides a hook for custom loaders
	// The actual implementation should be in the config loader
	return "", false
}
