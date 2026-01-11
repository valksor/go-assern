package config_test

import (
	"os"
	"path/filepath"
	"testing"

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
