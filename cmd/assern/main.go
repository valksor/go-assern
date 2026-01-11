// Package main is the entry point for the Assern CLI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/project"
	"github.com/valksor/go-assern/internal/transport"
	"github.com/valksor/go-assern/internal/version"
)

var (
	// Global flags.
	verbose      bool
	quiet        bool
	projectFlag  string
	configPath   string
	outputFormat string // "json" or "toon"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "assern",
	Short: "Assern - aggregate multiple MCP servers",
	Long: `Assern aggregates multiple MCP servers into a single unified interface
with project-level configuration support.

Running 'assern' without arguments starts the MCP server on stdio.

Configuration:
  Global: ~/.valksor/assern/mcp.json    (MCP server definitions)
  Global: ~/.valksor/assern/config.yaml (projects and settings)
  Local:  .assern/mcp.json              (project-specific servers)
  Local:  .assern/config.yaml           (project-specific config)`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to serve command
		return serveCmd.RunE(cmd, args)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP aggregator on stdio (default command)",
	Long: `Start the MCP aggregator server over stdio.

This is the default command - running 'assern' is equivalent to 'assern serve'.
The server aggregates all configured MCP servers and exposes their tools
with server-name prefixes (e.g., github_search, filesystem_read).`,
	RunE: runServe,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available servers and tools",
	RunE:  runList,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage mcp.json and config.yaml files",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create ~/.valksor/assern/ with mcp.json and config.yaml",
	Long: `Initialize the global configuration directory with default files.

Creates:
  ~/.valksor/assern/mcp.json    - MCP server definitions (add your servers here)
  ~/.valksor/assern/config.yaml - Projects and settings

Existing files are preserved.`,
	RunE: runConfigInit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	RunE:  runConfigValidate,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Info())
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress and info messages")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Explicit project name (overrides auto-detection)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config.yaml (default: ~/.valksor/assern/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output-format", "", "Output format for tool results: json or toon")

	// Add commands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	logger := createLogger()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Load effective configuration (merges global + local configs)
	cfg, err := config.LoadEffective(cwd, projectFlag)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Create context with timeout from config
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Settings.Timeout)
	defer cancel()

	// Create environment loader (only global .env)
	envLoader := project.NewEnvLoader()
	if err := envLoader.LoadGlobalEnv(); err != nil {
		logger.Debug("no global .env file", "error", err)
	}

	// Detect project for context (used for logging/display)
	projectCtx := detectProjectContext(cfg, cwd, logger)

	// Create aggregator
	agg, err := aggregator.New(aggregator.Options{
		Config:       cfg,
		Project:      projectCtx,
		EnvLoader:    envLoader,
		Logger:       logger,
		Timeout:      cfg.Settings.Timeout,
		OutputFormat: getOutputFormat(cfg, outputFormat),
	})
	if err != nil {
		return fmt.Errorf("creating aggregator: %w", err)
	}

	// Start serving
	return transport.ServeStdio(ctx, agg, logger)
}

func runList(cmd *cobra.Command, args []string) error {
	logger := createLogger()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Load effective configuration (merges global + local configs)
	cfg, err := config.LoadEffective(cwd, projectFlag)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Create context with timeout from config
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Settings.Timeout)
	defer cancel()

	// Create environment loader (only global .env)
	envLoader := project.NewEnvLoader()
	if err := envLoader.LoadGlobalEnv(); err != nil {
		logger.Debug("could not load global env", "error", err)
	}

	// Detect project for context
	projectCtx := detectProjectContext(cfg, cwd, logger)

	// Create aggregator
	agg, err := aggregator.New(aggregator.Options{
		Config:       cfg,
		Project:      projectCtx,
		EnvLoader:    envLoader,
		Logger:       logger,
		OutputFormat: getOutputFormat(cfg, outputFormat),
	})
	if err != nil {
		return fmt.Errorf("creating aggregator: %w", err)
	}

	// Start to discover tools
	if err := agg.Start(ctx); err != nil {
		return fmt.Errorf("starting aggregator: %w", err)
	}

	defer func() {
		if err := agg.Stop(); err != nil {
			logger.Warn("error stopping aggregator", "error", err)
		}
	}()

	// Print results
	projectName := "(none)"
	if projectCtx != nil && projectCtx.Name != "" {
		projectName = projectCtx.Name
		if projectCtx.Source == project.SourceAutoDetect {
			projectName = projectName + " (auto-detected)"
		}
	}
	fmt.Printf("Project: %s\n\n", projectName)

	fmt.Println("Servers:")

	for _, name := range agg.ServerNames() {
		fmt.Printf("  - %s\n", name)
	}

	fmt.Println()
	fmt.Println("Tools:")

	for _, tool := range agg.ListTools() {
		summary := tool.Summarize()
		fmt.Printf("  - %s (%s)\n", summary.PrefixedName, summary.Description)
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing configuration...")
	fmt.Println()

	// Ensure global directory exists
	dir, err := config.EnsureGlobalDir()
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	fmt.Printf("Global directory: %s\n", dir)

	// Create mcp.json if it doesn't exist
	mcpPath, err := config.GlobalMCPPath()
	if err != nil {
		return err
	}

	if !config.FileExists(mcpPath) {
		// Create default MCP config with example server
		defaultMCP := config.NewMCPConfig()
		defaultMCP.MCPServers["filesystem"] = &config.MCPServer{
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
		}

		if err := defaultMCP.Save(mcpPath); err != nil {
			return fmt.Errorf("saving mcp.json: %w", err)
		}

		fmt.Println("  [created] mcp.json     - MCP server definitions")
	} else {
		fmt.Println("  [exists]  mcp.json     - MCP server definitions")
	}

	// Create config.yaml if it doesn't exist
	cfgPath, err := config.GlobalConfigPath()
	if err != nil {
		return err
	}

	if !config.FileExists(cfgPath) {
		// Create default Assern config (projects and settings only)
		defaultCfg := &config.Config{
			Servers:  map[string]*config.ServerConfig{}, // Empty - servers come from mcp.json
			Projects: map[string]*config.ProjectConfig{},
			Settings: config.DefaultSettings(),
		}

		if err := defaultCfg.Save(cfgPath); err != nil {
			return fmt.Errorf("saving config.yaml: %w", err)
		}

		fmt.Println("  [created] config.yaml  - Projects and settings")
	} else {
		fmt.Println("  [exists]  config.yaml  - Projects and settings")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add MCP servers to mcp.json (can import from Claude Desktop)")
	fmt.Println("  2. Run 'assern config validate' to check configuration")
	fmt.Println("  3. Run 'assern list' to see available tools")

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Validate global MCP config
	mcpPath, err := config.GlobalMCPPath()
	if err != nil {
		return err
	}

	var mcpCfg *config.MCPConfig
	if config.FileExists(mcpPath) {
		mcpCfg, err = config.LoadMCPConfig(mcpPath)
		if err != nil {
			return fmt.Errorf("invalid global mcp.json at %s: %w", mcpPath, err)
		}

		fmt.Printf("[OK] %s (%d servers)\n", mcpPath, len(mcpCfg.MCPServers))
	} else {
		fmt.Printf("[--] %s (not found, optional)\n", mcpPath)
	}

	// Validate global config.yaml
	cfgPath, err := config.GlobalConfigPath()
	if err != nil {
		return err
	}

	var cfg *config.Config
	if config.FileExists(cfgPath) {
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("invalid global config.yaml at %s: %w", cfgPath, err)
		}

		fmt.Printf("[OK] %s (%d projects)\n", cfgPath, len(cfg.Projects))
	} else {
		fmt.Printf("[--] %s (not found, optional)\n", cfgPath)
	}

	// Summary
	serverCount := 0
	if mcpCfg != nil {
		serverCount = len(mcpCfg.MCPServers)
	}

	projectCount := 0
	if cfg != nil {
		projectCount = len(cfg.Projects)
	}

	fmt.Println()
	fmt.Println("Configuration valid!")
	fmt.Printf("  Servers:  %d\n", serverCount)
	fmt.Printf("  Projects: %d\n", projectCount)

	return nil
}

// detectProjectContext creates a project context for logging/display purposes.
// The actual config merging is done by LoadEffective.
func detectProjectContext(cfg *config.Config, cwd string, logger *slog.Logger) *project.Context {
	detector := project.NewDetector(cfg)

	ctx, err := detector.DetectWithExplicit(cwd, projectFlag)
	if err != nil {
		logger.Debug("project detection failed", "error", err)

		return nil
	}

	return ctx
}

func createLogger() *slog.Logger {
	level := slog.LevelInfo

	if verbose {
		level = slog.LevelDebug
	}

	if quiet {
		return transport.DiscardLogger()
	}

	return transport.StderrLogger(level)
}

// getOutputFormat determines the output format from flag, env var, and config.
// Priority: CLI flag > environment variable > config file > default.
func getOutputFormat(cfg *config.Config, flagValue string) string {
	// CLI flag takes highest precedence
	if flagValue != "" {
		return flagValue
	}

	// Environment variable ASSERN_OUTPUT_FORMAT
	if envValue := os.Getenv("ASSERN_OUTPUT_FORMAT"); envValue != "" {
		return envValue
	}

	// Config file setting
	if cfg.Settings != nil && cfg.Settings.OutputFormat != "" {
		return cfg.Settings.OutputFormat
	}

	return "json" // Default
}
