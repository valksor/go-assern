package project_test

import (
	"path/filepath"
	"testing"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/project"
)

func TestRegistry_Match_ExactPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "work", "myproject")

	cfg := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Directories: []string{testDir},
			},
		},
	}

	registry := project.NewRegistry(cfg)
	match := registry.Match(testDir)

	if match == nil {
		t.Fatal("expected match, got nil")
	}

	if match.Name != "myproject" {
		t.Errorf("expected project 'myproject', got '%s'", match.Name)
	}
}

func TestRegistry_Match_GlobPattern(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "work", "acme", "repo1")
	pattern := filepath.Join(tmpDir, "work", "acme", "*")

	cfg := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"acme": {
				Directories: []string{pattern},
			},
		},
	}

	registry := project.NewRegistry(cfg)
	match := registry.Match(testDir)

	if match == nil {
		t.Fatal("expected match, got nil")
	}

	if match.Name != "acme" {
		t.Errorf("expected project 'acme', got '%s'", match.Name)
	}
}

func TestRegistry_Match_NoMatch(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Directories: []string{"/specific/path"},
			},
		},
	}

	registry := project.NewRegistry(cfg)
	match := registry.Match("/different/path")

	if match != nil {
		t.Errorf("expected no match, got project '%s'", match.Name)
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"project1": {},
			"project2": {},
			"project3": {},
		},
	}

	registry := project.NewRegistry(cfg)
	names := registry.List()

	if len(names) != 3 {
		t.Errorf("expected 3 projects, got %d", len(names))
	}
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Projects: map[string]*config.ProjectConfig{
			"myproject": {
				Directories: []string{"/path"},
			},
		},
	}

	registry := project.NewRegistry(cfg)

	proj := registry.Get("myproject")
	if proj == nil {
		t.Fatal("expected project, got nil")
	}

	missing := registry.Get("nonexistent")
	if missing != nil {
		t.Error("expected nil for nonexistent project")
	}
}

func TestRegistry_NilConfig(t *testing.T) {
	t.Parallel()

	registry := project.NewRegistry(nil)

	if registry == nil {
		t.Fatal("registry should not be nil")
	}

	match := registry.Match("/any/path")
	if match != nil {
		t.Error("expected nil match for nil config")
	}
}
