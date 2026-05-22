package project

import (
	"os"
	"path/filepath"
	"testing"
)

// mockPathResolver is a mock implementation of PathResolver for testing.
type mockPathResolver struct {
	localConfigDir string
	localConfigMap map[string]string
}

func (m *mockPathResolver) FindLocalConfigDir(startDir string) string {
	return m.localConfigDir
}

func (m *mockPathResolver) LocalConfigPath(localDir string) string {
	if path, ok := m.localConfigMap[localDir]; ok {
		return path
	}

	return filepath.Join(localDir, "config.yaml")
}

func (m *mockPathResolver) FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}

	return true
}

// mockConfigWithProject is a mock config that has a Project field.
type mockConfigWithProject struct {
	Project string
}

func TestNewDetector(t *testing.T) {
	registry := NewRegistry()

	detector := NewDetector(nil, ".test", registry)

	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}

	if detector.localDirName != ".test" {
		t.Errorf("localDirName = %v, want '.test'", detector.localDirName)
	}
}

func TestDetector_DetectAutoDetect(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector(nil, ".test", nil)

	ctx, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx.Source != SourceAutoDetect {
		t.Errorf("Source = %v, want %v", ctx.Source, SourceAutoDetect)
	}

	if ctx.Name != filepath.Base(tmpDir) {
		t.Errorf("Name = %v, want %v", ctx.Name, filepath.Base(tmpDir))
	}
}

func TestDetector_DetectLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".test")
	err := os.Mkdir(localDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	resolver := &mockPathResolver{
		localConfigDir: localDir,
	}

	detector := NewDetector(resolver, ".test", nil)

	ctx, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx.Source != SourceLocal {
		t.Errorf("Source = %v, want %v", ctx.Source, SourceLocal)
	}

	if ctx.LocalConfigDir != localDir {
		t.Errorf("LocalConfigDir = %v, want %v", ctx.LocalConfigDir, localDir)
	}
}

func TestDetector_DetectFromCwd(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	detector := NewDetector(nil, ".test", nil)

	ctx, err := detector.DetectFromCwd()
	if err != nil {
		t.Fatalf("DetectFromCwd() error = %v", err)
	}

	if ctx.Name != filepath.Base(tmpDir) {
		t.Errorf("Name = %v, want %v", ctx.Name, filepath.Base(tmpDir))
	}
}

func TestDetector_DetectWithExplicit(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector(nil, ".test", nil)

	ctx, err := detector.DetectWithExplicit(tmpDir, "explicit-project")
	if err != nil {
		t.Fatalf("DetectWithExplicit() error = %v", err)
	}

	if ctx.Source != SourceExplicit {
		t.Errorf("Source = %v, want %v", ctx.Source, SourceExplicit)
	}

	if ctx.Name != "explicit-project" {
		t.Errorf("Name = %v, want 'explicit-project'", ctx.Name)
	}
}

func TestDetector_RequireProject(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector(nil, ".test", nil)

	// Should succeed with auto-detect
	ctx, err := detector.RequireProject(tmpDir, "")
	if err != nil {
		t.Fatalf("RequireProject() error = %v", err)
	}

	if ctx.Source == SourceNone {
		t.Error("RequireProject() returned SourceNone")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	registry.Register("myproject", []string{"~/work/myproject"}, nil)

	if len(registry.List()) != 1 {
		t.Errorf("List() length = %v, want 1", len(registry.List()))
	}

	proj := registry.Get("myproject")
	if proj == nil {
		t.Fatal("Get() returned nil for registered project")
	}

	if proj.Name != "myproject" {
		t.Errorf("Name = %v, want 'myproject'", proj.Name)
	}
}

func TestRegistry_Match(t *testing.T) {
	registry := NewRegistry()

	tmpDir := t.TempDir()
	registry.Register("testproject", []string{tmpDir}, nil)

	match := registry.Match(tmpDir)
	if match == nil {
		t.Fatal("Match() returned nil for matching directory")
	}

	if match.Name != "testproject" {
		t.Errorf("Name = %v, want 'testproject'", match.Name)
	}
}

func TestRegistry_MatchGlob(t *testing.T) {
	registry := NewRegistry()

	tmpDir := t.TempDir()
	pattern := filepath.Join(tmpDir, "work", "*")
	registry.Register("testproject", []string{pattern}, nil)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "work", "myrepo")
	err := os.MkdirAll(subDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	match := registry.Match(subDir)
	if match == nil {
		t.Fatal("Match() returned nil for glob pattern")
	}

	if match.Name != "testproject" {
		t.Errorf("Name = %v, want 'testproject'", match.Name)
	}
}

func TestRegistry_MatchDoublestar(t *testing.T) {
	registry := NewRegistry()

	tmpDir := t.TempDir()
	pattern := filepath.Join(tmpDir, "repos", "**")
	registry.Register("testproject", []string{pattern}, nil)

	// Create nested directories
	nestedDir := filepath.Join(tmpDir, "repos", "org", "repo")
	err := os.MkdirAll(nestedDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	match := registry.Match(nestedDir)
	if match == nil {
		t.Fatal("Match() returned nil for doublestar pattern")
	}

	if match.Name != "testproject" {
		t.Errorf("Name = %v, want 'testproject'", match.Name)
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	list := registry.List()
	if list != nil {
		t.Errorf("List() = %v, want nil for empty registry", list)
	}

	// Add projects
	registry.Register("project1", []string{"~/work/p1"}, nil)
	registry.Register("project2", []string{"~/work/p2"}, nil)

	list = registry.List()
	if len(list) != 2 {
		t.Errorf("List() length = %v, want 2", len(list))
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	config := &mockConfigWithProject{Project: "test"}
	registry.Register("myproject", []string{"~/work/myproject"}, config)

	proj := registry.Get("myproject")
	if proj == nil {
		t.Fatal("Get() returned nil")
	}

	if proj.Config != config {
		t.Error("Config not preserved")
	}

	// Test non-existent project
	proj = registry.Get("nonexistent")
	if proj != nil {
		t.Error("Get() should return nil for non-existent project")
	}
}

func TestNewRegistryFromMap(t *testing.T) {
	projects := map[string]*RegistryProject{
		"project1": {
			Name:        "project1",
			Directories: []string{"~/work/p1"},
			Config:      nil,
		},
		"project2": {
			Name:        "project2",
			Directories: []string{"~/work/p2"},
			Config:      nil,
		},
	}

	registry := NewRegistryFromMap(projects)

	if len(registry.List()) != 2 {
		t.Errorf("List() length = %v, want 2", len(registry.List()))
	}

	proj1 := registry.Get("project1")
	if proj1 == nil {
		t.Fatal("Get('project1') returned nil")
	}
}
