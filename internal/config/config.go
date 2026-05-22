// Package config provides configuration types and loading for Assern.
package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// MergeMode defines how environment variables are merged between global and project configs.
type MergeMode string

const (
	// MergeModeOverlay merges project env on top of global, keeping non-overridden values.
	MergeModeOverlay MergeMode = "overlay"
	// MergeModeReplace completely replaces global env with project env for the server.
	MergeModeReplace MergeMode = "replace"
)

// RetryConfig defines retry behavior for server operations.
type RetryConfig struct {
	MaxAttempts   int           `yaml:"max_attempts,omitempty" json:"maxAttempts,omitempty"`
	InitialDelay  time.Duration `yaml:"initial_delay,omitempty" json:"initialDelay,omitempty"`
	MaxDelay      time.Duration `yaml:"max_delay,omitempty" json:"maxDelay,omitempty"`
	BackoffFactor float64       `yaml:"backoff_factor,omitempty" json:"backoffFactor,omitempty"`
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

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

// OAuthConfig represents OAuth 2.0 configuration for authenticated transports.
// This matches the mcp-go transport.OAuthConfig structure.
type OAuthConfig struct {
	ClientID              string   `yaml:"client_id,omitempty" json:"clientId,omitempty"`
	ClientSecret          string   `yaml:"client_secret,omitempty" json:"clientSecret,omitempty"`
	RedirectURI           string   `yaml:"redirect_uri,omitempty" json:"redirectUri,omitempty"`
	Scopes                []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	AuthServerMetadataURL string   `yaml:"auth_server_metadata_url,omitempty" json:"authServerMetadataUrl,omitempty"`
	PKCEEnabled           bool     `yaml:"pkce_enabled,omitempty" json:"pkceEnabled,omitempty"`
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

// Config represents the complete Assern configuration (internal merged representation).
// Servers come from mcp.json, Projects and Settings come from config.yaml.
type Config struct {
	Servers  map[string]*ServerConfig  `yaml:"-" json:"-"` // Populated from mcp.json, not YAML
	Projects map[string]*ProjectConfig `yaml:"projects,omitempty"`
	Settings *Settings                 `yaml:"settings,omitempty"`
	// Auth holds named OAuth profiles that servers can reference by oauth_ref,
	// so several servers can share one set of OAuth credentials.
	Auth map[string]*OAuthConfig `yaml:"auth,omitempty"`
}

// ServerConfig defines an MCP server configuration.
type ServerConfig struct {
	// Stdio transport fields
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	WorkDir string            `yaml:"work_dir,omitempty"` // Working directory for stdio servers

	// HTTP/SSE transport fields
	URL     string            `yaml:"url,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"` // Custom HTTP headers (API keys, Bearer tokens)

	// OAuth configuration for authenticated HTTP/SSE transports
	OAuth *OAuthConfig `yaml:"oauth,omitempty"`

	// OAuthRef references a named profile under the top-level `auth:` map.
	// Used when OAuth is not set inline; inline OAuth takes precedence.
	OAuthRef string `yaml:"oauth_ref,omitempty"`

	// Transport type hint: "stdio", "sse", "http", "oauth-sse", "oauth-http" (auto-detected if not specified)
	Transport string `yaml:"transport,omitempty"`

	// Retry configuration for transient failures
	Retry *RetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`

	// Common fields
	Allowed   []string  `yaml:"allowed,omitempty"`
	Disabled  bool      `yaml:"disabled,omitempty"`
	MergeMode MergeMode `yaml:"merge_mode,omitempty"`
}

// ProjectConfig defines a project's configuration in the global registry.
type ProjectConfig struct {
	Directories []string                 `yaml:"directories,omitempty"`
	Env         map[string]string        `yaml:"env,omitempty"`
	Servers     map[string]*ServerConfig `yaml:"servers,omitempty"`
}

// LocalProjectConfig represents the .assern/config.yaml in a project directory.
type LocalProjectConfig struct {
	Project string                   `yaml:"project,omitempty"`
	Servers map[string]*ServerConfig `yaml:"servers,omitempty"`
	Env     map[string]string        `yaml:"env,omitempty"`
}

// Settings contains global Assern settings.
type Settings struct {
	LogLevel     string            `yaml:"log_level,omitempty"`
	LogFile      string            `yaml:"log_file,omitempty"`
	Timeout      time.Duration     `yaml:"timeout,omitempty"`
	OutputFormat string            `yaml:"output_format,omitempty"` // "json" or "toon"
	Aliases      map[string]string `yaml:"aliases,omitempty"`       // Tool aliases (alias -> prefixed_tool_name)
	Discovery    *DiscoveryConfig  `yaml:"discovery,omitempty"`     // Runtime tool discovery (progressive disclosure)
	CodeMode     *CodeModeConfig   `yaml:"code_mode,omitempty"`     // Sandboxed tool-composition via assern_execute
}

// CodeModeConfig controls the assern_execute meta-tool, which runs a sandboxed
// Starlark script that can orchestrate several aggregated tools in one call.
// Disabled by default; it adds a code-execution surface, so enable deliberately.
type CodeModeConfig struct {
	// Enabled exposes the assern_execute tool. Off by default.
	Enabled bool `yaml:"enabled,omitempty"`
	// Timeout bounds a single script's wall-clock execution time.
	Timeout time.Duration `yaml:"timeout,omitempty"`
	// MaxToolCalls caps how many tool calls one script may make.
	MaxToolCalls int `yaml:"max_tool_calls,omitempty"`
	// MaxOutputBytes caps the size of a script's captured output.
	MaxOutputBytes int `yaml:"max_output_bytes,omitempty"`
	// AllowedTools restricts which prefixed tool names a script may call.
	// Empty means any aggregated tool may be called.
	AllowedTools []string `yaml:"allowed_tools,omitempty"`
}

// IsEnabled reports whether code mode is configured and turned on.
func (c *CodeModeConfig) IsEnabled() bool {
	return c != nil && c.Enabled
}

// Clone creates a deep copy of the code-mode configuration.
func (c *CodeModeConfig) Clone() *CodeModeConfig {
	if c == nil {
		return nil
	}

	clone := &CodeModeConfig{
		Enabled:        c.Enabled,
		Timeout:        c.Timeout,
		MaxToolCalls:   c.MaxToolCalls,
		MaxOutputBytes: c.MaxOutputBytes,
	}

	if c.AllowedTools != nil {
		clone.AllowedTools = make([]string, len(c.AllowedTools))
		copy(clone.AllowedTools, c.AllowedTools)
	}

	return clone
}

// Default values for tool discovery. They only take effect when discovery is
// enabled; the feature is opt-in and off by default.
const (
	// DefaultDiscoveryMaxResults caps how many tools assern_search returns.
	DefaultDiscoveryMaxResults = 10
	// DefaultDiscoveryMaxLoaded caps how many tools a single session may have
	// loaded at once. Zero means unlimited.
	DefaultDiscoveryMaxLoaded = 30
)

// DiscoveryConfig controls runtime tool discovery (progressive disclosure).
// When disabled (the default), every aggregated tool is exposed to the client
// at startup, preserving the original behaviour. When enabled, only the
// assern_* meta-tools (plus any Pinned tools) are exposed up front, and clients
// pull in the tools they need at runtime via assern_search / assern_load.
type DiscoveryConfig struct {
	// Enabled turns progressive disclosure on. Off by default.
	Enabled bool `yaml:"enabled,omitempty"`
	// Pinned lists prefixed tool names (e.g. "github_search") that are always
	// exposed even in discovery mode, without needing a search.
	Pinned []string `yaml:"pinned,omitempty"`
	// MaxResults is the default number of matches assern_search returns.
	MaxResults int `yaml:"max_results,omitempty"`
	// MaxLoaded caps the number of tools a session may have loaded at once.
	// When the cap is reached, the least-recently loaded tool is evicted.
	// Zero uses DefaultDiscoveryMaxLoaded; a negative value means unlimited.
	MaxLoaded int `yaml:"max_loaded,omitempty"`
}

// IsEnabled reports whether discovery is configured and turned on.
func (d *DiscoveryConfig) IsEnabled() bool {
	return d != nil && d.Enabled
}

// EffectiveMaxResults returns the configured search limit or the default.
func (d *DiscoveryConfig) EffectiveMaxResults() int {
	if d == nil || d.MaxResults <= 0 {
		return DefaultDiscoveryMaxResults
	}

	return d.MaxResults
}

// EffectiveMaxLoaded returns the per-session load ceiling. A return of zero
// means unlimited (no eviction).
func (d *DiscoveryConfig) EffectiveMaxLoaded() int {
	if d == nil {
		return DefaultDiscoveryMaxLoaded
	}

	switch {
	case d.MaxLoaded < 0:
		return 0 // unlimited
	case d.MaxLoaded == 0:
		return DefaultDiscoveryMaxLoaded
	default:
		return d.MaxLoaded
	}
}

// Clone creates a deep copy of the discovery configuration.
func (d *DiscoveryConfig) Clone() *DiscoveryConfig {
	if d == nil {
		return nil
	}

	clone := &DiscoveryConfig{
		Enabled:    d.Enabled,
		MaxResults: d.MaxResults,
		MaxLoaded:  d.MaxLoaded,
	}

	if d.Pinned != nil {
		clone.Pinned = make([]string, len(d.Pinned))
		copy(clone.Pinned, d.Pinned)
	}

	return clone
}

// NewConfig creates a new empty Config with initialized maps.
func NewConfig() *Config {
	return &Config{
		Servers:  make(map[string]*ServerConfig),
		Projects: make(map[string]*ProjectConfig),
		Settings: DefaultSettings(),
	}
}

// DefaultSettings returns the default settings.
func DefaultSettings() *Settings {
	return &Settings{
		LogLevel:     "info",
		Timeout:      60 * time.Second,
		OutputFormat: "json", // Default to JSON for backward compatibility
	}
}

// Load reads a configuration file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return Parse(data)
}

// Parse parses YAML configuration data (config.yaml).
// Note: This only parses Projects and Settings. Servers come from mcp.json.
func Parse(data []byte) (*Config, error) {
	cfg := NewConfig()

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply defaults
	if cfg.Settings == nil {
		cfg.Settings = DefaultSettings()
	}

	// Set default merge mode for servers defined in project overrides
	for _, proj := range cfg.Projects {
		for _, srv := range proj.Servers {
			if srv.MergeMode == "" {
				srv.MergeMode = MergeModeOverlay
			}
		}
	}

	return cfg, nil
}

// LoadWithMCP loads both mcp.json and config.yaml from a directory and merges them.
func LoadWithMCP(mcpPath, configPath string) (*Config, error) {
	// Load MCP servers from mcp.json
	mcpCfg, err := LoadMCPConfig(mcpPath)
	if err != nil {
		return nil, fmt.Errorf("loading mcp config: %w", err)
	}

	// Load Assern config from config.yaml
	cfg, err := Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// config.yaml is optional, create empty config
			cfg = NewConfig()
		} else {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	}

	// Populate servers from MCP config
	cfg.Servers = mcpCfg.ToServerConfigs()

	return cfg, nil
}

// LoadGlobal loads the global configuration from ~/.valksor/assern/.
func LoadGlobal() (*Config, error) {
	mcpPath, err := GlobalMCPPath()
	if err != nil {
		return nil, err
	}

	configPath, err := GlobalConfigPath()
	if err != nil {
		return nil, err
	}

	return LoadWithMCP(mcpPath, configPath)
}

// LoadEffective loads all configuration sources and builds the effective config.
// It loads global mcp.json, global config.yaml, and optionally local .assern/ configs.
// The projectName is used to apply project-specific overrides from global config.
func LoadEffective(workDir, projectName string) (*Config, error) {
	// Load global MCP config
	globalMCPPath, err := GlobalMCPPath()
	if err != nil {
		return nil, fmt.Errorf("getting global mcp path: %w", err)
	}

	globalMCP, err := LoadMCPConfig(globalMCPPath)
	if err != nil {
		return nil, fmt.Errorf("loading global mcp config: %w", err)
	}

	// Load global Assern config
	globalConfigPath, err := GlobalConfigPath()
	if err != nil {
		return nil, fmt.Errorf("getting global config path: %w", err)
	}

	var globalConfig *Config
	if FileExists(globalConfigPath) {
		globalConfig, err = Load(globalConfigPath)
		if err != nil {
			return nil, fmt.Errorf("loading global config: %w", err)
		}
	}

	// Try to find local .assern directory
	var localMCP *MCPConfig
	var localConfig *LocalProjectConfig

	localDir := FindLocalConfigDir(workDir)
	if localDir != "" {
		// Load local MCP config if exists
		localMCPPath := LocalMCPPath(localDir)
		if FileExists(localMCPPath) {
			localMCP, err = LoadMCPConfig(localMCPPath)
			if err != nil {
				return nil, fmt.Errorf("loading local mcp config: %w", err)
			}
		}

		// Load local config if exists
		localConfigPath := LocalConfigPath(localDir)
		if FileExists(localConfigPath) {
			localConfig, err = LoadLocalProject(localConfigPath)
			if err != nil {
				return nil, fmt.Errorf("loading local config: %w", err)
			}

			// Use project name from local config if not specified
			if projectName == "" && localConfig.Project != "" {
				projectName = localConfig.Project
			}
		}
	}

	// Build effective config using all sources
	return BuildEffectiveConfig(globalMCP, globalConfig, localMCP, localConfig, projectName), nil
}

// LoadLocalProject reads a project-local .assern/config.yaml file.
func LoadLocalProject(path string) (*LocalProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading local project config: %w", err)
	}

	var cfg LocalProjectConfig

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing local project config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the given path.
func (c *Config) Save(path string) error {
	// Ensure directory exists. Owner-only: config may hold OAuth secrets.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// 0600: config can contain client secrets and credential headers.
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
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
