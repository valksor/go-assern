// Package config provides configuration types and loading for Assern.
package config

import (
	"fmt"
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

// Config represents the complete Assern configuration (internal merged representation).
// Servers come from mcp.json, Projects and Settings come from config.yaml.
type Config struct {
	Servers  map[string]*ServerConfig  `yaml:"-" json:"-"` // Populated from mcp.json, not YAML
	Projects map[string]*ProjectConfig `yaml:"projects,omitempty"`
	Settings *Settings                 `yaml:"settings,omitempty"`
}

// ServerConfig defines an MCP server configuration.
type ServerConfig struct {
	Command   string            `yaml:"command"`
	Args      []string          `yaml:"args,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Allowed   []string          `yaml:"allowed,omitempty"`
	Disabled  bool              `yaml:"disabled,omitempty"`
	MergeMode MergeMode         `yaml:"merge_mode,omitempty"`
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
	LogLevel string        `yaml:"log_level,omitempty"`
	LogFile  string        `yaml:"log_file,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`
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
		LogLevel: "info",
		Timeout:  60 * time.Second,
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
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
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

	// Clone settings
	if c.Settings != nil {
		clone.Settings = &Settings{
			LogLevel: c.Settings.LogLevel,
			LogFile:  c.Settings.LogFile,
			Timeout:  c.Settings.Timeout,
		}
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
		Allowed:   make([]string, len(s.Allowed)),
		Disabled:  s.Disabled,
		MergeMode: s.MergeMode,
	}

	copy(clone.Args, s.Args)
	copy(clone.Allowed, s.Allowed)

	for k, v := range s.Env {
		clone.Env[k] = v
	}

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

	for k, v := range p.Env {
		clone.Env[k] = v
	}

	for name, srv := range p.Servers {
		clone.Servers[name] = srv.Clone()
	}

	return clone
}
