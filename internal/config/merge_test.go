package config_test

import (
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

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
