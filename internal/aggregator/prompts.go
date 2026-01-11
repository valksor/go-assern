package aggregator

import (
	"strings"
	"sync"

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
	// entries maps prefixed name to entry
	entries map[string]*PromptEntry
	// byServer maps server name to list of prompt entries
	byServer map[string][]*PromptEntry
	mu       sync.RWMutex
}

// NewPromptRegistry creates a new prompt registry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		entries:  make(map[string]*PromptEntry),
		byServer: make(map[string][]*PromptEntry),
	}
}

// Register adds a prompt from a server to the registry.
func (r *PromptRegistry) Register(serverName string, prompt mcp.Prompt) {
	r.mu.Lock()
	defer r.mu.Unlock()

	prefixedName := PrefixPromptName(serverName, prompt.Name)

	entry := &PromptEntry{
		ServerName:   serverName,
		Prompt:       prompt,
		PrefixedName: prefixedName,
	}

	r.entries[prefixedName] = entry
	r.byServer[serverName] = append(r.byServer[serverName], entry)
}

// Get retrieves a prompt entry by its prefixed name.
func (r *PromptRegistry) Get(prefixedName string) (*PromptEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[prefixedName]

	return entry, ok
}

// GetByServer returns all prompt entries for a specific server.
func (r *PromptRegistry) GetByServer(serverName string) []*PromptEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := r.byServer[serverName]
	result := make([]*PromptEntry, len(entries))
	copy(result, entries)

	return result
}

// All returns all registered prompt entries.
func (r *PromptRegistry) All() []*PromptEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*PromptEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, entry)
	}

	return result
}

// Count returns the total number of registered prompts.
func (r *PromptRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}

// Clear removes all entries from the registry.
func (r *PromptRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = make(map[string]*PromptEntry)
	r.byServer = make(map[string][]*PromptEntry)
}

// RemoveServer removes all prompts for a specific server.
func (r *PromptRegistry) RemoveServer(serverName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := r.byServer[serverName]
	for _, entry := range entries {
		delete(r.entries, entry.PrefixedName)
	}

	delete(r.byServer, serverName)
}

// PrefixPromptName creates a prefixed prompt name from server and prompt names.
// Example: ("github", "create-issue") -> "github_create_issue".
func PrefixPromptName(serverName, promptName string) string {
	sanitizedServer := sanitizeName(serverName)
	sanitizedPrompt := sanitizeName(promptName)

	return sanitizedServer + "_" + sanitizedPrompt
}

// ParsePrefixedPromptName splits a prefixed prompt name into server and prompt names.
// Returns empty strings if the format is invalid.
func ParsePrefixedPromptName(prefixedName string) (string, string) {
	idx := strings.Index(prefixedName, "_")
	if idx == -1 {
		return "", ""
	}

	return prefixedName[:idx], prefixedName[idx+1:]
}
