package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockHomeDir sets up a mock home directory for testing and returns a cleanup function.
func mockHomeDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	restore := SetHomeDirForTesting(tmpDir)
	t.Cleanup(restore)

	return tmpDir
}

func TestGlobalDir(t *testing.T) {
	home := mockHomeDir(t)

	dir, err := GlobalDir()
	if err != nil {
		t.Fatalf("GlobalDir() error = %v", err)
	}

	if dir == "" {
		t.Error("GlobalDir() returned empty string")
	}

	// Should be under the mock home
	if !strings.HasPrefix(dir, home) {
		t.Errorf("GlobalDir() = %s, expected to be under %s", dir, home)
	}

	// Should contain .valksor/assern
	if !strings.Contains(dir, ".valksor") {
		t.Errorf("GlobalDir() = %s, expected to contain .valksor", dir)
	}
	if !strings.Contains(dir, "assern") {
		t.Errorf("GlobalDir() = %s, expected to contain assern", dir)
	}

	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("GlobalDir() = %s, expected absolute path", dir)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	mockHomeDir(t)

	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() error = %v", err)
	}

	if path == "" {
		t.Error("GlobalConfigPath() returned empty string")
	}

	// Should end with config.yaml
	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("GlobalConfigPath() = %s, expected to end with config.yaml", path)
	}
}

func TestGlobalEnvPath(t *testing.T) {
	mockHomeDir(t)

	path, err := GlobalEnvPath()
	if err != nil {
		t.Fatalf("GlobalEnvPath() error = %v", err)
	}

	if path == "" {
		t.Error("GlobalEnvPath() returned empty string")
	}

	// Should end with .env
	if !strings.HasSuffix(path, ".env") {
		t.Errorf("GlobalEnvPath() = %s, expected to end with .env", path)
	}
}

func TestFindLocalConfigDir_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a nested directory structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .assern directory in the middle level
	assernDir := filepath.Join(tmpDir, "a", ".assern")
	if err := os.Mkdir(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Search from nested directory - should find the .assern dir
	found := FindLocalConfigDir(nestedDir)
	if found != assernDir {
		t.Errorf("FindLocalConfigDir() = %s, want %s", found, assernDir)
	}
}

func TestFindLocalConfigDir_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	found := FindLocalConfigDir(tmpDir)
	if found != "" {
		t.Errorf("FindLocalConfigDir() = %s, want empty string", found)
	}
}

func TestFindLocalConfigDir_FromNested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create deeply nested directory
	deepDir := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .assern in the root of tmpDir
	assernDir := filepath.Join(tmpDir, ".assern")
	if err := os.Mkdir(assernDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Search from deep nested - should walk up and find .assern
	found := FindLocalConfigDir(deepDir)
	if found != assernDir {
		t.Errorf("FindLocalConfigDir() = %s, want %s", found, assernDir)
	}
}

func TestFindLocalConfigDir_StopsAtRoot(t *testing.T) {
	t.Parallel()

	// Use a directory that definitely doesn't have .assern
	found := FindLocalConfigDir("/tmp")
	// This might find one if there's a .assern in /, but that's unlikely
	// The main thing is that it doesn't hang or error
	if found != "" && !strings.Contains(found, ".assern") {
		t.Errorf("FindLocalConfigDir() = %s, expected .assern in path", found)
	}
}

func TestLocalConfigPath(t *testing.T) {
	t.Parallel()

	assernDir := "/path/to/.assern"
	got := LocalConfigPath(assernDir)

	want := filepath.Join(assernDir, "config.yaml")
	if got != want {
		t.Errorf("LocalConfigPath() = %s, want %s", got, want)
	}
}

func TestLocalMCPPath(t *testing.T) {
	t.Parallel()

	assernDir := "/path/to/.assern"
	got := LocalMCPPath(assernDir)

	want := filepath.Join(assernDir, "mcp.json")
	if got != want {
		t.Errorf("LocalMCPPath() = %s, want %s", got, want)
	}
}

func TestGlobalMCPPath(t *testing.T) {
	mockHomeDir(t)

	path, err := GlobalMCPPath()
	if err != nil {
		t.Fatalf("GlobalMCPPath() error = %v", err)
	}

	if path == "" {
		t.Error("GlobalMCPPath() returned empty string")
	}

	// Should end with mcp.json
	if !strings.HasSuffix(path, "mcp.json") {
		t.Errorf("GlobalMCPPath() = %s, expected to end with mcp.json", path)
	}
}

func TestEnsureGlobalDir(t *testing.T) {
	mockHomeDir(t)

	dir, err := EnsureGlobalDir()
	if err != nil {
		t.Fatalf("EnsureGlobalDir() error = %v", err)
	}

	if dir == "" {
		t.Error("EnsureGlobalDir() returned empty string")
	}

	// Verify the directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("EnsureGlobalDir() directory doesn't exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("EnsureGlobalDir() path is not a directory")
	}
}

func TestEnsureLocalDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	createdDir, err := EnsureLocalDir(tmpDir)
	if err != nil {
		t.Fatalf("EnsureLocalDir() error = %v", err)
	}

	expectedDir := filepath.Join(tmpDir, ".assern")
	if createdDir != expectedDir {
		t.Errorf("EnsureLocalDir() = %s, want %s", createdDir, expectedDir)
	}

	// Verify the directory exists
	info, err := os.Stat(createdDir)
	if err != nil {
		t.Fatalf("EnsureLocalDir() directory doesn't exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("EnsureLocalDir() path is not a directory")
	}

	// Calling again should not error
	_, err = EnsureLocalDir(tmpDir)
	if err != nil {
		t.Errorf("EnsureLocalDir() called twice error = %v", err)
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test existing file
	if !FileExists(filePath) {
		t.Error("FileExists() returned false for existing file")
	}

	// Test non-existent file
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	if FileExists(nonExistent) {
		t.Error("FileExists() returned true for non-existent file")
	}

	// Test with directory - should return false (it's a directory, not a file)
	if FileExists(tmpDir) {
		t.Error("FileExists() returned true for directory")
	}
}

func TestDirExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Test existing directory
	if !DirExists(tmpDir) {
		t.Error("DirExists() returned false for existing directory")
	}

	// Test non-existent directory
	nonExistent := filepath.Join(tmpDir, "nonexistent")
	if DirExists(nonExistent) {
		t.Error("DirExists() returned true for non-existent directory")
	}

	// Test with file - should return false (it's a file, not a directory)
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if DirExists(filePath) {
		t.Error("DirExists() returned true for file")
	}
}

func TestExpandPath(t *testing.T) {
	home := mockHomeDir(t)

	t.Run("empty string", func(t *testing.T) {
		got := ExpandPath("")
		if got != "" {
			t.Errorf("ExpandPath(\"\") = %q, want empty string", got)
		}
	})

	t.Run("no tilde", func(t *testing.T) {
		input := "/absolute/path/to/file"
		got := ExpandPath(input)
		if got != input {
			t.Errorf("ExpandPath(%q) = %q, want %q", input, got, input)
		}
	})

	t.Run("just tilde", func(t *testing.T) {
		got := ExpandPath("~")
		if got != home {
			t.Errorf("ExpandPath(\"~\") = %q, want %q", got, home)
		}
	})

	t.Run("tilde with slash", func(t *testing.T) {
		got := ExpandPath("~/subdir")
		expected := filepath.Join(home, "subdir")
		if got != expected {
			t.Errorf("ExpandPath(\"~/subdir\") = %q, want %q", got, expected)
		}
	})

	t.Run("tilde with separator", func(t *testing.T) {
		input := "~" + string(filepath.Separator) + "path"
		got := ExpandPath(input)
		expected := filepath.Join(home, "path")
		if got != expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", input, got, expected)
		}
	})

	t.Run("tilde in middle of path", func(t *testing.T) {
		// Tilde not at start should not be expanded
		input := "/path/~to/file"
		got := ExpandPath(input)
		if got != input {
			t.Errorf("ExpandPath(%q) = %q, want %q", input, got, input)
		}
	})

	t.Run("tilde followed by non-separator", func(t *testing.T) {
		// ~ followed by non-separator should not be expanded
		input := "~user/path"
		got := ExpandPath(input)
		if got != input {
			t.Errorf("ExpandPath(%q) = %q, want %q", input, got, input)
		}
	})
}
