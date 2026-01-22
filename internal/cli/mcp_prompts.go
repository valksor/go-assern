// Package cli provides interactive CLI components for assern.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-toolkit/project"
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
	}, &name, survey.WithValidator(func(ans interface{}) error {
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
	}, &name, survey.WithValidator(func(ans interface{}) error {
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

// promptTransport prompts for the transport type.
func promptTransport(input *MCPInput) error {
	// Skip if transport is already set (e.g., when editing)
	if input.Transport != "" {
		return nil
	}

	options := []string{"stdio", "http", "sse", "oauth-http", "oauth-sse"}

	var transport string
	if err := survey.AskOne(&survey.Select{
		Message: "Transport type:",
		Options: options,
		Default: "stdio",
		Help:    "stdio: local subprocess\nhttp/sse: remote server\noauth-*: authenticated remote server",
	}, &transport, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	input.Transport = transport

	return nil
}

// promptTransportConfig prompts for transport-specific configuration.
func promptTransportConfig(input *MCPInput) error {
	switch input.Transport {
	case "stdio":
		return promptStdioConfig(input)
	case "http", "sse":
		return promptHTTPConfig(input, false)
	case "oauth-http", "oauth-sse":
		return promptHTTPConfig(input, true)
	}

	return nil
}

// promptStdioConfig prompts for stdio transport configuration.
func promptStdioConfig(input *MCPInput) error {
	// Command
	if input.Command == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Command:",
			Help:    "Executable to run (e.g., npx, node, python)",
		}, &input.Command, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Args
	var addArgs bool
	if len(input.Args) == 0 {
		if err := survey.AskOne(&survey.Confirm{
			Message: "Add arguments?",
			Default: false,
		}, &addArgs); err != nil {
			return err
		}
		if addArgs {
			if err := promptArgs(input); err != nil {
				return err
			}
		}
	}

	// Working directory
	if input.WorkDir == "" {
		var addWorkDir bool
		if err := survey.AskOne(&survey.Confirm{
			Message: "Set working directory?",
			Default: false,
		}, &addWorkDir); err != nil {
			return err
		}
		if addWorkDir {
			if err := survey.AskOne(&survey.Input{
				Message: "Working directory:",
				Default: os.Getenv("PWD"),
			}, &input.WorkDir); err != nil {
				return err
			}
		}
	}

	// Environment variables
	if len(input.Env) == 0 {
		return promptEnvVars(input)
	}

	return nil
}

// promptArgs prompts for command arguments.
func promptArgs(input *MCPInput) error {
	args := []string{}
	for {
		var arg string
		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Argument %d (empty to finish):", len(args)+1),
		}, &arg); err != nil {
			return err
		}
		if arg == "" {
			break
		}
		args = append(args, arg)
	}
	input.Args = args

	return nil
}

// promptEnvVars prompts for environment variables.
func promptEnvVars(input *MCPInput) error {
	input.Env = make(map[string]string)

	var addEnv bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Add environment variables?",
		Default: false,
	}, &addEnv); err != nil {
		return err
	}

	if !addEnv {
		return nil
	}

	for {
		// Key
		var key string
		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Environment variable %d key (empty to finish):", len(input.Env)+1),
		}, &key, survey.WithValidator(func(ans interface{}) error {
			val, ok := ans.(string)
			if !ok {
				return errors.New("expected string value")
			}
			if val == "" {
				return nil // Allow empty to finish
			}

			return ValidateEnvVarKey(val)
		})); err != nil {
			return err
		}
		if key == "" {
			break
		}

		// Value
		var value string
		defaultValue := fmt.Sprintf("${%s}", key)
		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Value for %s:", key),
			Default: defaultValue,
		}, &value); err != nil {
			return err
		}

		input.Env[key] = value
	}

	return nil
}

// promptHTTPConfig prompts for HTTP/SSE transport configuration.
func promptHTTPConfig(input *MCPInput, useOAuth bool) error {
	// URL
	if input.URL == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Server URL:",
			Help:    "e.g., https://api.example.com/mcp",
		}, &input.URL, survey.WithValidator(func(ans interface{}) error {
			val, ok := ans.(string)
			if !ok {
				return errors.New("expected string value")
			}
			if useOAuth {
				return ValidateHTTPSURL(val)
			}

			return ValidateURL(val)
		})); err != nil {
			return err
		}
	}

	// Headers (skip for OAuth)
	if !useOAuth && len(input.Headers) == 0 {
		var addHeaders bool
		if err := survey.AskOne(&survey.Confirm{
			Message: "Add HTTP headers (API keys, etc.)?",
			Default: false,
		}, &addHeaders); err != nil {
			return err
		}
		if addHeaders {
			if err := promptHeaders(input); err != nil {
				return err
			}
		}
	}

	// OAuth configuration
	if useOAuth && input.OAuth == nil {
		return promptOAuthConfig(input)
	}

	return nil
}

// promptHeaders prompts for HTTP headers.
func promptHeaders(input *MCPInput) error {
	input.Headers = make(map[string]string)

	for {
		// Key
		var key string
		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Header %d key (empty to finish):", len(input.Headers)+1),
			Help:    "e.g., Authorization, X-API-Key",
		}, &key); err != nil {
			return err
		}
		if key == "" {
			break
		}

		// Value
		var value string
		if err := survey.AskOne(&survey.Input{
			Message: fmt.Sprintf("Value for %s:", key),
			Help:    "e.g., Bearer ${API_TOKEN}",
		}, &value); err != nil {
			return err
		}

		input.Headers[key] = value
	}

	return nil
}

// promptOAuthConfig prompts for OAuth configuration.
func promptOAuthConfig(input *MCPInput) error {
	oauth := &config.OAuthConfig{}

	// Client ID
	if err := survey.AskOne(&survey.Input{
		Message: "OAuth Client ID:",
	}, &oauth.ClientID, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Client Secret (optional for PKCE)
	var hasSecret bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Does your OAuth client have a secret?",
		Default: true,
	}, &hasSecret); err != nil {
		return err
	}

	if hasSecret {
		if err := survey.AskOne(&survey.Password{
			Message: "OAuth Client Secret:",
		}, &oauth.ClientSecret); err != nil {
			return err
		}
	}

	// Redirect URI
	if err := survey.AskOne(&survey.Input{
		Message: "Redirect URI (optional):",
		Default: "http://localhost:8080/callback",
	}, &oauth.RedirectURI); err != nil {
		return err
	}

	// Scopes
	var scopesStr string
	if err := survey.AskOne(&survey.Input{
		Message: "OAuth scopes (comma-separated):",
		Help:    "e.g., read, write, admin",
	}, &scopesStr, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	oauth.Scopes = parseScopes(scopesStr)

	// Auth Server Metadata URL
	if err := survey.AskOne(&survey.Input{
		Message: "OAuth Authorization Server Metadata URL:",
		Help:    "RFC 9728 metadata URL, e.g., https://auth.example.com/.well-known/oauth-authorization-server",
	}, &oauth.AuthServerMetadataURL, survey.WithValidator(func(ans interface{}) error {
		val, ok := ans.(string)
		if !ok {
			return errors.New("expected string value")
		}

		return ValidateHTTPSURL(val)
	})); err != nil {
		return err
	}

	// PKCE
	if !hasSecret {
		if err := survey.AskOne(&survey.Confirm{
			Message: "Enable PKCE (recommended for public clients)?",
			Default: true,
		}, &oauth.PKCEEnabled); err != nil {
			return err
		}
	}

	input.OAuth = oauth

	return nil
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

// promptConfirmation shows a summary and asks for confirmation.
func promptConfirmation(input *MCPInput) error {
	// Display summary
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Name: %s\n", input.Name)
	fmt.Printf("  Scope: %s", input.Scope)
	if input.Scope == ScopeProject && input.Project != "" {
		fmt.Printf(" (%s)", input.Project)
	}
	fmt.Println()
	fmt.Printf("  Transport: %s\n", input.Transport)

	switch input.Transport {
	case "stdio":
		fmt.Printf("    Command: %s\n", input.Command)
		if len(input.Args) > 0 {
			fmt.Printf("    Args: %s\n", strings.Join(input.Args, " "))
		}
		if input.WorkDir != "" {
			fmt.Printf("    Working Dir: %s\n", input.WorkDir)
		}
	case "http", "sse", "oauth-http", "oauth-sse":
		fmt.Printf("    URL: %s\n", input.URL)
		if input.OAuth != nil {
			fmt.Printf("    OAuth: ClientID=%s, Scopes=%v\n", input.OAuth.ClientID, input.OAuth.Scopes)
		}
	}

	if input.Disabled {
		fmt.Println("  Status: disabled")
	}

	fmt.Println()

	var confirm string

	return survey.AskOne(&survey.Select{
		Message: "Save configuration?",
		Options: []string{"save", "edit", "cancel"},
		Default: "save",
	}, &confirm, survey.WithValidator(func(ans interface{}) error {
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

// SelectServer prompts to select a server from the list.
func SelectServer(serverNames []string, promptMsg string) (string, error) {
	if len(serverNames) == 0 {
		return "", errors.New("no servers available")
	}

	var selected string
	if err := survey.AskOne(&survey.Select{
		Message: promptMsg,
		Options: serverNames,
	}, &selected, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return selected, nil
}

// SelectServers prompts to select multiple servers.
func SelectServers(serverNames []string, promptMsg string) ([]string, error) {
	if len(serverNames) == 0 {
		return nil, errors.New("no servers available")
	}

	var selected []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: promptMsg,
		Options: serverNames,
	}, &selected, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	return selected, nil
}

// ConfirmDelete asks for confirmation before deleting.
func ConfirmDelete(names []string) error {
	fmt.Printf("\nAbout to delete %d server(s):\n", len(names))
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	var confirm bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Continue?",
		Default: false,
	}, &confirm); err != nil {
		return err
	}

	if !confirm {
		return errors.New("cancelled by user")
	}

	return nil
}

// detectProject attempts to detect the current project.
func detectProject(cwd string) string {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return ""
	}

	// Create path resolver
	resolver := &configPathResolver{}
	registry := project.NewRegistry()
	for name, proj := range cfg.Projects {
		registry.Register(name, proj.Directories, nil)
	}

	detector := project.NewDetector(resolver, ".assern", registry)
	detector.SetConfigLoader(func(path string) (interface{}, error) {
		return config.LoadLocalProject(path)
	})

	ctx, err := detector.Detect(cwd)
	if err != nil {
		return ""
	}

	if ctx != nil {
		return ctx.Name
	}

	return ""
}

// configPathResolver adapts config functions to project.PathResolver.
type configPathResolver struct{}

func (r *configPathResolver) FindLocalConfigDir(startDir string) string {
	return config.FindLocalConfigDir(startDir)
}

func (r *configPathResolver) LocalConfigPath(localDir string) string {
	return config.LocalConfigPath(localDir)
}

func (r *configPathResolver) FileExists(path string) bool {
	return config.FileExists(path)
}

// parseScopes parses a comma-separated scopes string.
func parseScopes(scopesStr string) []string {
	scopes := strings.Split(scopesStr, ",")
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
