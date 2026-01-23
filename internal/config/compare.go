package config

import "slices"

// ConfigDiff represents the difference between two configurations.
type ConfigDiff struct {
	Added     []string // Server names added
	Removed   []string // Server names removed
	Modified  []string // Server names with changed config
	Unchanged []string // Server names with same config
}

// DiffConfigs compares old and new configurations and returns the differences.
// It compares the effective servers (enabled, with valid transport) from each config.
func DiffConfigs(old, updated *Config) *ConfigDiff {
	diff := &ConfigDiff{}

	oldServers := GetEffectiveServers(old)
	newServers := GetEffectiveServers(updated)

	// Find added and modified servers
	for name, newSrv := range newServers {
		if oldSrv, exists := oldServers[name]; exists {
			if newSrv.Equal(oldSrv) {
				diff.Unchanged = append(diff.Unchanged, name)
			} else {
				diff.Modified = append(diff.Modified, name)
			}
		} else {
			diff.Added = append(diff.Added, name)
		}
	}

	// Find removed servers
	for name := range oldServers {
		if _, exists := newServers[name]; !exists {
			diff.Removed = append(diff.Removed, name)
		}
	}

	return diff
}

// HasChanges returns true if there are any configuration changes.
func (d *ConfigDiff) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Removed) > 0 || len(d.Modified) > 0
}

// Equal compares two ServerConfig for equality.
// Returns true if both configs are functionally equivalent.
func (s *ServerConfig) Equal(other *ServerConfig) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	// Compare basic string fields
	if s.Command != other.Command ||
		s.WorkDir != other.WorkDir ||
		s.URL != other.URL ||
		s.Transport != other.Transport ||
		s.Disabled != other.Disabled ||
		s.MergeMode != other.MergeMode {
		return false
	}

	// Compare slices
	if !slices.Equal(s.Args, other.Args) {
		return false
	}
	if !slices.Equal(s.Allowed, other.Allowed) {
		return false
	}

	// Compare maps
	if !mapsEqual(s.Env, other.Env) {
		return false
	}
	if !mapsEqual(s.Headers, other.Headers) {
		return false
	}

	// Compare OAuth configs
	if !s.OAuth.Equal(other.OAuth) {
		return false
	}

	// Compare Retry configs
	if !s.Retry.Equal(other.Retry) {
		return false
	}

	return true
}

// Equal compares two OAuthConfig for equality.
func (o *OAuthConfig) Equal(other *OAuthConfig) bool {
	if o == nil && other == nil {
		return true
	}
	if o == nil || other == nil {
		return false
	}

	return o.ClientID == other.ClientID &&
		o.ClientSecret == other.ClientSecret &&
		o.RedirectURI == other.RedirectURI &&
		o.AuthServerMetadataURL == other.AuthServerMetadataURL &&
		o.PKCEEnabled == other.PKCEEnabled &&
		slices.Equal(o.Scopes, other.Scopes)
}

// Equal compares two RetryConfig for equality.
func (r *RetryConfig) Equal(other *RetryConfig) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}

	return r.MaxAttempts == other.MaxAttempts &&
		r.InitialDelay == other.InitialDelay &&
		r.MaxDelay == other.MaxDelay &&
		r.BackoffFactor == other.BackoffFactor
}

// mapsEqual compares two string maps for equality.
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}

	return true
}
