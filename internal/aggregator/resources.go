package aggregator

import (
	"strings"
	"sync"

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
	// entries maps prefixed URI to entry
	entries map[string]*ResourceEntry
	// byServer maps server name to list of resource entries
	byServer map[string][]*ResourceEntry
	mu       sync.RWMutex
}

// NewResourceRegistry creates a new resource registry.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		entries:  make(map[string]*ResourceEntry),
		byServer: make(map[string][]*ResourceEntry),
	}
}

// Register adds a resource from a server to the registry.
func (r *ResourceRegistry) Register(serverName string, resource mcp.Resource) {
	r.mu.Lock()
	defer r.mu.Unlock()

	prefixedURI := PrefixResourceURI(serverName, resource.URI)

	entry := &ResourceEntry{
		ServerName:  serverName,
		Resource:    resource,
		PrefixedURI: prefixedURI,
		OriginalURI: resource.URI,
	}

	r.entries[prefixedURI] = entry
	r.byServer[serverName] = append(r.byServer[serverName], entry)
}

// Get retrieves a resource entry by its prefixed URI.
func (r *ResourceRegistry) Get(prefixedURI string) (*ResourceEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[prefixedURI]

	return entry, ok
}

// GetByServer returns all resource entries for a specific server.
func (r *ResourceRegistry) GetByServer(serverName string) []*ResourceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := r.byServer[serverName]
	result := make([]*ResourceEntry, len(entries))
	copy(result, entries)

	return result
}

// All returns all registered resource entries.
func (r *ResourceRegistry) All() []*ResourceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ResourceEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, entry)
	}

	return result
}

// Count returns the total number of registered resources.
func (r *ResourceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}

// Clear removes all entries from the registry.
func (r *ResourceRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = make(map[string]*ResourceEntry)
	r.byServer = make(map[string][]*ResourceEntry)
}

// RemoveServer removes all resources for a specific server.
func (r *ResourceRegistry) RemoveServer(serverName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := r.byServer[serverName]
	for _, entry := range entries {
		delete(r.entries, entry.PrefixedURI)
	}

	delete(r.byServer, serverName)
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
