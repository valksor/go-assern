package aggregator

import (
	"sync"
	"time"
)

// HealthStatus represents the health state of a server.
type HealthStatus string

const (
	// HealthHealthy indicates the server is responding normally.
	HealthHealthy HealthStatus = "healthy"
	// HealthUnhealthy indicates the server has failed multiple consecutive requests.
	HealthUnhealthy HealthStatus = "unhealthy"
	// HealthUnknown indicates the server has not been tested yet.
	HealthUnknown HealthStatus = "unknown"
)

// DefaultHealthThreshold is the number of consecutive failures before marking unhealthy.
const DefaultHealthThreshold = 3

// HealthTracker monitors server health based on call success/failure patterns.
type HealthTracker struct {
	threshold int
	servers   map[string]*serverHealth
	mu        sync.RWMutex
}

// serverHealth tracks health state for a single server.
type serverHealth struct {
	status              HealthStatus
	consecutiveFailures int
	lastFailure         time.Time
	lastSuccess         time.Time
	totalCalls          int64
	totalFailures       int64
}

// NewHealthTracker creates a new health tracker with the specified failure threshold.
// If threshold is <= 0, DefaultHealthThreshold is used.
func NewHealthTracker(threshold int) *HealthTracker {
	if threshold <= 0 {
		threshold = DefaultHealthThreshold
	}

	return &HealthTracker{
		threshold: threshold,
		servers:   make(map[string]*serverHealth),
	}
}

// RecordSuccess records a successful call to a server.
// This resets the consecutive failure count and marks the server as healthy.
func (h *HealthTracker) RecordSuccess(serverName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sh := h.getOrCreate(serverName)
	sh.consecutiveFailures = 0
	sh.lastSuccess = time.Now()
	sh.totalCalls++
	sh.status = HealthHealthy
}

// RecordFailure records a failed call to a server.
// If consecutive failures exceed the threshold, the server is marked unhealthy.
func (h *HealthTracker) RecordFailure(serverName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sh := h.getOrCreate(serverName)
	sh.consecutiveFailures++
	sh.lastFailure = time.Now()
	sh.totalCalls++
	sh.totalFailures++

	if sh.consecutiveFailures >= h.threshold {
		sh.status = HealthUnhealthy
	}
}

// Status returns the health status of a server.
func (h *HealthTracker) Status(serverName string) HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sh, ok := h.servers[serverName]
	if !ok {
		return HealthUnknown
	}

	return sh.status
}

// IsHealthy returns true if the server is healthy or unknown (not yet unhealthy).
func (h *HealthTracker) IsHealthy(serverName string) bool {
	status := h.Status(serverName)

	return status != HealthUnhealthy
}

// Stats returns health statistics for a server.
func (h *HealthTracker) Stats(serverName string) HealthStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sh, ok := h.servers[serverName]
	if !ok {
		return HealthStats{Status: HealthUnknown}
	}

	return HealthStats{
		Status:              sh.status,
		ConsecutiveFailures: sh.consecutiveFailures,
		LastFailure:         sh.lastFailure,
		LastSuccess:         sh.lastSuccess,
		TotalCalls:          sh.totalCalls,
		TotalFailures:       sh.totalFailures,
	}
}

// AllStats returns health statistics for all tracked servers.
func (h *HealthTracker) AllStats() map[string]HealthStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]HealthStats, len(h.servers))
	for name, sh := range h.servers {
		result[name] = HealthStats{
			Status:              sh.status,
			ConsecutiveFailures: sh.consecutiveFailures,
			LastFailure:         sh.lastFailure,
			LastSuccess:         sh.lastSuccess,
			TotalCalls:          sh.totalCalls,
			TotalFailures:       sh.totalFailures,
		}
	}

	return result
}

// MarkHealthy manually marks a server as healthy.
// This can be used after a successful health check probe.
func (h *HealthTracker) MarkHealthy(serverName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sh := h.getOrCreate(serverName)
	sh.status = HealthHealthy
	sh.consecutiveFailures = 0
}

// Reset removes all tracking data for a server.
func (h *HealthTracker) Reset(serverName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.servers, serverName)
}

// Clear removes all tracking data.
func (h *HealthTracker) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.servers = make(map[string]*serverHealth)
}

// getOrCreate returns the health state for a server, creating it if necessary.
// Must be called with the lock held.
func (h *HealthTracker) getOrCreate(serverName string) *serverHealth {
	sh, ok := h.servers[serverName]
	if !ok {
		sh = &serverHealth{status: HealthUnknown}
		h.servers[serverName] = sh
	}

	return sh
}

// HealthStats contains health statistics for a server.
type HealthStats struct {
	Status              HealthStatus
	ConsecutiveFailures int
	LastFailure         time.Time
	LastSuccess         time.Time
	TotalCalls          int64
	TotalFailures       int64
}

// FailureRate returns the failure rate as a percentage (0-100).
// Returns 0 if no calls have been made.
func (s HealthStats) FailureRate() float64 {
	if s.TotalCalls == 0 {
		return 0
	}

	return float64(s.TotalFailures) / float64(s.TotalCalls) * 100
}
