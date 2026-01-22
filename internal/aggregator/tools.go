package aggregator

import (
	"fmt"
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
	// aliases maps alias names to prefixed tool names
	aliases map[string]string
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		r:       newRegistry[*ToolEntry, string](),
		aliases: make(map[string]string),
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

// Get retrieves a tool entry by its prefixed name or alias.
// Aliases are resolved first, then the actual tool is looked up.
func (r *ToolRegistry) Get(name string) (*ToolEntry, bool) {
	// Check if name is an alias
	if target, isAlias := r.aliases[name]; isAlias {
		return r.r.get(target)
	}

	return r.r.get(name)
}

// SetAliases sets the tool aliases.
// Each alias maps to a prefixed tool name.
func (r *ToolRegistry) SetAliases(aliases map[string]string) {
	r.aliases = make(map[string]string, len(aliases))
	for alias, target := range aliases {
		r.aliases[alias] = target
	}
}

// AddAlias adds a single alias mapping.
func (r *ToolRegistry) AddAlias(alias, prefixedName string) {
	r.aliases[alias] = prefixedName
}

// RemoveAlias removes an alias.
func (r *ToolRegistry) RemoveAlias(alias string) {
	delete(r.aliases, alias)
}

// ResolveAlias resolves an alias to its prefixed tool name.
// Returns the original name if not an alias.
func (r *ToolRegistry) ResolveAlias(name string) string {
	if target, isAlias := r.aliases[name]; isAlias {
		return target
	}

	return name
}

// Aliases returns a copy of all defined aliases.
func (r *ToolRegistry) Aliases() map[string]string {
	result := make(map[string]string, len(r.aliases))
	for k, v := range r.aliases {
		result[k] = v
	}

	return result
}

// IsAlias checks if a name is a defined alias.
func (r *ToolRegistry) IsAlias(name string) bool {
	_, ok := r.aliases[name]

	return ok
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

// Clear removes all entries and aliases from the registry.
func (r *ToolRegistry) Clear() {
	r.r.clear()
	r.aliases = make(map[string]string)
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
// Returns an error if the format is invalid.
func ParsePrefixedName(prefixedName string) (string, string, error) {
	if prefixedName == "" {
		return "", "", fmt.Errorf("%w: empty input", ErrInvalidPrefixedName)
	}

	server, tool, found := strings.Cut(prefixedName, "_")

	if !found {
		return "", "", fmt.Errorf("%w: %q missing underscore separator", ErrInvalidPrefixedName, prefixedName)
	}

	if server == "" {
		return "", "", fmt.Errorf("%w: %q has empty server name", ErrInvalidPrefixedName, prefixedName)
	}

	return server, tool, nil
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
