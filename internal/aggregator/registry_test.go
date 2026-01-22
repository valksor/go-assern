package aggregator

import (
	"sync"
	"testing"
)

// TestRegistry tests the generic registry directly.

func TestRegistry_NewRegistry(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	if r.count() != 0 {
		t.Errorf("count() = %d, want 0", r.count())
	}

	if r.serverCount() != 0 {
		t.Errorf("serverCount() = %d, want 0", r.serverCount())
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	r.register("server1", "entry1", func(server string, entry string) string {
		return server + "_" + entry
	})

	if r.count() != 1 {
		t.Errorf("count() = %d, want 1", r.count())
	}

	entry, ok := r.get("server1_entry1")
	if !ok {
		t.Fatal("get() returned false for existing entry")
	}
	if entry != "entry1" {
		t.Errorf("entry = %q, want %q", entry, "entry1")
	}
}

func TestRegistry_GetByServer(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	r.register("server1", "entry1", keyFunc)
	r.register("server1", "entry2", keyFunc)
	r.register("server2", "entry3", keyFunc)

	entries := r.getByServer("server1")
	if len(entries) != 2 {
		t.Errorf("getByServer() returned %d entries, want 2", len(entries))
	}

	// Non-existent server
	entries = r.getByServer("nonexistent")
	if len(entries) != 0 {
		t.Errorf("getByServer(nonexistent) returned %d entries, want 0", len(entries))
	}
}

func TestRegistry_All(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	r.register("server1", "entry1", keyFunc)
	r.register("server2", "entry2", keyFunc)

	all := r.all()
	if len(all) != 2 {
		t.Errorf("all() returned %d entries, want 2", len(all))
	}
}

func TestRegistry_All_CacheInvalidation(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	// Initial population
	r.register("server1", "entry1", keyFunc)
	first := r.all()
	if len(first) != 1 {
		t.Fatalf("first all() = %d, want 1", len(first))
	}

	// Add more - cache should be invalidated
	r.register("server1", "entry2", keyFunc)
	second := r.all()
	if len(second) != 2 {
		t.Errorf("second all() = %d, want 2 (cache should be invalidated)", len(second))
	}

	// Clear - cache should be invalidated
	r.clear()
	third := r.all()
	if len(third) != 0 {
		t.Errorf("third all() after clear = %d, want 0", len(third))
	}
}

func TestRegistry_RemoveServer(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	r.register("server1", "entry1", keyFunc)
	r.register("server1", "entry2", keyFunc)
	r.register("server2", "entry3", keyFunc)

	r.removeServer("server1", func(entry string) string {
		return "server1_" + entry
	})

	if r.count() != 1 {
		t.Errorf("count after removeServer = %d, want 1", r.count())
	}

	if r.serverCount() != 1 {
		t.Errorf("serverCount after removeServer = %d, want 1", r.serverCount())
	}

	// Verify server1 entries are gone
	entries := r.getByServer("server1")
	if len(entries) != 0 {
		t.Errorf("getByServer(server1) = %d, want 0", len(entries))
	}
}

func TestRegistry_Clear(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	r.register("server1", "entry1", keyFunc)
	r.register("server2", "entry2", keyFunc)

	r.clear()

	if r.count() != 0 {
		t.Errorf("count after clear = %d, want 0", r.count())
	}

	if r.serverCount() != 0 {
		t.Errorf("serverCount after clear = %d, want 0", r.serverCount())
	}
}

// Helper function for tests.
func keyFunc(server, entry string) string {
	return server + "_" + entry
}

// Race condition tests

func TestRegistry_RaceCondition_ReadWrite(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := range 100 {
				server := "server" + string(rune('0'+idx%5))
				entry := "entry" + string(rune('0'+j%26))
				r.register(server, entry, keyFunc)
			}
		}(i)
	}

	// Concurrent reads
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = r.count()
				_ = r.all()
				_ = r.serverCount()
				_, _ = r.get("server0_entry0")
				_ = r.getByServer("server0")
			}
		}()
	}

	wg.Wait()
}

func TestRegistry_RaceCondition_AllCache(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	// Pre-populate
	for i := range 50 {
		r.register("server"+string(rune('0'+i%5)), "entry"+string(rune('0'+i)), keyFunc)
	}

	var wg sync.WaitGroup

	// Hammer all() which uses the cache
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 500 {
				all := r.all()
				_ = len(all)
			}
		}()
	}

	// Invalidate cache by adding entries
	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := range 50 {
				r.register("server"+string(rune('0'+idx)), "new_entry"+string(rune('0'+j)), keyFunc)
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistry_RaceCondition_ClearDuringRead(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	// Pre-populate
	for i := range 100 {
		r.register("server"+string(rune('0'+i%5)), "entry"+string(rune('0'+i)), keyFunc)
	}

	var wg sync.WaitGroup

	// Concurrent reads
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = r.all()
				_ = r.count()
			}
		}()
	}

	// Clear in the middle
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10 {
			r.clear()
			// Re-populate
			for i := range 20 {
				r.register("server", "entry"+string(rune('0'+i)), keyFunc)
			}
		}
	}()

	wg.Wait()
}

func TestRegistry_RaceCondition_RemoveServerDuringRead(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	var wg sync.WaitGroup

	// Continuous adds
	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server := "server" + string(rune('0'+idx))
			for j := range 100 {
				r.register(server, "entry"+string(rune('0'+j%26)), keyFunc)
			}
		}(i)
	}

	// Continuous reads
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = r.all()
				_ = r.getByServer("server0")
			}
		}()
	}

	// Remove servers in the middle
	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server := "server" + string(rune('0'+idx))
			for range 20 {
				r.removeServer(server, func(entry string) string {
					return server + "_" + entry
				})
			}
		}(i)
	}

	wg.Wait()
}

// Tests for the double-check locking pattern in all()

func TestRegistry_DoubleCheckLocking(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	// Add initial data
	r.register("server", "entry1", keyFunc)

	// Prime the cache
	first := r.all()
	if len(first) != 1 {
		t.Fatalf("first = %d, want 1", len(first))
	}

	var wg sync.WaitGroup

	// Many goroutines calling all() simultaneously
	// This tests the double-check pattern - only one should rebuild cache
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			all := r.all()
			if len(all) != 1 {
				t.Errorf("all() = %d, want 1", len(all))
			}
		}()
	}

	wg.Wait()
}

func TestRegistry_CacheConsistency(t *testing.T) {
	t.Parallel()

	r := newRegistry[string, string]()

	// Add entries
	for i := range 10 {
		r.register("server", "entry"+string(rune('0'+i)), keyFunc)
	}

	// Call all() multiple times - should return consistent results
	first := r.all()
	for i := range 100 {
		current := r.all()
		if len(current) != len(first) {
			t.Errorf("iteration %d: len = %d, want %d", i, len(current), len(first))
		}
	}

	// Add more and verify cache updates
	r.register("server", "newentry", keyFunc)

	updated := r.all()
	if len(updated) != len(first)+1 {
		t.Errorf("after add: len = %d, want %d", len(updated), len(first)+1)
	}
}
