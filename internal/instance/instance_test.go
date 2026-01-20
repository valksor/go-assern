package instance

import (
	"testing"
)

func TestSharingEnabled_Default(t *testing.T) {
	// Clear the env var to test default behavior
	// Note: t.Setenv handles cleanup automatically
	t.Setenv(EnvNoSharing, "")

	// After t.Setenv with empty value, the env var exists but is empty
	// SharingEnabled checks for empty string, so this should return true
	if !SharingEnabled() {
		t.Error("SharingEnabled() should return true when env var is empty")
	}
}

func TestSharingEnabled_Disabled(t *testing.T) {
	t.Setenv(EnvNoSharing, "1")

	if SharingEnabled() {
		t.Error("SharingEnabled() should return false when ASSERN_NO_INSTANCE_SHARING is set")
	}
}

func TestSharingEnabled_DisabledAnyValue(t *testing.T) {
	t.Setenv(EnvNoSharing, "true")

	if SharingEnabled() {
		t.Error("SharingEnabled() should return false when ASSERN_NO_INSTANCE_SHARING is set to any non-empty value")
	}
}

func TestInfo_Fields(t *testing.T) {
	t.Parallel()

	info := Info{
		PID:        12345,
		SocketPath: "/tmp/test.sock",
	}

	if info.PID != 12345 {
		t.Errorf("Info.PID = %d, want 12345", info.PID)
	}

	if info.SocketPath != "/tmp/test.sock" {
		t.Errorf("Info.SocketPath = %s, want /tmp/test.sock", info.SocketPath)
	}
}

func TestEnvNoSharing_Constant(t *testing.T) {
	t.Parallel()

	if EnvNoSharing != "ASSERN_NO_INSTANCE_SHARING" {
		t.Errorf("EnvNoSharing = %s, want ASSERN_NO_INSTANCE_SHARING", EnvNoSharing)
	}
}
