// Package cli provides interactive CLI components for assern.
package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/project"
)

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
	detector.SetConfigLoader(func(path string) (any, error) {
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
