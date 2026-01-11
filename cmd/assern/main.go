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
	Short: "Valksor Assern - MCP Server Aggregator",
	Long: `Assern aggregates multiple MCP servers into a single unified interface
with project-level configuration support.

Use 'assern serve' to start the MCP server (default behavior).`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to serve command
		return serveCmd.RunE(cmd, args)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP aggregator server",
	Long:  `Start the MCP aggregator server over stdio. This is the default command.`,
	RunE:  runServe,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available servers and tools",
	RunE:  runList,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	Long:  `Create the global configuration directory and default config file.`,
	RunE:  runConfigInit,
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Explicit project name")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output-format", "", "Output format: json or toon (default: json)")

	// Add commands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
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
	ctx := context.Background()
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

	// Create environment loader (only global .env)
	envLoader := project.NewEnvLoader()
	_ = envLoader.LoadGlobalEnv()

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

	defer func() { _ = agg.Stop() }()

	// Print results
	projectName := ""
	if projectCtx != nil {
		projectName = projectCtx.Name
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
	// Ensure global directory exists
	dir, err := config.EnsureGlobalDir()
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	fmt.Printf("Config directory: %s\n", dir)

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
			return fmt.Errorf("saving mcp config: %w", err)
		}

		fmt.Printf("Created: %s\n", mcpPath)
	} else {
		fmt.Printf("Exists:  %s\n", mcpPath)
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
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Created: %s\n", cfgPath)
	} else {
		fmt.Printf("Exists:  %s\n", cfgPath)
	}

	fmt.Println("\nConfiguration files:")
	fmt.Println("  mcp.json    - MCP server definitions (copy-paste from Claude Desktop)")
	fmt.Println("  config.yaml - Projects, settings, and server overrides")

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
			return fmt.Errorf("invalid mcp.json: %w", err)
		}

		fmt.Printf("✓ %s (%d servers)\n", mcpPath, len(mcpCfg.MCPServers))
	} else {
		fmt.Printf("○ %s (not found)\n", mcpPath)
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
			return fmt.Errorf("invalid config.yaml: %w", err)
		}

		fmt.Printf("✓ %s (%d projects)\n", cfgPath, len(cfg.Projects))
	} else {
		fmt.Printf("○ %s (not found)\n", cfgPath)
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
	fmt.Printf("Configuration valid!\n")
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
