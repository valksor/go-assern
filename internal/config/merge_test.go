package config_test

import (
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

func TestMerge_OverlayMode(t *testing.T) {
	t.Parallel()

	base := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"github": {
				Command:   "npx",
				Args:      []string{"-y", "server-github"},
				Env:       map[string]string{"TOKEN": "global", "OTHER": "value"},
				MergeMode: config.MergeModeOverlay,
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Env: map[string]string{"TOKEN": "project"},
			},
		},
	}

	result := config.Merge(base, "myproject", nil)

	// Check TOKEN was overridden
	if result.Servers["github"].Env["TOKEN"] != "project" {
		t.Errorf("expected TOKEN='project', got '%s'", result.Servers["github"].Env["TOKEN"])
	}

	// Check OTHER was preserved (overlay mode)
	if result.Servers["github"].Env["OTHER"] != "value" {
		t.Errorf("expected OTHER='value', got '%s'", result.Servers["github"].Env["OTHER"])
	}
}

func TestMerge_ReplaceMode(t *testing.T) {
	t.Parallel()

	base := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"github": {
				Command:   "npx",
				Env:       map[string]string{"TOKEN": "global", "OTHER": "value"},
				MergeMode: config.MergeModeReplace,
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Env: map[string]string{"TOKEN": "project"},
			},
		},
	}

	result := config.Merge(base, "myproject", nil)

	// Check TOKEN was set
	if result.Servers["github"].Env["TOKEN"] != "project" {
		t.Errorf("expected TOKEN='project', got '%s'", result.Servers["github"].Env["TOKEN"])
	}

	// Check OTHER was removed (replace mode)
	if _, exists := result.Servers["github"].Env["OTHER"]; exists {
		t.Error("OTHER should not exist in replace mode")
	}
}

func TestMerge_ProjectServerOverride(t *testing.T) {
	t.Parallel()

	base := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"github": {
				Command: "npx",
				Args:    []string{"original"},
				Env:     map[string]string{"TOKEN": "global"},
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Servers: map[string]*config.ServerConfig{
					"github": {
						Env: map[string]string{"TOKEN": "project"},
					},
				},
			},
		},
	}

	result := config.Merge(base, "myproject", nil)

	// Check args preserved from global
	if len(result.Servers["github"].Args) != 1 || result.Servers["github"].Args[0] != "original" {
		t.Error("args should be preserved from global config")
	}

	// Check env overridden from project
	if result.Servers["github"].Env["TOKEN"] != "project" {
		t.Errorf("expected TOKEN='project', got '%s'", result.Servers["github"].Env["TOKEN"])
	}
}

func TestMerge_LocalProjectOverride(t *testing.T) {
	t.Parallel()

	base := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"github": {
				Command: "npx",
				Env:     map[string]string{"TOKEN": "global"},
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Env: map[string]string{"TOKEN": "registry"},
			},
		},
	}

	local := &config.LocalProjectConfig{
		Project: "myproject",
		Env:     map[string]string{"TOKEN": "local"},
	}

	result := config.Merge(base, "myproject", local)

	// Local should take precedence
	if result.Servers["github"].Env["TOKEN"] != "local" {
		t.Errorf("expected TOKEN='local', got '%s'", result.Servers["github"].Env["TOKEN"])
	}
}

func TestMerge_NewServerFromProject(t *testing.T) {
	t.Parallel()

	base := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"github": {
				Command: "npx",
			},
		},
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Servers: map[string]*config.ServerConfig{
					"jira": {
						Command: "jira-server",
						Env:     map[string]string{"JIRA_TOKEN": "token"},
					},
				},
			},
		},
	}

	result := config.Merge(base, "myproject", nil)

	// Check new server was added
	if _, exists := result.Servers["jira"]; !exists {
		t.Fatal("jira server should exist")
	}

	if result.Servers["jira"].Command != "jira-server" {
		t.Error("jira server command not set correctly")
	}
}

func TestGetEffectiveServers(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"active": {
				Command:  "cmd",
				Disabled: false,
			},
			"disabled": {
				Command:  "cmd",
				Disabled: true,
			},
			"empty": {
				Command: "",
			},
		},
	}

	effective := config.GetEffectiveServers(cfg)

	if len(effective) != 1 {
		t.Errorf("expected 1 effective server, got %d", len(effective))
	}

	if _, exists := effective["active"]; !exists {
		t.Error("active server should be in effective servers")
	}
}

func TestRegisterProject(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfig()

	// First registration
	cfg.RegisterProject("myproject", "/path/to/project")

	if len(cfg.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(cfg.Projects))
	}

	if len(cfg.Projects["myproject"].Directories) != 1 {
		t.Error("expected 1 directory")
	}

	// Register same directory again - should not duplicate
	cfg.RegisterProject("myproject", "/path/to/project")

	if len(cfg.Projects["myproject"].Directories) != 1 {
		t.Error("directory should not be duplicated")
	}

	// Register different directory
	cfg.RegisterProject("myproject", "/another/path")

	if len(cfg.Projects["myproject"].Directories) != 2 {
		t.Error("expected 2 directories")
	}
}
