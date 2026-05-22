// Package cli provides interactive CLI components for assern.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/valksor/go-assern/internal/config"
)

// PromptForMCPServer runs the interactive prompt flow for adding/editing a server.
func PromptForMCPServer(existing *MCPInput) (*MCPInput, error) {
	input := &MCPInput{}
	if existing != nil {
		input = existing
	}

	// 1. Server name (skip if editing and keeping name)
	if existing == nil || existing.Name == "" {
		if err := promptName(input); err != nil {
			return nil, err
		}
	}

	// 2. Scope
	if err := promptScope(input); err != nil {
		return nil, err
	}

	// 3. Transport type
	if err := promptTransport(input); err != nil {
		return nil, err
	}

	// 4. Transport-specific configuration
	if err := promptTransportConfig(input); err != nil {
		return nil, err
	}

	// 5. Advanced options
	if err := promptAdvancedOptions(input); err != nil {
		return nil, err
	}

	// 6. Confirmation
	if err := promptConfirmation(input); err != nil {
		return nil, err
	}

	return input, nil
}

// promptName prompts for the server name.
func promptName(input *MCPInput) error {
	var name string
	if err := survey.AskOne(&survey.Input{
		Message: "Server name:",
		Help:    "Unique identifier for this server (alphanumeric, hyphens, underscores)",
	}, &name, survey.WithValidator(func(ans any) error {
		val, ok := ans.(string)
		if !ok {
			return errors.New("expected string value")
		}
		if err := ValidateServerName(val); err != nil {
			return err
		}

		return nil
	})); err != nil {
		return err
	}

	input.Name = name

	return nil
}

// promptScope prompts for the configuration scope.
func promptScope(input *MCPInput) error {
	// Skip if scope is already set (e.g., when editing)
	if input.Scope != "" {
		return nil
	}

	cwd, _ := os.Getwd()
	detectedProject := detectProject(cwd)

	var scope string
	err := survey.AskOne(&survey.Select{
		Message: "Configuration scope:",
		Options: []string{"global", "project-specific"},
		Default: "global",
		Help:    "Global: available in all projects\nProject-specific: only available in selected project",
	}, &scope, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	input.Scope = ScopeType(scope)

	if input.Scope == ScopeProject {
		return promptProjectSelection(input, detectedProject)
	}

	return nil
}

// promptProjectSelection prompts for project selection.
func promptProjectSelection(input *MCPInput, detected string) error {
	// Load existing projects
	cfg, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("loading projects: %w", err)
	}

	options := []string{"Create new project"}
	for name := range cfg.Projects {
		options = append(options, name)
	}

	// If we detected a project, confirm with user
	if detected != "" {
		msg := fmt.Sprintf("Auto-detected project: %s. Use this project?", detected)
		var useDetected bool
		if err := survey.AskOne(&survey.Confirm{
			Message: msg,
			Default: true,
		}, &useDetected); err != nil {
			return err
		}
		if useDetected {
			input.Project = detected

			return nil
		}
	}

	var selection string
	if err := survey.AskOne(&survey.Select{
		Message: "Select project:",
		Options: options,
	}, &selection, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	if selection == "Create new project" {
		return promptNewProject(input)
	}

	input.Project = selection

	return nil
}

// promptNewProject prompts for creating a new project.
func promptNewProject(input *MCPInput) error {
	var name string
	if err := survey.AskOne(&survey.Input{
		Message: "Project name:",
		Help:    "Unique identifier for this project",
	}, &name, survey.WithValidator(func(ans any) error {
		val, ok := ans.(string)
		if !ok {
			return errors.New("expected string value")
		}
		if strings.TrimSpace(val) == "" {
			return errors.New("project name cannot be empty")
		}

		return nil
	})); err != nil {
		return err
	}

	input.Project = name

	// Ask for directories
	var addDir bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Add project directories?",
		Default: true,
	}, &addDir); err != nil {
		return err
	}

	if addDir {
		dirs, err := promptDirectories()
		if err != nil {
			return err
		}
		// Create the project in global config
		cfg, _ := config.LoadGlobal()
		if cfg.Projects == nil {
			cfg.Projects = make(map[string]*config.ProjectConfig)
		}
		cfg.Projects[name] = &config.ProjectConfig{
			Directories: dirs,
		}
		configPath, _ := config.GlobalConfigPath()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("saving project config: %w", err)
		}
		fmt.Printf("Project '%s' created with directories: %v\n", name, dirs)
	}

	return nil
}

// promptDirectories prompts for project directories.
func promptDirectories() ([]string, error) {
	var dirs []string
	for {
		var dir string
		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Directory %d (empty to finish):", len(dirs)+1),
			Help:    "Supports globs like ~/work/*",
		}, &dir); err != nil {
			return nil, err
		}
		if strings.TrimSpace(dir) == "" {
			break
		}
		dirs = append(dirs, dir)
	}

	return dirs, nil
}

// promptAdvancedOptions prompts for advanced options.
func promptAdvancedOptions(input *MCPInput) error {
	var configureAdvanced bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Configure advanced options?",
		Default: false,
	}, &configureAdvanced); err != nil {
		return err
	}

	if !configureAdvanced {
		return nil
	}

	// Disabled
	if err := survey.AskOne(&survey.Confirm{
		Message: "Disable this server?",
		Default: false,
	}, &input.Disabled); err != nil {
		return err
	}

	// Merge mode (for project-specific)
	if input.Scope == ScopeProject {
		var mode string
		if err := survey.AskOne(&survey.Select{
			Message: "Environment merge mode:",
			Options: []string{"overlay", "replace"},
			Default: "overlay",
			Help:    "overlay: merge project env with server env\nreplace: use only project env",
		}, &mode); err != nil {
			return err
		}
		input.MergeMode = config.MergeMode(mode)
	}

	// Allowed tools (deferred - requires connecting to server first)
	var restrictTools bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Restrict to specific tools?",
		Default: false,
	}, &restrictTools); err != nil {
		return err
	}

	if restrictTools {
		fmt.Println("Note: Tool restriction requires connecting to the server first.")
		fmt.Println("You can edit the server later with 'assern mcp:edit' to set allowed tools.")
	}

	return nil
}

// buildSummaryLines builds the configuration summary lines printed during
// confirmation. It is a pure function so the formatting can be tested without
// a TTY; promptConfirmation prints the returned lines verbatim.
func buildSummaryLines(input *MCPInput) []string {
	lines := make([]string, 0, 8)

	lines = append(lines, "  Name: "+input.Name)

	scopeLine := fmt.Sprintf("  Scope: %s", input.Scope)
	if input.Scope == ScopeProject && input.Project != "" {
		scopeLine += fmt.Sprintf(" (%s)", input.Project)
	}
	lines = append(lines, scopeLine)

	lines = append(lines, "  Transport: "+input.Transport)

	switch input.Transport {
	case transportStdio:
		lines = append(lines, "    Command: "+input.Command)
		if len(input.Args) > 0 {
			lines = append(lines, "    Args: "+strings.Join(input.Args, " "))
		}
		if input.WorkDir != "" {
			lines = append(lines, "    Working Dir: "+input.WorkDir)
		}
	case transportHTTP, transportSSE, transportOAuthHTTP, transportOAuthSSE:
		lines = append(lines, "    URL: "+input.URL)
		if input.OAuth != nil {
			lines = append(lines, fmt.Sprintf("    OAuth: ClientID=%s, Scopes=%v", input.OAuth.ClientID, input.OAuth.Scopes))
		}
	}

	if input.Disabled {
		lines = append(lines, "  Status: disabled")
	}

	return lines
}

// promptConfirmation shows a summary and asks for confirmation.
func promptConfirmation(input *MCPInput) error {
	// Display summary
	fmt.Println("\nConfiguration Summary:")
	for _, line := range buildSummaryLines(input) {
		fmt.Println(line)
	}
	fmt.Println()

	var confirm string

	return survey.AskOne(&survey.Select{
		Message: "Save configuration?",
		Options: []string{"save", "edit", "cancel"},
		Default: "save",
	}, &confirm, survey.WithValidator(func(ans any) error {
		val, ok := ans.(string)
		if !ok {
			return errors.New("expected string value")
		}
		if val == "cancel" {
			return errors.New("cancelled by user")
		}
		if val == "edit" {
			return errors.New("edit requested")
		}

		return nil
	}))
}
