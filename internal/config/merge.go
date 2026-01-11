package config

// BuildEffectiveConfig creates the final merged configuration from all sources.
// Resolution order (highest priority first):
//  1. Local mcp.json (project-specific MCP servers)
//  2. Local config.yaml (project-specific overrides)
//  3. Global config.yaml project definition
//  4. Global mcp.json (base server definitions)
func BuildEffectiveConfig(
	globalMCP *MCPConfig,
	globalConfig *Config,
	localMCP *MCPConfig,
	localConfig *LocalProjectConfig,
	projectName string,
) *Config {
	// Start with empty config
	result := NewConfig()

	// 1. Copy settings from global config
	if globalConfig != nil && globalConfig.Settings != nil {
		result.Settings = &Settings{
			LogLevel:     globalConfig.Settings.LogLevel,
			LogFile:      globalConfig.Settings.LogFile,
			Timeout:      globalConfig.Settings.Timeout,
			OutputFormat: globalConfig.Settings.OutputFormat,
		}
	}

	// 2. Copy projects from global config
	if globalConfig != nil {
		for name, proj := range globalConfig.Projects {
			result.Projects[name] = proj.Clone()
		}
	}

	// 3. Load base servers from global mcp.json
	if globalMCP != nil {
		result.Servers = globalMCP.ToServerConfigs()
	}

	// 4. Apply project-level overrides from global config.yaml
	if projectName != "" && globalConfig != nil {
		if projectCfg, ok := globalConfig.Projects[projectName]; ok {
			// Apply project-level environment variables to all servers
			for name, srv := range result.Servers {
				srv.Env = mergeEnv(srv.Env, projectCfg.Env, srv.MergeMode)
				result.Servers[name] = srv
			}

			// Apply project-level server overrides
			for name, projSrv := range projectCfg.Servers {
				if existing, ok := result.Servers[name]; ok {
					result.Servers[name] = mergeServer(existing, projSrv)
				}
			}
		}
	}

	// 5. Add servers from local mcp.json
	if localMCP != nil {
		for name, srv := range localMCP.MCPServers {
			if existing, ok := result.Servers[name]; ok {
				// Merge with existing server (local MCP overrides)
				localSrv := &ServerConfig{
					Command:   srv.Command,
					Args:      srv.Args,
					Env:       srv.Env,
					WorkDir:   srv.WorkDir,
					URL:       srv.URL,
					Headers:   srv.Headers,
					OAuth:     srv.OAuth.Clone(),
					Transport: srv.Transport,
					MergeMode: MergeModeOverlay,
				}
				result.Servers[name] = mergeServer(existing, localSrv)
			} else {
				// New server from local mcp.json
				result.Servers[name] = &ServerConfig{
					Command:   srv.Command,
					Args:      srv.Args,
					Env:       srv.Env,
					WorkDir:   srv.WorkDir,
					URL:       srv.URL,
					Headers:   srv.Headers,
					OAuth:     srv.OAuth.Clone(),
					Transport: srv.Transport,
					MergeMode: MergeModeOverlay,
				}
			}
		}
	}

	// 6. Apply local config.yaml overrides (highest priority)
	if localConfig != nil {
		// Apply local environment variables
		if len(localConfig.Env) > 0 {
			for name, srv := range result.Servers {
				srv.Env = mergeEnv(srv.Env, localConfig.Env, srv.MergeMode)
				result.Servers[name] = srv
			}
		}

		// Apply local server overrides
		for name, localSrv := range localConfig.Servers {
			if existing, ok := result.Servers[name]; ok {
				result.Servers[name] = mergeServer(existing, localSrv)
			}
		}
	}

	return result
}

// Merge combines the base configuration with project-specific overrides.
// The result is a new configuration with the merged values.
//
// Deprecated: Use BuildEffectiveConfig for full config resolution.
func Merge(base *Config, projectName string, local *LocalProjectConfig) *Config {
	if base == nil {
		return nil
	}

	result := base.Clone()

	// Get project config from registry if it exists
	var projectCfg *ProjectConfig
	if projectName != "" {
		projectCfg = base.Projects[projectName]
	}

	// Apply project-level environment variables to all servers
	if projectCfg != nil {
		for name, srv := range result.Servers {
			srv.Env = mergeEnv(srv.Env, projectCfg.Env, srv.MergeMode)
			result.Servers[name] = srv
		}

		// Apply project-level server overrides
		for name, projSrv := range projectCfg.Servers {
			if existing, ok := result.Servers[name]; ok {
				result.Servers[name] = mergeServer(existing, projSrv)
			} else {
				// New server defined in project
				result.Servers[name] = projSrv.Clone()
			}
		}
	}

	// Apply local project overrides (highest priority)
	if local != nil {
		// Apply local environment variables
		if len(local.Env) > 0 {
			for name, srv := range result.Servers {
				srv.Env = mergeEnv(srv.Env, local.Env, srv.MergeMode)
				result.Servers[name] = srv
			}
		}

		// Apply local server overrides
		for name, localSrv := range local.Servers {
			if existing, ok := result.Servers[name]; ok {
				result.Servers[name] = mergeServer(existing, localSrv)
			} else {
				// New server defined locally
				result.Servers[name] = localSrv.Clone()
			}
		}
	}

	return result
}

// mergeServer merges an override server config onto a base server config.
func mergeServer(base, override *ServerConfig) *ServerConfig {
	if base == nil {
		return override.Clone()
	}

	if override == nil {
		return base.Clone()
	}

	result := base.Clone()

	// Override command if specified
	if override.Command != "" {
		result.Command = override.Command
	}

	// Override args if specified
	if len(override.Args) > 0 {
		result.Args = make([]string, len(override.Args))
		copy(result.Args, override.Args)
	}

	// Override WorkDir if specified
	if override.WorkDir != "" {
		result.WorkDir = override.WorkDir
	}

	// Override URL if specified
	if override.URL != "" {
		result.URL = override.URL
	}

	// Override transport if specified
	if override.Transport != "" {
		result.Transport = override.Transport
	}

	// Determine merge mode (override's mode takes precedence)
	mergeMode := result.MergeMode
	if override.MergeMode != "" {
		mergeMode = override.MergeMode
		result.MergeMode = mergeMode
	}

	// Merge environment variables based on mode
	result.Env = mergeEnv(result.Env, override.Env, mergeMode)

	// Merge headers based on mode (same as env - overlay or replace)
	result.Headers = mergeEnv(result.Headers, override.Headers, mergeMode)

	// Override OAuth config if specified (full replacement, not merge)
	if override.OAuth != nil {
		result.OAuth = override.OAuth.Clone()
	}

	// Override allowed list if specified
	if len(override.Allowed) > 0 {
		result.Allowed = make([]string, len(override.Allowed))
		copy(result.Allowed, override.Allowed)
	}

	// Override disabled flag if set
	if override.Disabled {
		result.Disabled = true
	}

	return result
}

// mergeEnv merges environment variables based on the merge mode.
func mergeEnv(base, override map[string]string, mode MergeMode) map[string]string {
	if len(override) == 0 {
		return cloneMap(base)
	}

	if len(base) == 0 || mode == MergeModeReplace {
		return cloneMap(override)
	}

	// Overlay mode: merge override on top of base
	result := cloneMap(base)
	for k, v := range override {
		result[k] = v
	}

	return result
}

// cloneMap creates a copy of a string map.
func cloneMap(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}

	return result
}

// GetEffectiveServers returns the list of enabled servers after applying
// project configuration and filtering disabled servers.
func GetEffectiveServers(cfg *Config) map[string]*ServerConfig {
	result := make(map[string]*ServerConfig)

	for name, srv := range cfg.Servers {
		// Server must have either command (stdio) or url (sse/http) and not be disabled
		hasTransport := srv.Command != "" || srv.URL != ""
		if !srv.Disabled && hasTransport {
			result[name] = srv
		}
	}

	return result
}

// RegisterProject adds or updates a project in the global configuration.
func (c *Config) RegisterProject(name string, directory string) {
	if c.Projects == nil {
		c.Projects = make(map[string]*ProjectConfig)
	}

	proj, exists := c.Projects[name]
	if !exists {
		proj = &ProjectConfig{
			Directories: []string{},
			Env:         make(map[string]string),
			Servers:     make(map[string]*ServerConfig),
		}
		c.Projects[name] = proj
	}

	// Add directory if not already present
	for _, d := range proj.Directories {
		if d == directory {
			return
		}
	}

	proj.Directories = append(proj.Directories, directory)
}
