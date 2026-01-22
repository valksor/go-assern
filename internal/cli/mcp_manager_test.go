// Package cli provides interactive CLI components for assern.
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/go-assern/internal/config"
)

func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Set up test config directory
	configDir := filepath.Join(tmpDir, ".valksor", "assern")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Create test mcp.json
	mcpPath := filepath.Join(configDir, "mcp.json")
	mcpCfg := config.NewMCPConfig()
	mcpCfg.MCPServers["test-server"] = &config.MCPServer{
		Command: "node",
		Args:    []string{"server.js"},
	}
	if err := mcpCfg.Save(mcpPath); err != nil {
		t.Fatalf("failed to save mcp.json: %v", err)
	}

	// Override home directory for testing
	restore := config.SetHomeDirForTesting(tmpDir)

	return tmpDir, restore
}

func TestNewMCPManager(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	// Create manager with test directory
	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	if mgr == nil {
		t.Fatal("NewMCPManager() returned nil manager")
	}

	if mgr.globalMCP == nil {
		t.Error("globalMCP is nil")
	}

	if len(mgr.globalMCP.MCPServers) == 0 {
		t.Error("globalMCP has no servers")
	}
}

func TestMCPManagerListServers(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	servers := mgr.ListServers()

	if len(servers) == 0 {
		t.Error("ListServers() returned no servers")
	}

	// Check for test-server
	found := false
	for _, srv := range servers {
		if srv.Name == "test-server" {
			found = true
			if srv.Scope != ScopeGlobal {
				t.Errorf("test-server scope = %v, want %v", srv.Scope, ScopeGlobal)
			}
			if srv.Transport != "stdio" {
				t.Errorf("test-server transport = %v, want stdio", srv.Transport)
			}
		}
	}

	if !found {
		t.Error("test-server not found in list")
	}
}

func TestMCPManagerGetServer(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	server, scope, err := mgr.GetServer("test-server")
	if err != nil {
		t.Fatalf("GetServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("GetServer() returned nil server")
	}

	if scope != ScopeGlobal {
		t.Errorf("GetServer() scope = %v, want %v", scope, ScopeGlobal)
	}

	if server.Command != "node" {
		t.Errorf("GetServer() command = %v, want node", server.Command)
	}
}

func TestMCPManagerGetServerNotFound(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	_, _, err = mgr.GetServer("non-existent")
	if err == nil {
		t.Error("GetServer() expected error for non-existent server, got nil")
	}
}

func TestMCPManagerAddServer(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	input := &MCPInput{
		Name:      "new-server",
		Scope:     ScopeGlobal,
		Transport: "http",
		URL:       "https://example.com/mcp",
	}

	err = mgr.AddServer(input)
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	// Verify server was added
	servers := mgr.ListServers()
	found := false
	for _, srv := range servers {
		if srv.Name == "new-server" {
			found = true
			if srv.Transport != "http" {
				t.Errorf("new-server transport = %v, want http", srv.Transport)
			}
		}
	}

	if !found {
		t.Error("new-server not found after AddServer()")
	}
}

func TestMCPManagerAddServerDuplicate(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	input := &MCPInput{
		Name:      "test-server", // Already exists
		Scope:     ScopeGlobal,
		Transport: "http",
		URL:       "https://example.com/mcp",
	}

	err = mgr.AddServer(input)
	if err == nil {
		t.Error("AddServer() expected error for duplicate server name, got nil")
	}
}

func TestMCPManagerUpdateServer(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	input := &MCPInput{
		Name:      "test-server",
		Scope:     ScopeGlobal,
		Transport: "stdio",
		Command:   "python",
		Args:      []string{"-m", "server"},
	}

	err = mgr.UpdateServer("test-server", input)
	if err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	// Verify server was updated
	server, _, _ := mgr.GetServer("test-server")
	if server.Command != "python" {
		t.Errorf("UpdateServer() command = %v, want python", server.Command)
	}
}

func TestMCPManagerDeleteServer(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	err = mgr.DeleteServer([]string{"test-server"})
	if err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}

	// Verify server was deleted
	_, _, err = mgr.GetServer("test-server")
	if err == nil {
		t.Error("DeleteServer() server still exists after deletion")
	}
}

func TestMCPManagerServerNames(t *testing.T) {
	tmpDir, restore := setupTestConfig(t)
	defer restore()

	t.Chdir(tmpDir)

	mgr, err := NewMCPManager()
	if err != nil {
		t.Fatalf("NewMCPManager() error = %v", err)
	}

	global, local := mgr.ServerNames()

	if len(global) == 0 {
		t.Error("ServerNames() returned no global servers")
	}

	found := false
	for _, name := range global {
		if name == "test-server" {
			found = true
		}
	}

	if !found {
		t.Error("test-server not found in global server names")
	}

	if len(local) != 0 {
		t.Error("ServerNames() returned local servers when none exist")
	}
}

func TestDetectTransport(t *testing.T) {
	tests := []struct {
		name     string
		server   *config.MCPServer
		expected string
	}{
		{
			name: "stdio server",
			server: &config.MCPServer{
				Command: "node",
				Args:    []string{"server.js"},
			},
			expected: "stdio",
		},
		{
			name: "http server",
			server: &config.MCPServer{
				URL: "https://example.com/mcp",
			},
			expected: "http",
		},
		{
			name: "oauth server",
			server: &config.MCPServer{
				URL: "https://example.com/mcp",
				OAuth: &config.OAuthConfig{
					ClientID: "test-client",
				},
			},
			expected: "oauth-http",
		},
		{
			name:     "unknown server",
			server:   &config.MCPServer{},
			expected: "unknown",
		},
		{
			name: "explicit transport",
			server: &config.MCPServer{
				Transport: "sse",
			},
			expected: "sse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectTransport(tt.server)
			if result != tt.expected {
				t.Errorf("detectTransport() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInputToMCPServer(t *testing.T) {
	mgr := &MCPManager{}

	tests := []struct {
		name  string
		input *MCPInput
		check func(*config.MCPServer)
	}{
		{
			name: "stdio input",
			input: &MCPInput{
				Name:      "test",
				Transport: "stdio",
				Command:   "node",
				Args:      []string{"server.js"},
				Env:       map[string]string{"KEY": "value"},
				WorkDir:   "/tmp",
			},
			check: func(s *config.MCPServer) {
				if s.Command != "node" {
					t.Errorf("Command = %v, want node", s.Command)
				}
				if len(s.Args) != 1 || s.Args[0] != "server.js" {
					t.Errorf("Args = %v, want [server.js]", s.Args)
				}
				if s.Env["KEY"] != "value" {
					t.Errorf("Env[KEY] = %v, want value", s.Env["KEY"])
				}
				if s.WorkDir != "/tmp" {
					t.Errorf("WorkDir = %v, want /tmp", s.WorkDir)
				}
			},
		},
		{
			name: "http input",
			input: &MCPInput{
				Name:      "test",
				Transport: "http",
				URL:       "https://example.com/mcp",
				Headers:   map[string]string{"Authorization": "Bearer token"},
			},
			check: func(s *config.MCPServer) {
				if s.URL != "https://example.com/mcp" {
					t.Errorf("URL = %v, want https://example.com/mcp", s.URL)
				}
				if s.Headers["Authorization"] != "Bearer token" {
					t.Errorf("Headers[Authorization] = %v, want Bearer token", s.Headers["Authorization"])
				}
			},
		},
		{
			name: "oauth input",
			input: &MCPInput{
				Name:      "test",
				Transport: "oauth-http",
				URL:       "https://example.com/mcp",
				OAuth: &config.OAuthConfig{
					ClientID: "test-client",
					Scopes:   []string{"read", "write"},
				},
			},
			check: func(s *config.MCPServer) {
				if s.OAuth == nil {
					t.Error("OAuth is nil")
				}
				if s.OAuth.ClientID != "test-client" {
					t.Errorf("OAuth.ClientID = %v, want test-client", s.OAuth.ClientID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.inputToMCPServer(tt.input)
			if tt.check != nil {
				tt.check(result)
			}
		})
	}
}
