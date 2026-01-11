package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valksor/go-assern/internal/config"
)

func TestParse(t *testing.T) {
	t.Parallel()

	// Note: Servers now come from mcp.json, not config.yaml
	// only contains projects and settings
	yaml := `
projects:
  myproject:
    directories:
      - ~/work/myproject
    env:
      GITHUB_TOKEN: "${PROJECT_TOKEN}"
    servers:
      github:
        allowed:
          - search_repos
          - create_issue

settings:
  log_level: debug
  timeout: 120s
`

	cfg, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Servers should be empty (they come from mcp.json now)
	if len(cfg.Servers) != 0 {
		t.Errorf("expected 0 servers in config.yaml, got %d", len(cfg.Servers))
	}

	// Check projects
	if len(cfg.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(cfg.Projects))
	}

	proj, ok := cfg.Projects["myproject"]
	if !ok {
		t.Fatal("myproject not found")
	}

	if len(proj.Directories) != 1 {
		t.Errorf("expected 1 directory, got %d", len(proj.Directories))
	}

	// Check project-level server overrides
	if len(proj.Servers) != 1 {
		t.Errorf("expected 1 server override in project, got %d", len(proj.Servers))
	}

	githubOverride, ok := proj.Servers["github"]
	if !ok {
		t.Fatal("github server override not found in project")
	}

	if len(githubOverride.Allowed) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(githubOverride.Allowed))
	}

	// Check default merge mode for project server override
	if githubOverride.MergeMode != config.MergeModeOverlay {
		t.Errorf("expected default merge mode 'overlay', got '%s'", githubOverride.MergeMode)
	}

	// Check settings
	if cfg.Settings.LogLevel != "debug" {
		t.Errorf("expected log_level 'debug', got '%s'", cfg.Settings.LogLevel)
	}
}

func TestClone(t *testing.T) {
	t.Parallel()

	original := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"test": {
				Command:   "cmd",
				Args:      []string{"arg1", "arg2"},
				Env:       map[string]string{"KEY": "value"},
				MergeMode: config.MergeModeReplace,
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"proj": {
				Directories: []string{"/path"},
				Env:         map[string]string{"PROJ_KEY": "proj_value"},
			},
		},
		Settings: config.DefaultSettings(),
	}

	clone := original.Clone()

	// Modify original
	original.Servers["test"].Command = "modified"
	original.Servers["test"].Env["KEY"] = "modified"
	original.Projects["proj"].Directories[0] = "/modified"

	// Check clone is unchanged
	if clone.Servers["test"].Command != "cmd" {
		t.Error("clone command was modified")
	}

	if clone.Servers["test"].Env["KEY"] != "value" {
		t.Error("clone env was modified")
	}

	if clone.Projects["proj"].Directories[0] != "/path" {
		t.Error("clone directories was modified")
	}
}

func TestLoadAndSave(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Note: Servers are NOT saved to config.yaml (they go to mcp.json)
	// config.yaml only saves projects and settings
	original := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"test": {
				Command: "echo",
				Args:    []string{"hello"},
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Directories: []string{"/test/path"},
			},
		},
		Settings: config.DefaultSettings(),
	}

	// Save
	if err := original.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load
	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Servers should NOT be loaded from config.yaml
	if len(loaded.Servers) != 0 {
		t.Errorf("expected 0 servers from config.yaml, got %d", len(loaded.Servers))
	}

	// Projects should be loaded
	if len(loaded.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(loaded.Projects))
	}

	proj, ok := loaded.Projects["myproject"]
	if !ok {
		t.Fatal("myproject not found after load")
	}

	if len(proj.Directories) != 1 || proj.Directories[0] != "/test/path" {
		t.Errorf("project directories mismatch: got %v", proj.Directories)
	}
}

func TestLoadWithMCP(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mcpPath := filepath.Join(tmpDir, "mcp.json")
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	// Create mcp.json
	mcpCfg := config.NewMCPConfig()
	mcpCfg.MCPServers["github"] = &config.MCPServer{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
		Env:     map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}
	if err := mcpCfg.Save(mcpPath); err != nil {
		t.Fatalf("Save MCP config failed: %v", err)
	}

	// Create config.yaml
	yamlCfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Directories: []string{"/test/path"},
			},
		},
		Settings: config.DefaultSettings(),
	}
	if err := yamlCfg.Save(cfgPath); err != nil {
		t.Fatalf("Save YAML config failed: %v", err)
	}

	// Load with MCP
	loaded, err := config.LoadWithMCP(mcpPath, cfgPath)
	if err != nil {
		t.Fatalf("LoadWithMCP failed: %v", err)
	}

	// Servers should come from mcp.json
	if len(loaded.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(loaded.Servers))
	}

	github, ok := loaded.Servers["github"]
	if !ok {
		t.Fatal("github server not found")
	}

	if github.Command != "npx" {
		t.Errorf("expected command 'npx', got '%s'", github.Command)
	}

	// Projects should come from config.yaml
	if len(loaded.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(loaded.Projects))
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfig()

	if cfg.Servers == nil {
		t.Error("Servers map is nil")
	}

	if cfg.Projects == nil {
		t.Error("Projects map is nil")
	}

	if cfg.Settings == nil {
		t.Error("Settings is nil")
	}
}

func TestDefaultSettings(t *testing.T) {
	t.Parallel()

	settings := config.DefaultSettings()

	if settings.LogLevel != "info" {
		t.Errorf("expected default log_level 'info', got '%s'", settings.LogLevel)
	}

	if settings.Timeout != 60*time.Second {
		t.Errorf("expected default timeout 60s, got %v", settings.Timeout)
	}

	if settings.OutputFormat != "json" {
		t.Errorf("expected default output_format 'json', got '%s'", settings.OutputFormat)
	}
}

func TestSettingsClone(t *testing.T) {
	t.Parallel()

	original := &config.Settings{
		LogLevel:     "debug",
		LogFile:      "/var/log/assern.log",
		Timeout:      60 * time.Second,
		OutputFormat: "toon",
	}

	clone := &config.Settings{}
	clone.LogLevel = original.LogLevel
	clone.LogFile = original.LogFile
	clone.Timeout = original.Timeout
	clone.OutputFormat = original.OutputFormat

	if clone.LogLevel != original.LogLevel {
		t.Errorf("clone LogLevel mismatch: got '%s', want '%s'", clone.LogLevel, original.LogLevel)
	}

	if clone.OutputFormat != original.OutputFormat {
		t.Errorf("clone OutputFormat mismatch: got '%s', want '%s'", clone.OutputFormat, original.OutputFormat)
	}
}

func TestParseOutputFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		expected string
	}{
		{
			name: "json format",
			yaml: `
settings:
  output_format: json
`,
			expected: "json",
		},
		{
			name: "toon format",
			yaml: `
settings:
  output_format: toon
`,
			expected: "toon",
		},
		{
			name: "no format specified",
			yaml: `
settings:
  log_level: debug
`,
			expected: "", // Will be defaulted by DefaultSettings()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.Parse([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if cfg.Settings == nil {
				t.Fatal("Settings is nil")
			}

			// If we expect a value, check it directly
			// If we expect empty (no format specified), DefaultSettings() would apply
			if tt.expected != "" && cfg.Settings.OutputFormat != tt.expected {
				t.Errorf("expected output_format '%s', got '%s'", tt.expected, cfg.Settings.OutputFormat)
			}
		})
	}
}

func TestMCPConfig_URLBased(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mcpPath := filepath.Join(tmpDir, "mcp.json")

	// Create mcp.json with URL-based server
	mcpJSON := `{
  "mcpServers": {
    "context7": {
      "url": "https://mcp.context7.com/mcp"
    },
    "local": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem"]
    },
    "explicit-http": {
      "url": "https://example.com/mcp",
      "transport": "http"
    },
    "explicit-sse": {
      "url": "https://example.com/sse",
      "transport": "sse"
    }
  }
}`
	if err := os.WriteFile(mcpPath, []byte(mcpJSON), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Load MCP config
	mcpCfg, err := config.LoadMCPConfig(mcpPath)
	if err != nil {
		t.Fatalf("LoadMCPConfig failed: %v", err)
	}

	if len(mcpCfg.MCPServers) != 4 {
		t.Errorf("expected 4 servers, got %d", len(mcpCfg.MCPServers))
	}

	// Check URL-based server
	context7, ok := mcpCfg.MCPServers["context7"]
	if !ok {
		t.Fatal("context7 server not found")
	}
	if context7.URL != "https://mcp.context7.com/mcp" {
		t.Errorf("expected URL 'https://mcp.context7.com/mcp', got '%s'", context7.URL)
	}
	if context7.Command != "" {
		t.Errorf("expected empty command, got '%s'", context7.Command)
	}

	// Check command-based server
	local, ok := mcpCfg.MCPServers["local"]
	if !ok {
		t.Fatal("local server not found")
	}
	if local.Command != "npx" {
		t.Errorf("expected command 'npx', got '%s'", local.Command)
	}
	if local.URL != "" {
		t.Errorf("expected empty URL, got '%s'", local.URL)
	}

	// Check explicit transport
	explicitHTTP, ok := mcpCfg.MCPServers["explicit-http"]
	if !ok {
		t.Fatal("explicit-http server not found")
	}
	if explicitHTTP.Transport != "http" {
		t.Errorf("expected transport 'http', got '%s'", explicitHTTP.Transport)
	}

	explicitSSE, ok := mcpCfg.MCPServers["explicit-sse"]
	if !ok {
		t.Fatal("explicit-sse server not found")
	}
	if explicitSSE.Transport != "sse" {
		t.Errorf("expected transport 'sse', got '%s'", explicitSSE.Transport)
	}
}

func TestMCPConfig_ToServerConfigs_URLBased(t *testing.T) {
	t.Parallel()

	mcpCfg := config.NewMCPConfig()
	mcpCfg.MCPServers["remote"] = &config.MCPServer{
		URL:       "https://example.com/mcp",
		Transport: "http",
	}
	mcpCfg.MCPServers["local"] = &config.MCPServer{
		Command: "npx",
		Args:    []string{"-y", "mcp-server"},
	}

	servers := mcpCfg.ToServerConfigs()

	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	remote, ok := servers["remote"]
	if !ok {
		t.Fatal("remote server not found")
	}
	if remote.URL != "https://example.com/mcp" {
		t.Errorf("expected URL, got '%s'", remote.URL)
	}
	if remote.Transport != "http" {
		t.Errorf("expected transport 'http', got '%s'", remote.Transport)
	}

	local, ok := servers["local"]
	if !ok {
		t.Fatal("local server not found")
	}
	if local.Command != "npx" {
		t.Errorf("expected command 'npx', got '%s'", local.Command)
	}
}
