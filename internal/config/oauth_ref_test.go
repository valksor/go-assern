package config_test

import (
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

func buildWithAuth(t *testing.T, mcp *config.MCPConfig, global *config.Config) *config.Config {
	t.Helper()

	return config.BuildEffectiveConfig(mcp, global, nil, nil, "")
}

func TestBuildEffectiveConfigResolvesOAuthRef(t *testing.T) {
	t.Parallel()

	mcp := &config.MCPConfig{
		MCPServers: map[string]*config.MCPServer{
			"gh": {URL: "https://api.example.com", Transport: "oauth-http", OAuthRef: "github"},
		},
	}

	global := &config.Config{
		Auth: map[string]*config.OAuthConfig{
			"github": {ClientID: "cid", Scopes: []string{"repo"}},
		},
	}

	eff := buildWithAuth(t, mcp, global)

	srv := eff.Servers["gh"]
	if srv == nil || srv.OAuth == nil {
		t.Fatalf("oauth_ref was not resolved: %+v", srv)
	}

	if srv.OAuth.ClientID != "cid" {
		t.Errorf("resolved ClientID = %q, want cid", srv.OAuth.ClientID)
	}

	// Resolution must clone, not alias, the profile.
	srv.OAuth.ClientID = "mutated"
	if global.Auth["github"].ClientID != "cid" {
		t.Error("resolveOAuthRefs aliased the profile instead of cloning it")
	}
}

func TestBuildEffectiveConfigInlineOAuthWins(t *testing.T) {
	t.Parallel()

	mcp := &config.MCPConfig{
		MCPServers: map[string]*config.MCPServer{
			"gh": {
				URL:       "https://api.example.com",
				Transport: "oauth-http",
				OAuthRef:  "github",
				OAuth:     &config.OAuthConfig{ClientID: "inline"},
			},
		},
	}

	global := &config.Config{
		Auth: map[string]*config.OAuthConfig{
			"github": {ClientID: "profile"},
		},
	}

	eff := buildWithAuth(t, mcp, global)

	if got := eff.Servers["gh"].OAuth.ClientID; got != "inline" {
		t.Errorf("inline OAuth should win: ClientID = %q, want inline", got)
	}
}

func TestBuildEffectiveConfigUnknownRefLeftNil(t *testing.T) {
	t.Parallel()

	mcp := &config.MCPConfig{
		MCPServers: map[string]*config.MCPServer{
			"gh": {URL: "https://api.example.com", Transport: "oauth-http", OAuthRef: "missing"},
		},
	}

	eff := buildWithAuth(t, mcp, &config.Config{})

	if eff.Servers["gh"].OAuth != nil {
		t.Errorf("unknown oauth_ref should leave OAuth nil, got %+v", eff.Servers["gh"].OAuth)
	}
}

// TestBuildEffectiveConfigCarriesSettings guards against the previous bug where
// BuildEffectiveConfig dropped Aliases and Discovery from global settings.
func TestBuildEffectiveConfigCarriesSettings(t *testing.T) {
	t.Parallel()

	global := &config.Config{
		Settings: &config.Settings{
			Aliases:   map[string]string{"gh": "github_search_repos"},
			Discovery: &config.DiscoveryConfig{Enabled: true, MaxResults: 5},
			CodeMode:  &config.CodeModeConfig{Enabled: true, MaxToolCalls: 7},
		},
	}

	eff := buildWithAuth(t, config.NewMCPConfig(), global)

	if eff.Settings == nil {
		t.Fatal("settings dropped entirely")
	}

	if !eff.Settings.Discovery.IsEnabled() {
		t.Error("discovery settings were dropped by BuildEffectiveConfig")
	}

	if eff.Settings.Discovery.EffectiveMaxResults() != 5 {
		t.Errorf("discovery max_results = %d, want 5", eff.Settings.Discovery.EffectiveMaxResults())
	}

	if !eff.Settings.CodeMode.IsEnabled() || eff.Settings.CodeMode.MaxToolCalls != 7 {
		t.Errorf("code_mode settings were dropped by BuildEffectiveConfig: %+v", eff.Settings.CodeMode)
	}

	if eff.Settings.Aliases["gh"] != "github_search_repos" {
		t.Errorf("aliases were dropped: %v", eff.Settings.Aliases)
	}
}
