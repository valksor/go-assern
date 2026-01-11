package aggregator

import (
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolEntry represents a tool from a backend server.
type ToolEntry struct {
	// ServerName is the name of the backend server.
	ServerName string
	// Tool is the original tool definition.
	Tool mcp.Tool
	// PrefixedName is the tool name with server prefix (e.g., "github_search").
	PrefixedName string
}

// ToolRegistry manages the mapping of prefixed tool names to backend servers.
type ToolRegistry struct {
	// entries maps prefixed tool name to entry
	entries map[string]*ToolEntry
	// byServer maps server name to list of tool entries
	byServer map[string][]*ToolEntry
	mu       sync.RWMutex
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		entries:  make(map[string]*ToolEntry),
		byServer: make(map[string][]*ToolEntry),
	}
}

// Register adds tools from a server to the registry.
// If allowed is non-empty, only tools in the allowed list are registered.
func (r *ToolRegistry) Register(serverName string, tool mcp.Tool, allowed []string) {
	// Check if tool is in allowed list (if specified)
	if len(allowed) > 0 && !isAllowed(tool.Name, allowed) {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	prefixedName := PrefixToolName(serverName, tool.Name)

	entry := &ToolEntry{
		ServerName:   serverName,
		Tool:         tool,
		PrefixedName: prefixedName,
	}

	r.entries[prefixedName] = entry
	r.byServer[serverName] = append(r.byServer[serverName], entry)
}

// Get retrieves a tool entry by its prefixed name.
func (r *ToolRegistry) Get(prefixedName string) (*ToolEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[prefixedName]

	return entry, ok
}

// GetByServer returns all tool entries for a specific server.
func (r *ToolRegistry) GetByServer(serverName string) []*ToolEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := r.byServer[serverName]
	result := make([]*ToolEntry, len(entries))
	copy(result, entries)

	return result
}

// All returns all registered tool entries.
func (r *ToolRegistry) All() []*ToolEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ToolEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, entry)
	}

	return result
}

// Count returns the total number of registered tools.
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}

// ServerCount returns the number of servers with registered tools.
func (r *ToolRegistry) ServerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.byServer)
}

// Clear removes all entries from the registry.
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = make(map[string]*ToolEntry)
	r.byServer = make(map[string][]*ToolEntry)
}

// RemoveServer removes all tools for a specific server.
func (r *ToolRegistry) RemoveServer(serverName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := r.byServer[serverName]
	for _, entry := range entries {
		delete(r.entries, entry.PrefixedName)
	}

	delete(r.byServer, serverName)
}

// PrefixToolName creates a prefixed tool name from server and tool names.
// Example: ("github", "search-repos") -> "github_search_repos".
func PrefixToolName(serverName, toolName string) string {
	// Sanitize both names (replace dashes with underscores for compatibility)
	sanitizedServer := sanitizeName(serverName)
	sanitizedTool := sanitizeName(toolName)

	return sanitizedServer + "_" + sanitizedTool
}

// ParsePrefixedName splits a prefixed tool name into server and tool names.
// Returns empty strings if the format is invalid.
func ParsePrefixedName(prefixedName string) (string, string) {
	idx := strings.Index(prefixedName, "_")
	if idx == -1 {
		return "", ""
	}

	return prefixedName[:idx], prefixedName[idx+1:]
}

// sanitizeName replaces characters that may cause issues in tool names.
func sanitizeName(name string) string {
	// Replace dashes with underscores (Cursor compatibility)
	return strings.ReplaceAll(name, "-", "_")
}

// isAllowed checks if a tool name is in the allowed list.
func isAllowed(toolName string, allowed []string) bool {
	for _, a := range allowed {
		if a == toolName {
			return true
		}
	}

	return false
}

// ToolSummary provides a summary of a tool for display.
type ToolSummary struct {
	PrefixedName string
	ServerName   string
	OriginalName string
	Description  string
}

// Summarize returns a summary of a tool entry.
func (e *ToolEntry) Summarize() ToolSummary {
	return ToolSummary{
		PrefixedName: e.PrefixedName,
		ServerName:   e.ServerName,
		OriginalName: e.Tool.Name,
		Description:  e.Tool.Description,
	}
}
