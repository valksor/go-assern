// Package cli provides interactive CLI components for assern.
package cli

import (
	"slices"
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

func TestParseScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "single value",
			input: "read",
			want:  []string{"read"},
		},
		{
			name:  "comma separated with spaces",
			input: "read, write, admin",
			want:  []string{"read", "write", "admin"},
		},
		{
			name:  "comma separated without spaces",
			input: "read,write,admin",
			want:  []string{"read", "write", "admin"},
		},
		{
			name:  "trailing comma yields no empty entries",
			input: "read,write,",
			want:  []string{"read", "write"},
		},
		{
			name:  "leading and internal blank entries dropped",
			input: ",read, ,write",
			want:  []string{"read", "write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseScopes(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseScopes(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTransportNeedsOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transport string
		want      bool
	}{
		{name: "stdio", transport: transportStdio, want: false},
		{name: "http", transport: transportHTTP, want: false},
		{name: "sse", transport: transportSSE, want: false},
		{name: "oauth-http", transport: transportOAuthHTTP, want: true},
		{name: "oauth-sse", transport: transportOAuthSSE, want: true},
		{name: "unknown", transport: "websocket", want: false},
		{name: "empty", transport: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := transportNeedsOAuth(tt.transport); got != tt.want {
				t.Errorf("transportNeedsOAuth(%q) = %v, want %v", tt.transport, got, tt.want)
			}
		})
	}
}

func TestTransportConfigKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transport string
		want      string
	}{
		{name: "stdio", transport: transportStdio, want: "stdio"},
		{name: "http", transport: transportHTTP, want: "http"},
		{name: "sse", transport: transportSSE, want: "http"},
		{name: "oauth-http", transport: transportOAuthHTTP, want: "http"},
		{name: "oauth-sse", transport: transportOAuthSSE, want: "http"},
		{name: "unknown", transport: "websocket", want: ""},
		{name: "empty", transport: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := transportConfigKind(tt.transport); got != tt.want {
				t.Errorf("transportConfigKind(%q) = %v, want %v", tt.transport, got, tt.want)
			}
		})
	}
}

func TestBuildSummaryLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *MCPInput
		want  []string
	}{
		{
			name: "stdio with args and workdir",
			input: &MCPInput{
				Name:      "myserver",
				Scope:     ScopeGlobal,
				Transport: transportStdio,
				Command:   "npx",
				Args:      []string{"-y", "some-pkg"},
				WorkDir:   "/tmp/work",
			},
			want: []string{
				"  Name: myserver",
				"  Scope: global",
				"  Transport: stdio",
				"    Command: npx",
				"    Args: -y some-pkg",
				"    Working Dir: /tmp/work",
			},
		},
		{
			name: "stdio minimal without args or workdir",
			input: &MCPInput{
				Name:      "barebones",
				Scope:     ScopeGlobal,
				Transport: transportStdio,
				Command:   "node",
			},
			want: []string{
				"  Name: barebones",
				"  Scope: global",
				"  Transport: stdio",
				"    Command: node",
			},
		},
		{
			name: "http without oauth",
			input: &MCPInput{
				Name:      "remote",
				Scope:     ScopeGlobal,
				Transport: transportHTTP,
				URL:       "https://api.example.com/mcp",
			},
			want: []string{
				"  Name: remote",
				"  Scope: global",
				"  Transport: http",
				"    URL: https://api.example.com/mcp",
			},
		},
		{
			name: "oauth-http with oauth config",
			input: &MCPInput{
				Name:      "secured",
				Scope:     ScopeGlobal,
				Transport: transportOAuthHTTP,
				URL:       "https://api.example.com/mcp",
				OAuth: &config.OAuthConfig{
					ClientID: "client-123",
					Scopes:   []string{"read", "write"},
				},
			},
			want: []string{
				"  Name: secured",
				"  Scope: global",
				"  Transport: oauth-http",
				"    URL: https://api.example.com/mcp",
				"    OAuth: ClientID=client-123, Scopes=[read write]",
			},
		},
		{
			name: "disabled flag adds status line",
			input: &MCPInput{
				Name:      "off",
				Scope:     ScopeGlobal,
				Transport: transportStdio,
				Command:   "python",
				Disabled:  true,
			},
			want: []string{
				"  Name: off",
				"  Scope: global",
				"  Transport: stdio",
				"    Command: python",
				"  Status: disabled",
			},
		},
		{
			name: "project scope with project name",
			input: &MCPInput{
				Name:      "scoped",
				Scope:     ScopeProject,
				Project:   "myproj",
				Transport: transportStdio,
				Command:   "go",
			},
			want: []string{
				"  Name: scoped",
				"  Scope: project (myproj)",
				"  Transport: stdio",
				"    Command: go",
			},
		},
		{
			name: "project scope without project name",
			input: &MCPInput{
				Name:      "scoped",
				Scope:     ScopeProject,
				Transport: transportStdio,
				Command:   "go",
			},
			want: []string{
				"  Name: scoped",
				"  Scope: project",
				"  Transport: stdio",
				"    Command: go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSummaryLines(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildSummaryLines() =\n%#v\nwant\n%#v", got, tt.want)
			}
		})
	}
}
