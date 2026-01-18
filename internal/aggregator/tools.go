package aggregator

import (
	"strings"

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
	// Use the generic registry with entry pointer and string key
	r *registry[*ToolEntry, string]
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		r: newRegistry[*ToolEntry, string](),
	}
}

// Register adds tools from a server to the registry.
// If allowed is non-empty, only tools in the allowed list are registered.
func (r *ToolRegistry) Register(serverName string, tool mcp.Tool, allowed []string) {
	// Check if tool is in allowed list (if specified)
	if len(allowed) > 0 && !isAllowed(tool.Name, allowed) {
		return
	}

	prefixedName := PrefixToolName(serverName, tool.Name)

	entry := &ToolEntry{
		ServerName:   serverName,
		Tool:         tool,
		PrefixedName: prefixedName,
	}

	r.r.register(serverName, entry, func(_ string, e *ToolEntry) string {
		return e.PrefixedName
	})
}

// Get retrieves a tool entry by its prefixed name.
func (r *ToolRegistry) Get(prefixedName string) (*ToolEntry, bool) {
	return r.r.get(prefixedName)
}

// GetByServer returns all tool entries for a specific server.
func (r *ToolRegistry) GetByServer(serverName string) []*ToolEntry {
	return r.r.getByServer(serverName)
}

// All returns all registered tool entries.
func (r *ToolRegistry) All() []*ToolEntry {
	return r.r.all()
}

// Count returns the total number of registered tools.
func (r *ToolRegistry) Count() int {
	return r.r.count()
}

// ServerCount returns the number of servers with registered tools.
func (r *ToolRegistry) ServerCount() int {
	return r.r.serverCount()
}

// Clear removes all entries from the registry.
func (r *ToolRegistry) Clear() {
	r.r.clear()
}

// RemoveServer removes all tools for a specific server.
func (r *ToolRegistry) RemoveServer(serverName string) {
	r.r.removeServer(serverName, func(e *ToolEntry) string {
		return e.PrefixedName
	})
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
