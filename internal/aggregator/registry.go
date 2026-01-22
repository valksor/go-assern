package aggregator

import (
	"sync"
)

// registry is a generic registry for managing prefixed entries from multiple servers.
// Type parameter E is the entry type (e.g., *ToolEntry, *ResourceEntry, *PromptEntry).
// Type parameter K is the prefixed key type (string for tool/prompt names, URIs for resources).
type registry[E any, K comparable] struct {
	// entries maps prefixed key to entry
	entries map[K]E
	// byServer maps server name to list of entries
	byServer map[string][]E
	// cachedAll is a cached slice of all entries for faster all() calls
	cachedAll []E
	// cacheValid indicates whether cachedAll is up-to-date
	cacheValid bool
	mu         sync.RWMutex
}

// newRegistry creates a new generic registry.
func newRegistry[E any, K comparable]() *registry[E, K] {
	return &registry[E, K]{
		entries:  make(map[K]E),
		byServer: make(map[string][]E),
	}
}

// register adds an entry to the registry.
// The prefixFunc generates the prefixed key from the server name and entry.
func (r *registry[E, K]) register(serverName string, entry E, prefixFunc func(string, E) K) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := prefixFunc(serverName, entry)
	r.entries[key] = entry
	r.byServer[serverName] = append(r.byServer[serverName], entry)
	r.cacheValid = false // invalidate cache
}

// get retrieves an entry by its prefixed key.
func (r *registry[E, K]) get(key K) (E, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[key]

	return entry, ok
}

// getByServer returns all entries for a specific server.
func (r *registry[E, K]) getByServer(serverName string) []E {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := r.byServer[serverName]
	result := make([]E, len(entries))
	copy(result, entries)

	return result
}

// all returns all registered entries.
// The result is cached for repeated calls; cache is invalidated on mutations.
func (r *registry[E, K]) all() []E {
	r.mu.RLock()
	if r.cacheValid {
		// Return a copy of the cached slice to prevent external modification
		result := make([]E, len(r.cachedAll))
		copy(result, r.cachedAll)
		r.mu.RUnlock()

		return result
	}
	r.mu.RUnlock()

	// Cache miss - need to rebuild
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if r.cacheValid {
		result := make([]E, len(r.cachedAll))
		copy(result, r.cachedAll)

		return result
	}

	// Rebuild cache
	r.cachedAll = make([]E, 0, len(r.entries))
	for _, entry := range r.entries {
		r.cachedAll = append(r.cachedAll, entry)
	}
	r.cacheValid = true

	// Return a copy
	result := make([]E, len(r.cachedAll))
	copy(result, r.cachedAll)

	return result
}

// count returns the total number of registered entries.
func (r *registry[E, K]) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}

// serverCount returns the number of servers with registered entries.
func (r *registry[E, K]) serverCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.byServer)
}

// clear removes all entries from the registry.
func (r *registry[E, K]) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = make(map[K]E)
	r.byServer = make(map[string][]E)
	r.cachedAll = nil
	r.cacheValid = false
}

// removeServer removes all entries for a specific server.
// The keyFunc extracts the prefixed key from an entry.
func (r *registry[E, K]) removeServer(serverName string, keyFunc func(E) K) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := r.byServer[serverName]
	for _, entry := range entries {
		key := keyFunc(entry)
		delete(r.entries, key)
	}

	delete(r.byServer, serverName)
	r.cacheValid = false // invalidate cache
}
