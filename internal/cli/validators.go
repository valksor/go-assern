// Package cli provides interactive CLI components for assern.
package cli

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// reservedNames are server names that cannot be used.
var reservedNames = map[string]bool{
	"all":     true,
	"global":  true,
	"project": true,
}

// serverNameRegex validates server names: alphanumeric with hyphens/underscores.
var serverNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// ValidateServerName checks if a server name is valid.
func ValidateServerName(name string) error {
	if name == "" {
		return errors.New("server name cannot be empty")
	}

	if len(name) > 64 {
		return errors.New("server name must be 64 characters or less")
	}

	if reservedNames[strings.ToLower(name)] {
		return fmt.Errorf("'%s' is a reserved server name", name)
	}

	if !serverNameRegex.MatchString(name) {
		return errors.New("server name must start with a letter and contain only letters, numbers, hyphens, and underscores")
	}

	return nil
}

// ValidateURL checks if a string is a valid URL.
func ValidateURL(u string) error {
	if u == "" {
		return errors.New("URL cannot be empty")
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme == "" {
		return errors.New("URL must include a scheme (http:// or https://)")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("URL scheme must be http or https")
	}

	if parsed.Host == "" {
		return errors.New("URL must include a host")
	}

	return nil
}

// ValidateHTTPSURL checks if a string is a valid HTTPS URL.
func ValidateHTTPSURL(u string) error {
	if err := ValidateURL(u); err != nil {
		return err
	}

	parsed, _ := url.Parse(u)
	if parsed.Scheme != "https" {
		return errors.New("OAuth requires HTTPS URL")
	}

	return nil
}

// ValidateEnvVarKey checks if an environment variable key is valid.
func ValidateEnvVarKey(key string) error {
	if key == "" {
		return errors.New("environment variable key cannot be empty")
	}

	// Env var keys should be alphanumeric with underscores, typically uppercase
	envVarRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !envVarRegex.MatchString(key) {
		return errors.New("environment variable key must contain only letters, numbers, and underscores")
	}

	return nil
}

// ValidateRequired checks if a required field is empty.
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	return nil
}

// ValidateTransport checks if a transport type is valid.
func ValidateTransport(transport string) error {
	validTransports := map[string]bool{
		"stdio":      true,
		"http":       true,
		"sse":        true,
		"oauth-http": true,
		"oauth-sse":  true,
		"":           true, // Auto-detect
	}

	if !validTransports[transport] {
		return fmt.Errorf("invalid transport type: %s (must be stdio, http, sse, oauth-http, or oauth-sse)", transport)
	}

	return nil
}

// IsReservedName checks if a name is reserved.
func IsReservedName(name string) bool {
	return reservedNames[strings.ToLower(name)]
}
