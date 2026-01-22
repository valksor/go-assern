package aggregator

import (
	"sync"
	"testing"
)

func TestHealthTracker_NewTracker(t *testing.T) {
	t.Parallel()

	t.Run("default threshold", func(t *testing.T) {
		t.Parallel()

		ht := NewHealthTracker(0)
		if ht.threshold != DefaultHealthThreshold {
			t.Errorf("threshold = %d, want %d", ht.threshold, DefaultHealthThreshold)
		}
	})

	t.Run("custom threshold", func(t *testing.T) {
		t.Parallel()

		ht := NewHealthTracker(5)
		if ht.threshold != 5 {
			t.Errorf("threshold = %d, want 5", ht.threshold)
		}
	})

	t.Run("negative threshold uses default", func(t *testing.T) {
		t.Parallel()

		ht := NewHealthTracker(-1)
		if ht.threshold != DefaultHealthThreshold {
			t.Errorf("threshold = %d, want %d", ht.threshold, DefaultHealthThreshold)
		}
	})
}

func TestHealthTracker_UnknownServer(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	status := ht.Status("unknown-server")
	if status != HealthUnknown {
		t.Errorf("Status() = %q, want %q", status, HealthUnknown)
	}

	if !ht.IsHealthy("unknown-server") {
		t.Error("unknown server should be considered healthy")
	}
}

func TestHealthTracker_RecordSuccess(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	ht.RecordSuccess("server1")

	if ht.Status("server1") != HealthHealthy {
		t.Errorf("Status() = %q, want %q", ht.Status("server1"), HealthHealthy)
	}

	stats := ht.Stats("server1")
	if stats.TotalCalls != 1 {
		t.Errorf("TotalCalls = %d, want 1", stats.TotalCalls)
	}
	if stats.TotalFailures != 0 {
		t.Errorf("TotalFailures = %d, want 0", stats.TotalFailures)
	}
	if stats.LastSuccess.IsZero() {
		t.Error("LastSuccess should be set")
	}
}

func TestHealthTracker_RecordFailure(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	// First failure - still unknown/healthy
	ht.RecordFailure("server1")
	if ht.Status("server1") != HealthUnknown {
		t.Errorf("Status() = %q, want %q after 1 failure", ht.Status("server1"), HealthUnknown)
	}

	// Second failure - still unknown/healthy
	ht.RecordFailure("server1")
	if ht.Status("server1") != HealthUnknown {
		t.Errorf("Status() = %q, want %q after 2 failures", ht.Status("server1"), HealthUnknown)
	}

	// Third failure - should be unhealthy
	ht.RecordFailure("server1")
	if ht.Status("server1") != HealthUnhealthy {
		t.Errorf("Status() = %q, want %q after 3 failures", ht.Status("server1"), HealthUnhealthy)
	}

	stats := ht.Stats("server1")
	if stats.ConsecutiveFailures != 3 {
		t.Errorf("ConsecutiveFailures = %d, want 3", stats.ConsecutiveFailures)
	}
	if stats.TotalCalls != 3 {
		t.Errorf("TotalCalls = %d, want 3", stats.TotalCalls)
	}
	if stats.TotalFailures != 3 {
		t.Errorf("TotalFailures = %d, want 3", stats.TotalFailures)
	}
}

func TestHealthTracker_SuccessResetsFailures(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	// Two failures
	ht.RecordFailure("server1")
	ht.RecordFailure("server1")

	// One success resets counter
	ht.RecordSuccess("server1")

	stats := ht.Stats("server1")
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", stats.ConsecutiveFailures)
	}
	if stats.Status != HealthHealthy {
		t.Errorf("Status = %q, want %q", stats.Status, HealthHealthy)
	}

	// Two more failures won't trigger unhealthy
	ht.RecordFailure("server1")
	ht.RecordFailure("server1")
	if ht.Status("server1") != HealthHealthy {
		t.Error("should still be healthy after 2 failures (threshold is 3)")
	}
}

func TestHealthTracker_RecoverFromUnhealthy(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(2)

	// Become unhealthy
	ht.RecordFailure("server1")
	ht.RecordFailure("server1")
	if ht.Status("server1") != HealthUnhealthy {
		t.Error("should be unhealthy")
	}

	// Success recovers
	ht.RecordSuccess("server1")
	if ht.Status("server1") != HealthHealthy {
		t.Errorf("Status() = %q, want %q after recovery", ht.Status("server1"), HealthHealthy)
	}
}

func TestHealthTracker_MarkHealthy(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(2)

	// Become unhealthy
	ht.RecordFailure("server1")
	ht.RecordFailure("server1")

	// Manual mark healthy
	ht.MarkHealthy("server1")
	if ht.Status("server1") != HealthHealthy {
		t.Errorf("Status() = %q, want %q", ht.Status("server1"), HealthHealthy)
	}

	stats := ht.Stats("server1")
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0 after MarkHealthy", stats.ConsecutiveFailures)
	}
}

func TestHealthTracker_Reset(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	ht.RecordSuccess("server1")
	ht.RecordFailure("server2")

	ht.Reset("server1")

	if ht.Status("server1") != HealthUnknown {
		t.Errorf("Status() = %q, want %q after Reset", ht.Status("server1"), HealthUnknown)
	}
	// server2 should still exist
	stats := ht.Stats("server2")
	if stats.TotalCalls != 1 {
		t.Error("server2 should not be affected by reset of server1")
	}
}

func TestHealthTracker_Clear(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	ht.RecordSuccess("server1")
	ht.RecordFailure("server2")

	ht.Clear()

	if ht.Status("server1") != HealthUnknown {
		t.Errorf("Status(server1) = %q, want %q after Clear", ht.Status("server1"), HealthUnknown)
	}
	if ht.Status("server2") != HealthUnknown {
		t.Errorf("Status(server2) = %q, want %q after Clear", ht.Status("server2"), HealthUnknown)
	}
}

func TestHealthTracker_AllStats(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(3)

	ht.RecordSuccess("server1")
	ht.RecordFailure("server2")
	ht.RecordFailure("server2")

	all := ht.AllStats()

	if len(all) != 2 {
		t.Errorf("AllStats() returned %d entries, want 2", len(all))
	}

	if all["server1"].Status != HealthHealthy {
		t.Errorf("server1 status = %q, want %q", all["server1"].Status, HealthHealthy)
	}
	if all["server2"].ConsecutiveFailures != 2 {
		t.Errorf("server2 consecutive failures = %d, want 2", all["server2"].ConsecutiveFailures)
	}
}

func TestHealthStats_FailureRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		calls    int64
		failures int64
		expected float64
	}{
		{"no calls", 0, 0, 0},
		{"all success", 100, 0, 0},
		{"all failures", 100, 100, 100},
		{"50% failure", 100, 50, 50},
		{"10% failure", 100, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := HealthStats{
				TotalCalls:    tt.calls,
				TotalFailures: tt.failures,
			}

			rate := stats.FailureRate()
			if rate != tt.expected {
				t.Errorf("FailureRate() = %v, want %v", rate, tt.expected)
			}
		})
	}
}

func TestHealthTracker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(10)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server := "server" + string(rune('0'+idx%3))
			for range 100 {
				if idx%2 == 0 {
					ht.RecordSuccess(server)
				} else {
					ht.RecordFailure(server)
				}
			}
		}(i)
	}

	// Concurrent reads
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = ht.Status("server0")
				_ = ht.IsHealthy("server1")
				_ = ht.Stats("server2")
				_ = ht.AllStats()
			}
		}()
	}

	wg.Wait()
	// No race conditions should occur
}

func TestHealthTracker_IsHealthy(t *testing.T) {
	t.Parallel()

	ht := NewHealthTracker(2)

	// Unknown is healthy
	if !ht.IsHealthy("new-server") {
		t.Error("unknown server should be considered healthy")
	}

	// Healthy is healthy
	ht.RecordSuccess("server1")
	if !ht.IsHealthy("server1") {
		t.Error("healthy server should return IsHealthy=true")
	}

	// Unhealthy is not healthy
	ht.RecordFailure("server2")
	ht.RecordFailure("server2")
	if ht.IsHealthy("server2") {
		t.Error("unhealthy server should return IsHealthy=false")
	}
}
