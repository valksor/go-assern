// Package cli provides interactive CLI components for assern.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/valksor/go-assern/internal/config"
)

// transportConfigKind classifies a transport into the kind of configuration
// prompt flow it requires: "stdio", "http", or "" (none/unknown).
func transportConfigKind(transport string) string {
	switch transport {
	case transportStdio:
		return "stdio"
	case transportHTTP, transportSSE, transportOAuthHTTP, transportOAuthSSE:
		return "http"
	default:
		return ""
	}
}

// transportNeedsOAuth reports whether the given transport uses OAuth.
func transportNeedsOAuth(transport string) bool {
	switch transport {
	case transportOAuthHTTP, transportOAuthSSE:
		return true
	default:
		return false
	}
}

// promptTransport prompts for the transport type.
func promptTransport(input *MCPInput) error {
	// Skip if transport is already set (e.g., when editing)
	if input.Transport != "" {
		return nil
	}

	options := []string{transportStdio, transportHTTP, transportSSE, transportOAuthHTTP, transportOAuthSSE}

	var transport string
	if err := survey.AskOne(&survey.Select{
		Message: "Transport type:",
		Options: options,
		Default: transportStdio,
		Help:    "stdio: local subprocess\nhttp/sse: remote server\noauth-*: authenticated remote server",
	}, &transport, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	input.Transport = transport

	return nil
}

// promptTransportConfig prompts for transport-specific configuration.
func promptTransportConfig(input *MCPInput) error {
	switch transportConfigKind(input.Transport) {
	case "stdio":
		return promptStdioConfig(input)
	case "http":
		return promptHTTPConfig(input, transportNeedsOAuth(input.Transport))
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
		}, &key, survey.WithValidator(func(ans any) error {
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
		}, &input.URL, survey.WithValidator(func(ans any) error {
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
	}, &oauth.AuthServerMetadataURL, survey.WithValidator(func(ans any) error {
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
