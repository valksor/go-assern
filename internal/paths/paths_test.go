package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_GlobalDir(t *testing.T) {
	cfg := &Config{
		Vendor:   "valksor",
		ToolName: "mehrhof",
	}

	dir, err := cfg.GlobalDir()
	if err != nil {
		t.Fatalf("GlobalDir() error = %v", err)
	}

	// Should contain home directory
	home, _ := os.UserHomeDir()
	if filepath.Base(dir) != "mehrhof" {
		t.Errorf("GlobalDir() = %v, want ends with mehrhof", dir)
	}

	expectedDir := filepath.Join(home, "valksor", "mehrhof")
	if dir != expectedDir {
		t.Errorf("GlobalDir() = %v, want %v", dir, expectedDir)
	}
}

func TestConfig_GlobalConfigPath(t *testing.T) {
	cfg := &Config{
		Vendor:   "valksor",
		ToolName: "assern",
	}

	path, err := cfg.GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, "valksor", "assern", "config.yaml")
	if path != expectedPath {
		t.Errorf("GlobalConfigPath() = %v, want %v", path, expectedPath)
	}
}

func TestConfig_FindLocalConfigDir(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	cfg := &Config{
		Vendor:   "valksor",
		ToolName: "test",
		LocalDir: ".test",
	}

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	err := os.MkdirAll(nestedDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// No .test directory exists yet
	found := cfg.FindLocalConfigDir(nestedDir)
	if found != "" {
		t.Errorf("FindLocalConfigDir() = %v, want empty string", found)
	}

	// Create .test directory at level1
	localDir := filepath.Join(tmpDir, "level1", ".test")
	err = os.Mkdir(localDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create local config dir: %v", err)
	}

	// Should find it from level3
	found = cfg.FindLocalConfigDir(nestedDir)
	if found != localDir {
		t.Errorf("FindLocalConfigDir() = %v, want %v", found, localDir)
	}
}

func TestConfig_LocalConfigPath(t *testing.T) {
	cfg := &Config{
		Vendor:   "valksor",
		ToolName: "mehrhof",
		LocalDir: ".mehrhof",
	}

	path := cfg.LocalConfigPath("/path/to/project/.mehrhof")
	expectedPath := "/path/to/project/.mehrhof/config.yaml"
	if path != expectedPath {
		t.Errorf("LocalConfigPath() = %v, want %v", path, expectedPath)
	}
}

func TestConfig_EnsureGlobalDir(t *testing.T) {
	tmpDir := t.TempDir()

	restore := SetHomeDirForTesting(tmpDir)
	defer restore()

	cfg := &Config{
		Vendor:   "valksor",
		ToolName: "testtool",
	}

	dir, err := cfg.EnsureGlobalDir()
	if err != nil {
		t.Fatalf("EnsureGlobalDir() error = %v", err)
	}

	expectedDir := filepath.Join(tmpDir, "valksor", "testtool")
	if dir != expectedDir {
		t.Errorf("EnsureGlobalDir() = %v, want %v", dir, expectedDir)
	}

	// Verify directory exists
	if info, err := os.Stat(dir); err != nil {
		t.Errorf("Directory not created: %v", err)
	} else if !info.IsDir() {
		t.Errorf("Path is not a directory: %v", dir)
	}
}

func TestConfig_EnsureLocalDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Vendor:   "valksor",
		ToolName: "test",
		LocalDir: ".test",
	}

	dir, err := cfg.EnsureLocalDir(tmpDir)
	if err != nil {
		t.Fatalf("EnsureLocalDir() error = %v", err)
	}

	expectedDir := filepath.Join(tmpDir, ".test")
	if dir != expectedDir {
		t.Errorf("EnsureLocalDir() = %v, want %v", dir, expectedDir)
	}

	// Verify directory exists
	if info, err := os.Stat(dir); err != nil {
		t.Errorf("Directory not created: %v", err)
	} else if !info.IsDir() {
		t.Errorf("Path is not a directory: %v", dir)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(filePath, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !FileExists(filePath) {
		t.Errorf("FileExists(%v) = false, want true", filePath)
	}

	// Test non-existent file
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	if FileExists(nonExistent) {
		t.Errorf("FileExists(%v) = true, want false", nonExistent)
	}

	// Test directory (should return false for directories)
	if FileExists(tmpDir) {
		t.Errorf("FileExists(%v) = true, want false for directory", tmpDir)
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test existing directory
	if !DirExists(tmpDir) {
		t.Errorf("DirExists(%v) = false, want true", tmpDir)
	}

	// Test non-existent directory
	nonExistent := filepath.Join(tmpDir, "nonexistent")
	if DirExists(nonExistent) {
		t.Errorf("DirExists(%v) = true, want false", nonExistent)
	}

	// Test file (should return false for files)
	filePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(filePath, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if DirExists(filePath) {
		t.Errorf("DirExists(%v) = true, want false for file", filePath)
	}
}

func TestExpandPath(t *testing.T) {
	restore := SetHomeDirForTesting("/home/testuser")
	defer restore()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde alone",
			input:    "~",
			expected: "/home/testuser",
		},
		{
			name:     "tilde with slash",
			input:    "~/path/to/file",
			expected: "/home/testuser/path/to/file",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandPath(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
