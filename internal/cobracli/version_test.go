package cobracli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/valksor/go-assern/internal/version"
)

func TestNewVersionCommand(t *testing.T) {
	version.Set("1.2.3", "abc123", "2026-01-15T12:00:00Z")
	t.Cleanup(func() { version.Set("dev", "none", "unknown") })

	cmd := NewVersionCommand("assern")
	if cmd.Use != "version" {
		t.Errorf("Use = %q, want %q", cmd.Use, "version")
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, nil)

	out := buf.String()
	for _, want := range []string{"assern", "1.2.3", "abc123", "2026-01-15T12:00:00Z"} {
		if !strings.Contains(out, want) {
			t.Errorf("version output missing %q; got:\n%s", want, out)
		}
	}
}
