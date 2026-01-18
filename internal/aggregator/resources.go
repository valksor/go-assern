package aggregator

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ResourceEntry represents a resource from a backend server.
type ResourceEntry struct {
	// ServerName is the name of the backend server.
	ServerName string
	// Resource is the original resource definition.
	Resource mcp.Resource
	// PrefixedURI is the URI with server prefix (e.g., "assern://github/file:///repo/file.txt").
	PrefixedURI string
	// OriginalURI is the original URI from the backend server.
	OriginalURI string
}

// ResourceRegistry manages the mapping of prefixed resource URIs to backend servers.
type ResourceRegistry struct {
	// Use the generic registry with entry pointer and string key
	r *registry[*ResourceEntry, string]
}

// NewResourceRegistry creates a new resource registry.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		r: newRegistry[*ResourceEntry, string](),
	}
}

// Register adds a resource from a server to the registry.
func (r *ResourceRegistry) Register(serverName string, resource mcp.Resource) {
	prefixedURI := PrefixResourceURI(serverName, resource.URI)

	entry := &ResourceEntry{
		ServerName:  serverName,
		Resource:    resource,
		PrefixedURI: prefixedURI,
		OriginalURI: resource.URI,
	}

	r.r.register(serverName, entry, func(_ string, e *ResourceEntry) string {
		return e.PrefixedURI
	})
}

// Get retrieves a resource entry by its prefixed URI.
func (r *ResourceRegistry) Get(prefixedURI string) (*ResourceEntry, bool) {
	return r.r.get(prefixedURI)
}

// GetByServer returns all resource entries for a specific server.
func (r *ResourceRegistry) GetByServer(serverName string) []*ResourceEntry {
	return r.r.getByServer(serverName)
}

// All returns all registered resource entries.
func (r *ResourceRegistry) All() []*ResourceEntry {
	return r.r.all()
}

// Count returns the total number of registered resources.
func (r *ResourceRegistry) Count() int {
	return r.r.count()
}

// ServerCount returns the number of servers with registered resources.
func (r *ResourceRegistry) ServerCount() int {
	return r.r.serverCount()
}

// Clear removes all entries from the registry.
func (r *ResourceRegistry) Clear() {
	r.r.clear()
}

// RemoveServer removes all resources for a specific server.
func (r *ResourceRegistry) RemoveServer(serverName string) {
	r.r.removeServer(serverName, func(e *ResourceEntry) string {
		return e.PrefixedURI
	})
}

// PrefixResourceURI creates a prefixed URI from server and resource URI.
// Format: "assern://{server}/{original-uri}"
// Example: ("github", "file:///repo/README.md") -> "assern://github/file:///repo/README.md"
func PrefixResourceURI(serverName, uri string) string {
	sanitizedServer := sanitizeName(serverName)

	return "assern://" + sanitizedServer + "/" + uri
}

// ParsePrefixedURI splits a prefixed URI into server name and original URI.
// Returns empty strings if the format is invalid.
func ParsePrefixedURI(prefixedURI string) (string, string) {
	const prefix = "assern://"

	if !strings.HasPrefix(prefixedURI, prefix) {
		return "", ""
	}

	rest := prefixedURI[len(prefix):]
	idx := strings.Index(rest, "/")

	if idx == -1 {
		return "", ""
	}

	return rest[:idx], rest[idx+1:]
}
