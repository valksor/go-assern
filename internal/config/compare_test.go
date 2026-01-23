package config

import (
	"testing"
	"time"
)

func TestServerConfigEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        *ServerConfig
		b        *ServerConfig
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "a nil",
			a:        nil,
			b:        &ServerConfig{Command: "test"},
			expected: false,
		},
		{
			name:     "b nil",
			a:        &ServerConfig{Command: "test"},
			b:        nil,
			expected: false,
		},
		{
			name: "identical configs",
			a: &ServerConfig{
				Command: "node",
				Args:    []string{"server.js"},
				Env:     map[string]string{"PORT": "3000"},
				URL:     "",
			},
			b: &ServerConfig{
				Command: "node",
				Args:    []string{"server.js"},
				Env:     map[string]string{"PORT": "3000"},
				URL:     "",
			},
			expected: true,
		},
		{
			name: "different command",
			a: &ServerConfig{
				Command: "node",
			},
			b: &ServerConfig{
				Command: "python",
			},
			expected: false,
		},
		{
			name: "different args",
			a: &ServerConfig{
				Command: "node",
				Args:    []string{"server.js"},
			},
			b: &ServerConfig{
				Command: "node",
				Args:    []string{"server.js", "--debug"},
			},
			expected: false,
		},
		{
			name: "different env",
			a: &ServerConfig{
				Command: "node",
				Env:     map[string]string{"PORT": "3000"},
			},
			b: &ServerConfig{
				Command: "node",
				Env:     map[string]string{"PORT": "4000"},
			},
			expected: false,
		},
		{
			name: "different allowed list",
			a: &ServerConfig{
				Command: "node",
				Allowed: []string{"tool1"},
			},
			b: &ServerConfig{
				Command: "node",
				Allowed: []string{"tool1", "tool2"},
			},
			expected: false,
		},
		{
			name: "url vs command",
			a: &ServerConfig{
				Command: "node",
			},
			b: &ServerConfig{
				URL: "http://localhost:3000",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Equal(tt.b)
			if result != tt.expected {
				t.Errorf("Equal() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestOAuthConfigEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        *OAuthConfig
		b        *OAuthConfig
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name: "identical",
			a: &OAuthConfig{
				ClientID:     "client1",
				ClientSecret: "secret",
				Scopes:       []string{"read", "write"},
			},
			b: &OAuthConfig{
				ClientID:     "client1",
				ClientSecret: "secret",
				Scopes:       []string{"read", "write"},
			},
			expected: true,
		},
		{
			name: "different scopes",
			a: &OAuthConfig{
				ClientID: "client1",
				Scopes:   []string{"read"},
			},
			b: &OAuthConfig{
				ClientID: "client1",
				Scopes:   []string{"read", "write"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Equal(tt.b)
			if result != tt.expected {
				t.Errorf("Equal() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRetryConfigEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        *RetryConfig
		b        *RetryConfig
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name: "identical",
			a: &RetryConfig{
				MaxAttempts:   3,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      5 * time.Second,
				BackoffFactor: 2.0,
			},
			b: &RetryConfig{
				MaxAttempts:   3,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      5 * time.Second,
				BackoffFactor: 2.0,
			},
			expected: true,
		},
		{
			name: "different max attempts",
			a: &RetryConfig{
				MaxAttempts: 3,
			},
			b: &RetryConfig{
				MaxAttempts: 5,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Equal(tt.b)
			if result != tt.expected {
				t.Errorf("Equal() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDiffConfigs(t *testing.T) {
	tests := []struct {
		name         string
		old          *Config
		new          *Config
		wantAdded    []string
		wantRemoved  []string
		wantModified []string
	}{
		{
			name: "no changes",
			old: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node", Args: []string{"s1.js"}},
				},
			},
			new: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node", Args: []string{"s1.js"}},
				},
			},
			wantAdded:    nil,
			wantRemoved:  nil,
			wantModified: nil,
		},
		{
			name: "server added",
			old: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node"},
				},
			},
			new: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node"},
					"server2": {Command: "python"},
				},
			},
			wantAdded:    []string{"server2"},
			wantRemoved:  nil,
			wantModified: nil,
		},
		{
			name: "server removed",
			old: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node"},
					"server2": {Command: "python"},
				},
			},
			new: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node"},
				},
			},
			wantAdded:    nil,
			wantRemoved:  []string{"server2"},
			wantModified: nil,
		},
		{
			name: "server modified",
			old: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node", Args: []string{"old.js"}},
				},
			},
			new: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node", Args: []string{"new.js"}},
				},
			},
			wantAdded:    nil,
			wantRemoved:  nil,
			wantModified: []string{"server1"},
		},
		{
			name: "mixed changes",
			old: &Config{
				Servers: map[string]*ServerConfig{
					"keep":   {Command: "node"},
					"modify": {Command: "python", Args: []string{"old.py"}},
					"remove": {Command: "go"},
				},
			},
			new: &Config{
				Servers: map[string]*ServerConfig{
					"keep":   {Command: "node"},
					"modify": {Command: "python", Args: []string{"new.py"}},
					"add":    {Command: "rust"},
				},
			},
			wantAdded:    []string{"add"},
			wantRemoved:  []string{"remove"},
			wantModified: []string{"modify"},
		},
		{
			name: "disabled server not in effective",
			old: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node"},
				},
			},
			new: &Config{
				Servers: map[string]*ServerConfig{
					"server1": {Command: "node", Disabled: true},
				},
			},
			wantAdded:    nil,
			wantRemoved:  []string{"server1"},
			wantModified: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := DiffConfigs(tt.old, tt.new)

			if !slicesContainSame(diff.Added, tt.wantAdded) {
				t.Errorf("Added = %v, want %v", diff.Added, tt.wantAdded)
			}
			if !slicesContainSame(diff.Removed, tt.wantRemoved) {
				t.Errorf("Removed = %v, want %v", diff.Removed, tt.wantRemoved)
			}
			if !slicesContainSame(diff.Modified, tt.wantModified) {
				t.Errorf("Modified = %v, want %v", diff.Modified, tt.wantModified)
			}
		})
	}
}

func TestConfigDiffHasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     *ConfigDiff
		expected bool
	}{
		{
			name:     "no changes",
			diff:     &ConfigDiff{Unchanged: []string{"server1"}},
			expected: false,
		},
		{
			name:     "has added",
			diff:     &ConfigDiff{Added: []string{"server1"}},
			expected: true,
		},
		{
			name:     "has removed",
			diff:     &ConfigDiff{Removed: []string{"server1"}},
			expected: true,
		},
		{
			name:     "has modified",
			diff:     &ConfigDiff{Modified: []string{"server1"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.diff.HasChanges()
			if result != tt.expected {
				t.Errorf("HasChanges() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// slicesContainSame checks if two slices contain the same elements (order-independent).
func slicesContainSame(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}

	aMap := make(map[string]int)
	for _, v := range a {
		aMap[v]++
	}
	for _, v := range b {
		aMap[v]--
		if aMap[v] < 0 {
			return false
		}
	}

	return true
}
