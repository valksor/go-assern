package version

import (
	"strings"
	"testing"
)

func TestInfo(t *testing.T) {
	Set("1.0.0", "abc123", "2024-01-15T12:00:00Z")
	defer Set("dev", "none", "unknown") // Reset after test

	info := Info("testapp")

	if !strings.Contains(info, "testapp") {
		t.Errorf("Info() should contain app name")
	}
	if !strings.Contains(info, "1.0.0") {
		t.Errorf("Info() should contain version")
	}
	if !strings.Contains(info, "abc123") {
		t.Errorf("Info() should contain commit")
	}
	if !strings.Contains(info, "2024-01-15T12:00:00Z") {
		t.Errorf("Info() should contain build time")
	}
	if !strings.Contains(info, "Go:") {
		t.Errorf("Info() should contain Go version")
	}
}

func TestShort(t *testing.T) {
	Set("2.0.0", "def456", "2024-02-20T12:00:00Z")
	defer Set("dev", "none", "unknown") // Reset after test

	if got := Short(); got != "2.0.0" {
		t.Errorf("Short() = %v, want %v", got, "2.0.0")
	}
}

func TestDefaultValues(t *testing.T) {
	Set("dev", "none", "unknown")
	defer Set("dev", "none", "unknown")

	info := Info("myapp")
	short := Short()

	if !strings.Contains(info, "dev") {
		t.Errorf("Default version should be 'dev'")
	}
	if !strings.Contains(info, "none") {
		t.Errorf("Default commit should be 'none'")
	}
	if !strings.Contains(info, "unknown") {
		t.Errorf("Default build time should be 'unknown'")
	}
	if short != "dev" {
		t.Errorf("Short() = %v, want %v", short, "dev")
	}
}

func TestSet(t *testing.T) {
	Set("3.0.0", "ghi789", "2024-03-25T12:00:00Z")

	if Version != "3.0.0" {
		t.Errorf("Version = %v, want %v", Version, "3.0.0")
	}
	if Commit != "ghi789" {
		t.Errorf("Commit = %v, want %v", Commit, "ghi789")
	}
	if BuildTime != "2024-03-25T12:00:00Z" {
		t.Errorf("BuildTime = %v, want %v", BuildTime, "2024-03-25T12:00:00Z")
	}
}
