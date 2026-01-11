package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/project"
	"github.com/valksor/go-assern/internal/version"
)

// captureOutput captures stdout during function execution.
func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	return buf.String()
}

func TestRootCmd(t *testing.T) {
	t.Parallel()

	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "assern" {
		t.Errorf("rootCmd.Use = %q, want 'assern'", rootCmd.Use)
	}

	if rootCmd.Short != "Valksor Assern - MCP Server Aggregator" {
		t.Errorf("rootCmd.Short = %q, want 'Valksor Assern - MCP Server Aggregator'", rootCmd.Short)
	}

	if !rootCmd.SilenceUsage {
		t.Error("rootCmd.SilenceUsage is false, expected true")
	}

	if !rootCmd.SilenceErrors {
		t.Error("rootCmd.SilenceErrors is false, expected true")
	}
}

func TestVersionCmd(t *testing.T) {
	t.Parallel()

	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want 'version'", versionCmd.Use)
	}

	t.Run("version output", func(t *testing.T) {
		t.Parallel()

		// Set version for testing
		oldVersion := version.Version
		version.Version = "1.0.0-test"
		defer func() { version.Version = oldVersion }()

		output := captureOutput(func() {
			versionCmd.Run(versionCmd, nil)
		})

		if !strings.Contains(output, "1.0.0-test") {
			t.Errorf("Version output does not contain version number. Got: %s", output)
		}
	})
}

func TestServeCmd(t *testing.T) {
	t.Parallel()

	if serveCmd == nil {
		t.Fatal("serveCmd is nil")
	}

	if serveCmd.Use != "serve" {
		t.Errorf("serveCmd.Use = %q, want 'serve'", serveCmd.Use)
	}

	if serveCmd.Short != "Start the MCP aggregator server" {
		t.Errorf("serveCmd.Short = %q, want 'Start the MCP aggregator server'", serveCmd.Short)
	}
}

func TestListCmd(t *testing.T) {
	t.Parallel()

	if listCmd == nil {
		t.Fatal("listCmd is nil")
	}

	if listCmd.Use != "list" {
		t.Errorf("listCmd.Use = %q, want 'list'", listCmd.Use)
	}

	if listCmd.Short != "List available servers and tools" {
		t.Errorf("listCmd.Short = %q, want 'List available servers and tools'", listCmd.Short)
	}
}

func TestConfigCmd(t *testing.T) {
	t.Parallel()

	if configCmd == nil {
		t.Fatal("configCmd is nil")
	}

	if configCmd.Use != "config" {
		t.Errorf("configCmd.Use = %q, want 'config'", configCmd.Use)
	}

	if configInitCmd == nil {
		t.Fatal("configInitCmd is nil")
	}

	if configValidateCmd == nil {
		t.Fatal("configValidateCmd is nil")
	}

	if configInitCmd.Use != "init" {
		t.Errorf("configInitCmd.Use = %q, want 'init'", configInitCmd.Use)
	}

	if configValidateCmd.Use != "validate" {
		t.Errorf("configValidateCmd.Use = %q, want 'validate'", configValidateCmd.Use)
	}
}

func TestCommandsRegistered(t *testing.T) {
	t.Parallel()

	commands := rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"serve", "list", "config", "version"}
	for _, name := range expectedCommands {
		if !commandNames[name] {
			t.Errorf("Command '%s' not registered", name)
		}
	}

	// Check config subcommands
	configSubcommands := configCmd.Commands()
	if len(configSubcommands) != 2 {
		t.Errorf("configCmd has %d subcommands, want 2", len(configSubcommands))
	}
}

func TestGlobalFlags(t *testing.T) {
	t.Parallel()

	if rootCmd.PersistentFlags() == nil {
		t.Fatal("rootCmd.PersistentFlags() is nil")
	}

	// Check that flags are defined
	flags := []string{"verbose", "quiet", "project", "config"}
	for _, flag := range flags {
		f := rootCmd.PersistentFlags().Lookup(flag)
		if f == nil {
			t.Errorf("Global flag '%s' not found", flag)
		}
	}
}

func TestCreateLogger(t *testing.T) {
	// Not parallel - all subtests modify global verbose/quiet flags
	t.Run("default logger", func(t *testing.T) {
		quiet = false
		verbose = false

		logger := createLogger()
		if logger == nil {
			t.Error("createLogger() returned nil")
		}
	})

	t.Run("verbose logger", func(t *testing.T) {
		quiet = false
		verbose = true

		logger := createLogger()
		if logger == nil {
			t.Error("createLogger() returned nil")
		}
	})

	t.Run("quiet logger", func(t *testing.T) {
		quiet = true
		verbose = false

		logger := createLogger()
		if logger == nil {
			t.Error("createLogger() returned nil")
		}
	})
}

func TestDetectProjectContext(t *testing.T) {
	t.Parallel()

	t.Run("with empty config", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewConfig()
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		cwd, _ := os.Getwd()
		ctx := detectProjectContext(cfg, cwd, logger)
		// Should return nil or empty context when no project matches
		if ctx != nil && ctx.Name != "" {
			t.Logf("detectProjectContext returned context: %+v", ctx)
		}
	})

	t.Run("with explicit project flag", func(t *testing.T) {
		t.Parallel()

		originalProjectFlag := projectFlag
		projectFlag = "explicit-project"
		defer func() { projectFlag = originalProjectFlag }()

		cfg := config.NewConfig()
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		cwd, _ := os.Getwd()
		ctx := detectProjectContext(cfg, cwd, logger)

		if ctx == nil {
			t.Fatal("detectProjectContext() returned nil context")
		}

		if ctx.Name != "explicit-project" {
			t.Errorf("detectProjectContext() Name = %q, want 'explicit-project'", ctx.Name)
		}

		if ctx.Source != project.SourceExplicit {
			t.Errorf("detectProjectContext() Source = %q, want 'explicit'", ctx.Source)
		}
	})
}

func TestRunConfigInit(t *testing.T) {
	t.Parallel()

	t.Run("creates config directory", func(t *testing.T) {
		t.Parallel()

		// Use the actual config init logic
		// Note: This will create config in the actual global directory
		// which is not ideal for tests, but we can at least verify it doesn't error
		// Skip in automated tests to avoid side effects
		t.Skip("Skipping config init test to avoid side effects on global config")
	})
}

func TestRunConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("validates empty config", func(t *testing.T) {
		t.Parallel()

		// Test LoadEffective with a non-existent path (will use defaults)
		tmpDir := t.TempDir()

		cfg, err := config.LoadEffective(tmpDir, "")
		if err != nil {
			t.Fatalf("LoadEffective() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("LoadEffective() returned nil")
		}
	})

	t.Run("validates config with servers", func(t *testing.T) {
		t.Parallel()

		// Create a temp directory structure with both config files
		tmpDir := t.TempDir()
		assernDir := filepath.Join(tmpDir, ".assern")
		if err := os.MkdirAll(assernDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create mcp.json with a test server
		mcpCfg := config.NewMCPConfig()
		mcpCfg.MCPServers["test"] = &config.MCPServer{
			Command: "echo",
			Args:    []string{"test"},
		}
		mcpPath := filepath.Join(assernDir, "mcp.json")
		if err := mcpCfg.Save(mcpPath); err != nil {
			t.Fatal(err)
		}

		// Create config.yaml with a test project
		testCfg := &config.Config{
			Servers: map[string]*config.ServerConfig{}, // Servers come from mcp.json
			Projects: map[string]*config.ProjectConfig{
				"myproject": {
					Directories: []string{"/path/to/project"},
				},
			},
			Settings: config.DefaultSettings(),
		}
		cfgPath := filepath.Join(assernDir, "config.yaml")
		if err := testCfg.Save(cfgPath); err != nil {
			t.Fatal(err)
		}

		// Load the config from the temp directory
		cfg, err := config.LoadWithMCP(mcpPath, cfgPath)
		if err != nil {
			t.Fatalf("LoadWithMCP() error = %v", err)
		}

		if len(cfg.Servers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(cfg.Servers))
		}

		if len(cfg.Projects) != 1 {
			t.Errorf("Expected 1 project, got %d", len(cfg.Projects))
		}
	})
}

func TestRunList(t *testing.T) {
	// Not parallel - modifies global homeDirFunc and reads verbose/quiet globals
	t.Run("with empty config", func(t *testing.T) {
		// Create a temp home directory with empty mcp.json
		tmpHome := t.TempDir()
		assernDir := filepath.Join(tmpHome, ".valksor", "assern")
		if err := os.MkdirAll(assernDir, 0o755); err != nil {
			t.Fatal(err)
		}
		// Create empty mcp.json (no servers)
		mcpPath := filepath.Join(assernDir, "mcp.json")
		if err := os.WriteFile(mcpPath, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
			t.Fatal(err)
		}

		// Override home directory
		restore := config.SetHomeDirForTesting(tmpHome)
		defer restore()

		// This will use an empty config and should error (no servers configured)
		err := runList(listCmd, nil)
		if err == nil {
			t.Error("runList() with empty config should return error, got nil")
		}
	})
}
