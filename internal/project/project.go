// Package project provides project detection and context management.
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/valksor/go-assern/internal/config"
)

// Context represents the resolved project context.
type Context struct {
	// Name is the project name (from local config or registry match).
	Name string
	// Directory is the project root directory.
	Directory string
	// LocalConfigDir is the path to .assern directory if it exists.
	LocalConfigDir string
	// LocalConfig is the parsed local project configuration.
	LocalConfig *config.LocalProjectConfig
	// Source indicates how the project was detected.
	Source DetectionSource
}

// DetectionSource indicates how the project was detected.
type DetectionSource string

const (
	// SourceLocal means project was detected from local .assern/config.yaml.
	SourceLocal DetectionSource = "local"
	// SourceRegistry means project was matched from global registry.
	SourceRegistry DetectionSource = "registry"
	// SourceExplicit means project was explicitly specified via flag.
	SourceExplicit DetectionSource = "explicit"
	// SourceAutoDetect means project name was auto-detected from directory name.
	SourceAutoDetect DetectionSource = "auto"
	// SourceNone means no project context was detected.
	SourceNone DetectionSource = "none"
)

// Detector handles project detection logic.
type Detector struct {
	globalConfig *config.Config
	registry     *Registry
}

// NewDetector creates a new project detector.
func NewDetector(globalConfig *config.Config) *Detector {
	return &Detector{
		globalConfig: globalConfig,
		registry:     NewRegistry(globalConfig),
	}
}

// Detect attempts to detect the project context from the given directory.
// Detection priority:
// 1. Local .assern/config.yaml with explicit project name
// 2. Local .assern directory (use directory name as project name)
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

	// Step 1: Look for local .assern directory
	localDir := config.FindLocalConfigDir(absDir)
	if localDir != "" {
		ctx.LocalConfigDir = localDir
		ctx.Directory = filepath.Dir(localDir)

		// Try to load local config
		configPath := config.LocalConfigPath(localDir)
		if config.FileExists(configPath) {
			localCfg, err := config.LoadLocalProject(configPath)
			if err != nil {
				return nil, fmt.Errorf("loading local project config: %w", err)
			}

			ctx.LocalConfig = localCfg

			// Use explicit project name if specified
			if localCfg.Project != "" {
				ctx.Name = localCfg.Project
				ctx.Source = SourceLocal

				return ctx, nil
			}
		}

		// No explicit project name, use directory name
		ctx.Name = filepath.Base(ctx.Directory)
		ctx.Source = SourceLocal

		return ctx, nil
	}

	// Step 2: Try to match against global registry
	if d.registry != nil {
		if match := d.registry.Match(absDir); match != nil {
			ctx.Name = match.Name
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
		return nil, errors.New("no project context detected; use --project flag or create .assern/config.yaml")
	}

	return ctx, nil
}

// AutoRegister registers a newly detected local project to the global config.
func (d *Detector) AutoRegister(ctx *Context) error {
	if ctx == nil || ctx.Source != SourceLocal || ctx.Name == "" {
		return nil
	}

	// Check if already registered
	if d.globalConfig.Projects != nil {
		if proj, exists := d.globalConfig.Projects[ctx.Name]; exists {
			// Check if directory is already in the list
			for _, dir := range proj.Directories {
				expandedDir := config.ExpandPath(dir)
				if expandedDir == ctx.Directory {
					return nil // Already registered
				}
			}
		}
	}

	// Register the project
	d.globalConfig.RegisterProject(ctx.Name, ctx.Directory)

	// Save the updated global config
	globalPath, err := config.GlobalConfigPath()
	if err != nil {
		return fmt.Errorf("getting global config path: %w", err)
	}

	if err := d.globalConfig.Save(globalPath); err != nil {
		return fmt.Errorf("saving global config: %w", err)
	}

	return nil
}
