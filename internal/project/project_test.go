package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/project"
)

func TestNewDetector(t *testing.T) {
	t.Parallel()

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestDetect_LocalConfigWithProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .assern directory with config.yaml
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(assernDir, "config.yaml")
	configContent := `
project: "myproject"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	if ctx.Name != "myproject" {
		t.Errorf("Detect() Name = %q, want 'myproject'", ctx.Name)
	}

	if ctx.Source != project.SourceLocal {
		t.Errorf("Detect() Source = %q, want 'local'", ctx.Source)
	}

	if ctx.LocalConfigDir != assernDir {
		t.Errorf("Detect() LocalConfigDir = %q, want %q", ctx.LocalConfigDir, assernDir)
	}
}

func TestDetect_LocalConfigWithoutProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .assern directory without config.yaml
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	// Should use directory name as project name
	dirName := filepath.Base(tmpDir)
	if ctx.Name != dirName {
		t.Errorf("Detect() Name = %q, want %q (directory name)", ctx.Name, dirName)
	}

	if ctx.Source != project.SourceLocal {
		t.Errorf("Detect() Source = %q, want 'local'", ctx.Source)
	}
}

func TestDetect_RegistryMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := config.NewConfig()
	projDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg.Projects["myproject"] = &config.ProjectConfig{
		Directories: []string{projDir},
	}

	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(projDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	if ctx.Name != "myproject" {
		t.Errorf("Detect() Name = %q, want 'myproject'", ctx.Name)
	}

	if ctx.Source != project.SourceRegistry {
		t.Errorf("Detect() Source = %q, want 'registry'", ctx.Source)
	}
}

func TestDetect_NoMatch(t *testing.T) {
	t.Parallel()

	// Create a directory with a known name for testing
	baseDir := t.TempDir()
	testDir := filepath.Join(baseDir, "my-test-project")
	if err := os.Mkdir(testDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(testDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	// Should auto-detect project name from directory basename
	if ctx.Name != "my-test-project" {
		t.Errorf("Detect() Name = %q, want 'my-test-project'", ctx.Name)
	}

	if ctx.Source != project.SourceAutoDetect {
		t.Errorf("Detect() Source = %q, want 'auto'", ctx.Source)
	}
}

func TestDetect_ExplicitFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	explicitProject := "explicit_project"
	ctx, err := detector.DetectWithExplicit(tmpDir, explicitProject)
	if err != nil {
		t.Fatalf("DetectWithExplicit() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("DetectWithExplicit() returned nil context")
	}

	if ctx.Name != explicitProject {
		t.Errorf("DetectWithExplicit() Name = %q, want %q", ctx.Name, explicitProject)
	}

	if ctx.Source != project.SourceExplicit {
		t.Errorf("DetectWithExplicit() Source = %q, want 'explicit'", ctx.Source)
	}

	// Directory should still be resolved to absolute path
	if !filepath.IsAbs(ctx.Directory) {
		t.Errorf("DetectWithExplicit() Directory = %q, want absolute path", ctx.Directory)
	}
}

func TestDetect_ExplicitFlagOverridesLocal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .assern directory with config.yaml
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(assernDir, "config.yaml")
	configContent := `
project: "local_project"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	explicitProject := "explicit_project"
	ctx, err := detector.DetectWithExplicit(tmpDir, explicitProject)
	if err != nil {
		t.Fatalf("DetectWithExplicit() error = %v", err)
	}

	if ctx.Name != explicitProject {
		t.Errorf("DetectWithExplicit() Name = %q, want %q (explicit should override local)", ctx.Name, explicitProject)
	}

	if ctx.Source != project.SourceExplicit {
		t.Errorf("DetectWithExplicit() Source = %q, want 'explicit'", ctx.Source)
	}
}

func TestDetectFromCwd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .assern directory
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	// Change to tmpDir for testing
	t.Chdir(tmpDir)

	ctx, err := detector.DetectFromCwd()
	if err != nil {
		t.Fatalf("DetectFromCwd() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("DetectFromCwd() returned nil context")
	}

	if ctx.Source != project.SourceLocal {
		t.Errorf("DetectFromCwd() Source = %q, want 'local'", ctx.Source)
	}
}

func TestRequireProject_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .assern directory
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	ctx, err := detector.RequireProject(tmpDir, "")
	if err != nil {
		t.Fatalf("RequireProject() error = %v", err)
	}

	if ctx.Source == project.SourceNone {
		t.Error("RequireProject() returned SourceNone when project exists")
	}
}

func TestRequireProject_AutoDetect(t *testing.T) {
	t.Parallel()

	// Create a directory with a known name
	baseDir := t.TempDir()
	testDir := filepath.Join(baseDir, "auto-detected-project")
	if err := os.Mkdir(testDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	// With auto-detection, RequireProject should succeed even without explicit config
	ctx, err := detector.RequireProject(testDir, "")
	if err != nil {
		t.Fatalf("RequireProject() error = %v, expected success with auto-detection", err)
	}

	if ctx.Name != "auto-detected-project" {
		t.Errorf("RequireProject() Name = %q, want 'auto-detected-project'", ctx.Name)
	}

	if ctx.Source != project.SourceAutoDetect {
		t.Errorf("RequireProject() Source = %q, want 'auto'", ctx.Source)
	}
}

func TestRequireProject_Explicit(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	explicitProject := "explicit_project"
	ctx, err := detector.RequireProject(tmpDir, explicitProject)
	if err != nil {
		t.Fatalf("RequireProject() error = %v", err)
	}

	if ctx.Name != explicitProject {
		t.Errorf("RequireProject() Name = %q, want %q", ctx.Name, explicitProject)
	}
}

func TestContext_Fields(t *testing.T) {
	t.Parallel()

	ctx := &project.Context{
		Name:           "test_project",
		Directory:      "/path/to/project",
		LocalConfigDir: "/path/to/project/.assern",
		Source:         project.SourceLocal,
	}

	if ctx.Name != "test_project" {
		t.Errorf("Context Name = %q, want 'test_project'", ctx.Name)
	}

	if ctx.Directory != "/path/to/project" {
		t.Errorf("Context Directory = %q, want '/path/to/project'", ctx.Directory)
	}

	if ctx.LocalConfigDir != "/path/to/project/.assern" {
		t.Errorf("Context LocalConfigDir = %q, want '/path/to/project/.assern'", ctx.LocalConfigDir)
	}

	if ctx.Source != project.SourceLocal {
		t.Errorf("Context Source = %q, want 'local'", ctx.Source)
	}
}

func TestDetect_NestedDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .assern in parent directory
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create nested directory
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(nestedDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	// Should find .assern in parent
	if ctx.LocalConfigDir != assernDir {
		t.Errorf("Detect() LocalConfigDir = %q, want %q", ctx.LocalConfigDir, assernDir)
	}

	// Directory should be the parent (where .assern is)
	expectedDir := tmpDir
	if ctx.Directory != expectedDir {
		t.Errorf("Detect() Directory = %q, want %q", ctx.Directory, expectedDir)
	}
}

func TestDetect_RegistryGlobPattern(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a project directory that matches a glob pattern
	projectDir := filepath.Join(tmpDir, "projects", "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	// Register with glob pattern - the test directory path varies
	// so we'll use the actual path
	cfg.Projects["myproject"] = &config.ProjectConfig{
		Directories: []string{filepath.Join(tmpDir, "projects", "*")},
	}

	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(projectDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	if ctx.Name != "myproject" {
		t.Errorf("Detect() Name = %q, want 'myproject'", ctx.Name)
	}

	if ctx.Source != project.SourceRegistry {
		t.Errorf("Detect() Source = %q, want 'registry'", ctx.Source)
	}
}

func TestDetect_LocalConfigWithServers(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .assern directory with config.yaml containing servers
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.MkdirAll(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(assernDir, "config.yaml")
	configContent := `
project: "myproject"
servers:
  test_server:
    command: echo
    args: ["test"]
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.NewConfig()
	detector := project.NewDetector(cfg)

	ctx, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if ctx == nil {
		t.Fatal("Detect() returned nil context")
	}

	if ctx.LocalConfig == nil {
		t.Error("Detect() LocalConfig is nil")
	} else {
		if ctx.LocalConfig.Servers == nil {
			t.Error("Detect() LocalConfig.Servers is nil")
		} else if _, ok := ctx.LocalConfig.Servers["test_server"]; !ok {
			t.Error("Detect() LocalConfig.Servers does not contain test_server")
		}
	}
}
