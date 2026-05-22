package config

import (
	"maps"
	"slices"
)

// Clone creates a deep copy of the retry configuration.
func (r *RetryConfig) Clone() *RetryConfig {
	if r == nil {
		return nil
	}

	return &RetryConfig{
		MaxAttempts:   r.MaxAttempts,
		InitialDelay:  r.InitialDelay,
		MaxDelay:      r.MaxDelay,
		BackoffFactor: r.BackoffFactor,
	}
}

// Clone creates a deep copy of the OAuth configuration.
func (o *OAuthConfig) Clone() *OAuthConfig {
	if o == nil {
		return nil
	}

	clone := &OAuthConfig{
		ClientID:              o.ClientID,
		ClientSecret:          o.ClientSecret,
		RedirectURI:           o.RedirectURI,
		Scopes:                make([]string, len(o.Scopes)),
		AuthServerMetadataURL: o.AuthServerMetadataURL,
		PKCEEnabled:           o.PKCEEnabled,
	}

	copy(clone.Scopes, o.Scopes)

	return clone
}

// Clone creates a deep copy of the code-mode configuration.
func (c *CodeModeConfig) Clone() *CodeModeConfig {
	if c == nil {
		return nil
	}

	return &CodeModeConfig{
		Enabled:        c.Enabled,
		Timeout:        c.Timeout,
		MaxToolCalls:   c.MaxToolCalls,
		MaxOutputBytes: c.MaxOutputBytes,
		AllowedTools:   slices.Clone(c.AllowedTools),
	}
}

// Clone creates a deep copy of the discovery configuration.
func (d *DiscoveryConfig) Clone() *DiscoveryConfig {
	if d == nil {
		return nil
	}

	return &DiscoveryConfig{
		Enabled:    d.Enabled,
		MaxResults: d.MaxResults,
		MaxLoaded:  d.MaxLoaded,
		Pinned:     slices.Clone(d.Pinned),
	}
}

// Clone creates a deep copy of the configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	clone := NewConfig()

	// Clone servers
	for name, srv := range c.Servers {
		clone.Servers[name] = srv.Clone()
	}

	// Clone projects
	for name, proj := range c.Projects {
		clone.Projects[name] = proj.Clone()
	}

	// Clone auth profiles
	if len(c.Auth) > 0 {
		clone.Auth = make(map[string]*OAuthConfig, len(c.Auth))
		for name, profile := range c.Auth {
			clone.Auth[name] = profile.Clone()
		}
	}

	// Clone settings
	if c.Settings != nil {
		clone.Settings = &Settings{
			LogLevel:     c.Settings.LogLevel,
			LogFile:      c.Settings.LogFile,
			Timeout:      c.Settings.Timeout,
			OutputFormat: c.Settings.OutputFormat,
			Aliases:      make(map[string]string, len(c.Settings.Aliases)),
			Discovery:    c.Settings.Discovery.Clone(),
			CodeMode:     c.Settings.CodeMode.Clone(),
		}
		maps.Copy(clone.Settings.Aliases, c.Settings.Aliases)
	}

	return clone
}

// Clone creates a deep copy of the server configuration.
func (s *ServerConfig) Clone() *ServerConfig {
	if s == nil {
		return nil
	}

	clone := &ServerConfig{
		Command:   s.Command,
		Args:      make([]string, len(s.Args)),
		Env:       make(map[string]string, len(s.Env)),
		WorkDir:   s.WorkDir,
		URL:       s.URL,
		Headers:   make(map[string]string, len(s.Headers)),
		OAuth:     s.OAuth.Clone(),
		OAuthRef:  s.OAuthRef,
		Transport: s.Transport,
		Retry:     s.Retry.Clone(),
		Allowed:   make([]string, len(s.Allowed)),
		Disabled:  s.Disabled,
		MergeMode: s.MergeMode,
	}

	copy(clone.Args, s.Args)
	copy(clone.Allowed, s.Allowed)

	maps.Copy(clone.Env, s.Env)

	maps.Copy(clone.Headers, s.Headers)

	return clone
}

// Clone creates a deep copy of the project configuration.
func (p *ProjectConfig) Clone() *ProjectConfig {
	if p == nil {
		return nil
	}

	clone := &ProjectConfig{
		Directories: make([]string, len(p.Directories)),
		Env:         make(map[string]string, len(p.Env)),
		Servers:     make(map[string]*ServerConfig, len(p.Servers)),
	}

	copy(clone.Directories, p.Directories)

	maps.Copy(clone.Env, p.Env)

	for name, srv := range p.Servers {
		clone.Servers[name] = srv.Clone()
	}

	return clone
}
