package aggregator

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// PromptEntry represents a prompt from a backend server.
type PromptEntry struct {
	// ServerName is the name of the backend server.
	ServerName string
	// Prompt is the original prompt definition.
	Prompt mcp.Prompt
	// PrefixedName is the prompt name with server prefix.
	PrefixedName string
}

// PromptRegistry manages the mapping of prefixed prompt names to backend servers.
type PromptRegistry struct {
	// Use the generic registry with entry pointer and string key
	r *registry[*PromptEntry, string]
}

// NewPromptRegistry creates a new prompt registry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		r: newRegistry[*PromptEntry, string](),
	}
}

// Register adds a prompt from a server to the registry.
func (r *PromptRegistry) Register(serverName string, prompt mcp.Prompt) {
	prefixedName := PrefixPromptName(serverName, prompt.Name)

	entry := &PromptEntry{
		ServerName:   serverName,
		Prompt:       prompt,
		PrefixedName: prefixedName,
	}

	r.r.register(serverName, entry, func(_ string, e *PromptEntry) string {
		return e.PrefixedName
	})
}

// Get retrieves a prompt entry by its prefixed name.
func (r *PromptRegistry) Get(prefixedName string) (*PromptEntry, bool) {
	return r.r.get(prefixedName)
}

// GetByServer returns all prompt entries for a specific server.
func (r *PromptRegistry) GetByServer(serverName string) []*PromptEntry {
	return r.r.getByServer(serverName)
}

// All returns all registered prompt entries.
func (r *PromptRegistry) All() []*PromptEntry {
	return r.r.all()
}

// Count returns the total number of registered prompts.
func (r *PromptRegistry) Count() int {
	return r.r.count()
}

// ServerCount returns the number of servers with registered prompts.
func (r *PromptRegistry) ServerCount() int {
	return r.r.serverCount()
}

// Clear removes all entries from the registry.
func (r *PromptRegistry) Clear() {
	r.r.clear()
}

// RemoveServer removes all prompts for a specific server.
func (r *PromptRegistry) RemoveServer(serverName string) {
	r.r.removeServer(serverName, func(e *PromptEntry) string {
		return e.PrefixedName
	})
}

// PrefixPromptName creates a prefixed prompt name from server and prompt names.
// Example: ("github", "create-issue") -> "github_create_issue".
func PrefixPromptName(serverName, promptName string) string {
	sanitizedServer := sanitizeName(serverName)
	sanitizedPrompt := sanitizeName(promptName)

	return sanitizedServer + "_" + sanitizedPrompt
}

// ParsePrefixedPromptName splits a prefixed prompt name into server and prompt names.
// Returns an error if the format is invalid.
func ParsePrefixedPromptName(prefixedName string) (string, string, error) {
	if prefixedName == "" {
		return "", "", fmt.Errorf("%w: empty input", ErrInvalidPrefixedName)
	}

	server, prompt, found := strings.Cut(prefixedName, "_")

	if !found {
		return "", "", fmt.Errorf("%w: %q missing underscore separator", ErrInvalidPrefixedName, prefixedName)
	}

	if server == "" {
		return "", "", fmt.Errorf("%w: %q has empty server name", ErrInvalidPrefixedName, prefixedName)
	}

	return server, prompt, nil
}
