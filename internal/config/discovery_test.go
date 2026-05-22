package config_test

import (
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

func TestDiscoveryConfigIsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.DiscoveryConfig
		want bool
	}{
		{name: "nil", cfg: nil, want: false},
		{name: "zero value", cfg: &config.DiscoveryConfig{}, want: false},
		{name: "enabled", cfg: &config.DiscoveryConfig{Enabled: true}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscoveryConfigEffectiveMaxResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.DiscoveryConfig
		want int
	}{
		{name: "nil uses default", cfg: nil, want: config.DefaultDiscoveryMaxResults},
		{name: "zero uses default", cfg: &config.DiscoveryConfig{MaxResults: 0}, want: config.DefaultDiscoveryMaxResults},
		{name: "negative uses default", cfg: &config.DiscoveryConfig{MaxResults: -3}, want: config.DefaultDiscoveryMaxResults},
		{name: "explicit", cfg: &config.DiscoveryConfig{MaxResults: 5}, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.EffectiveMaxResults(); got != tt.want {
				t.Errorf("EffectiveMaxResults() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDiscoveryConfigEffectiveMaxLoaded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.DiscoveryConfig
		want int
	}{
		{name: "nil uses default", cfg: nil, want: config.DefaultDiscoveryMaxLoaded},
		{name: "zero uses default", cfg: &config.DiscoveryConfig{MaxLoaded: 0}, want: config.DefaultDiscoveryMaxLoaded},
		{name: "negative means unlimited", cfg: &config.DiscoveryConfig{MaxLoaded: -1}, want: 0},
		{name: "explicit", cfg: &config.DiscoveryConfig{MaxLoaded: 12}, want: 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.cfg.EffectiveMaxLoaded(); got != tt.want {
				t.Errorf("EffectiveMaxLoaded() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDiscoveryConfigClone(t *testing.T) {
	t.Parallel()

	original := &config.DiscoveryConfig{
		Enabled:    true,
		Pinned:     []string{"github_search", "linear_ticket"},
		MaxResults: 7,
		MaxLoaded:  20,
	}

	clone := original.Clone()

	if clone == original {
		t.Fatal("Clone() returned the same pointer")
	}

	// Mutating the clone's slice must not affect the original.
	clone.Pinned[0] = "mutated"

	if original.Pinned[0] != "github_search" {
		t.Error("Clone() did not deep-copy Pinned slice")
	}

	var nilCfg *config.DiscoveryConfig
	if nilCfg.Clone() != nil {
		t.Error("Clone() on nil should return nil")
	}
}

func TestParseDiscoverySettings(t *testing.T) {
	t.Parallel()

	yaml := `
settings:
  discovery:
    enabled: true
    pinned:
      - github_search
    max_results: 8
    max_loaded: 25
`

	cfg, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Settings == nil || cfg.Settings.Discovery == nil {
		t.Fatal("expected discovery settings to be parsed")
	}

	d := cfg.Settings.Discovery
	if !d.IsEnabled() {
		t.Error("discovery should be enabled")
	}

	if d.EffectiveMaxResults() != 8 {
		t.Errorf("max_results = %d, want 8", d.EffectiveMaxResults())
	}

	if d.EffectiveMaxLoaded() != 25 {
		t.Errorf("max_loaded = %d, want 25", d.EffectiveMaxLoaded())
	}

	if len(d.Pinned) != 1 || d.Pinned[0] != "github_search" {
		t.Errorf("pinned = %v, want [github_search]", d.Pinned)
	}
}
