// Package main is the entry point for the Assern CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/instance"
	"github.com/valksor/go-assern/internal/transport"
	"github.com/valksor/go-toolkit/cli/disambiguate"
	"github.com/valksor/go-toolkit/env"
	"github.com/valksor/go-toolkit/log"
	"github.com/valksor/go-toolkit/project"
	"github.com/valksor/go-toolkit/version"
)

var (
	// Global flags.
	verbose      bool
	quiet        bool
	projectFlag  string
	configPath   string
	outputFormat string // "json" or "toon"

	// config init flags.
	forceInit bool

	// list flags.
	freshList bool
)

// contextKey is the type used for context keys to prevent collisions.
type contextKey string

// cancelKey is the context key for storing the cancel function.
const cancelKey contextKey = "cancel"

// Execute runs the root command with colon notation support.
func Execute() error {
	// Pre-process args to handle colon notation before Cobra sees them
	args := os.Args[1:]
	if len(args) > 0 && strings.Contains(args[0], ":") {
		resolved, matches, err := disambiguate.ResolveColonPath(rootCmd, args[0])
		if err == nil {
			// Unambiguous match - use resolved path
			if len(matches) == 0 {
				rootCmd.SetArgs(append(resolved, args[1:]...))

				return rootCmd.Execute()
			}
			// Ambiguous - try interactive selection
			if !disambiguate.IsInteractive() {
				return errors.New(disambiguate.FormatAmbiguousError(args[0], matches))
			}
			selected, err := disambiguate.SelectCommand(matches, args[0])
			if err != nil {
				return err
			}
			rootCmd.SetArgs(append(selected.Path, args[1:]...))

			return rootCmd.Execute()
		}
		// If error is "not a colon path", fall through to normal execution
		if !strings.Contains(err.Error(), "not a colon path") {
			return err
		}
	}

	return rootCmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
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

Existing files are preserved unless --force is used.`,
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
		fmt.Println(version.Info("assern"))
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

	// config init flags
	configInitCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing configuration files")

	// list flags
	listCmd.Flags().BoolVarP(&freshList, "fresh", "f", false, "Force fresh discovery (ignore running instance)")
}

// setupAggregator initializes and configures the aggregator with common setup.
// Returns the aggregator, context, logger, and any error encountered.
func setupAggregator() (*aggregator.Aggregator, context.Context, *slog.Logger, error) {
	configureLogger()
	logger := log.Logger()

	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getting working directory: %w", err)
	}

	// Load effective configuration (merges global + local configs)
	cfg, err := config.LoadEffective(cwd, projectFlag)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Settings.Timeout)

	// Note: The caller is responsible for calling cancel() when done
	// We attach it to the context so callers can access it if needed
	ctx = context.WithValue(ctx, cancelKey, cancel)

	envLoader := loadGlobalEnv(logger)

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
		cancel()

		return nil, nil, nil, fmt.Errorf("creating aggregator: %w", err)
	}

	return agg, ctx, logger, nil
}

func runServe(cmd *cobra.Command, args []string) error {
	configureLogger()
	logger := log.Logger()

	// Check for existing instance
	detector := instance.NewDetector(logger)
	existing, err := detector.DetectRunning()
	if err != nil {
		logger.Debug("instance detection failed", "error", err)
		// Continue as primary - detection failure shouldn't block
	}

	if existing != nil {
		// Run as proxy to existing instance
		logger.Info("running in PROXY MODE - forwarding to existing instance",
			"primary_pid", existing.PID,
			"socket", existing.SocketPath,
		)

		return runAsProxy(existing.SocketPath, logger)
	}

	// Run as primary instance
	return runAsPrimary(cmd, args, logger)
}

func runAsPrimary(_ *cobra.Command, _ []string, _ *slog.Logger) error {
	agg, ctx, logger, err := setupAggregator()
	if err != nil {
		return err
	}
	defer func() {
		if cancel, ok := ctx.Value(cancelKey).(context.CancelFunc); ok {
			cancel()
		}
	}()

	// Start the aggregator
	if err := agg.Start(ctx); err != nil {
		return fmt.Errorf("starting aggregator: %w", err)
	}

	// Create MCP server
	mcpServer := agg.CreateMCPServer()

	// Start socket server for instance sharing
	socketPath, err := config.SocketPath()
	if err != nil {
		logger.Warn("failed to get socket path", "error", err)
	} else {
		sockServer := instance.NewServer(socketPath, mcpServer, logger)
		if err := sockServer.Start(); err != nil {
			logger.Warn("failed to start socket server", "error", err)
			// Continue without socket - stdio still works
		} else {
			defer func() { _ = sockServer.Stop() }()
		}
	}

	// Serve stdio (existing transport code)
	return transport.ServeStdioWithServer(ctx, agg, mcpServer, logger)
}

func runAsProxy(socketPath string, logger *slog.Logger) error {
	proxy := instance.NewProxy(socketPath, logger)
	defer func() { _ = proxy.Close() }()

	return proxy.ServeStdio(context.Background())
}

func runList(cmd *cobra.Command, args []string) error {
	configureLogger()
	logger := log.Logger()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Load effective configuration (merges global + local configs)
	cfg, err := config.LoadEffective(cwd, projectFlag)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check if any servers are configured
	effectiveServers := config.GetEffectiveServers(cfg)
	if len(effectiveServers) == 0 {
		fmt.Println("No MCP servers configured.")
		fmt.Println()
		fmt.Println("Add servers to:")
		fmt.Println("  Global: ~/.valksor/assern/mcp.json")
		fmt.Println("  Local:  .assern/mcp.json (project-specific)")
		fmt.Println()
		fmt.Println("Run 'assern config init' to create default config")

		return nil
	}

	// Try to query from running instance (unless --fresh flag is set)
	if !freshList {
		if result := tryListFromInstance(logger); result != nil {
			// Print results from running instance
			projectName := "(none)"
			if projectCtx := detectProjectContext(cfg, cwd, logger); projectCtx != nil && projectCtx.Name != "" {
				projectName = projectCtx.Name
				if projectCtx.Source == project.SourceAutoDetect {
					projectName = projectName + " (auto-detected)"
				}
			}
			fmt.Printf("Project: %s\n", projectName)
			fmt.Println("(from running instance)")
			fmt.Println()

			fmt.Println("Tools:")

			for _, tool := range result.Tools {
				fmt.Printf("  - %s (%s)\n", tool.Name, tool.Description)
			}

			return nil
		}
	}

	// Fall back to fresh discovery
	return runListFresh(cfg, cwd, logger)
}

// tryListFromInstance attempts to query tools from a running instance.
// Returns nil if no instance is running or query fails.
func tryListFromInstance(logger *slog.Logger) *instance.ListResult {
	detector := instance.NewDetector(logger)
	existing, err := detector.DetectRunning()
	if err != nil {
		logger.Debug("instance detection failed", "error", err)

		return nil
	}

	if existing == nil {
		logger.Debug("no running instance found")

		return nil
	}

	logger.Debug("found running instance, querying tools",
		"pid", existing.PID,
		"socket", existing.SocketPath,
	)

	ctx, cancel := context.WithTimeout(context.Background(), instance.ClientTimeout)
	defer cancel()

	result, err := instance.QueryTools(ctx, existing.SocketPath)
	if err != nil {
		logger.Debug("failed to query tools from instance", "error", err)

		return nil
	}

	return result
}

func runListFresh(cfg *config.Config, cwd string, logger *slog.Logger) error {
	// Use helper to create aggregator
	agg, ctx, logger, err := setupAggregator()
	if err != nil {
		return err
	}
	defer func() {
		if cancel, ok := ctx.Value(cancelKey).(context.CancelFunc); ok {
			cancel()
		}
	}()

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
	if projectCtx := detectProjectContext(cfg, cwd, logger); projectCtx != nil && projectCtx.Name != "" {
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
	fmt.Println()

	var mcpCreated, cfgCreated bool

	// Handle mcp.json
	mcpPath, err := config.GlobalMCPPath()
	if err != nil {
		return err
	}

	mcpExists := config.FileExists(mcpPath)

	if forceInit || !mcpExists {
		// Create empty MCP config
		defaultMCP := config.NewMCPConfig()

		if err := defaultMCP.Save(mcpPath); err != nil {
			return fmt.Errorf("saving mcp.json: %w", err)
		}

		mcpCreated = true

		if forceInit && mcpExists {
			fmt.Printf("  [overwrite] %s\n", mcpPath)
		} else {
			fmt.Printf("  [created]   %s\n", mcpPath)
		}
	} else {
		fmt.Printf("  [exists]    %s\n", mcpPath)
	}

	// Handle config.yaml
	cfgPath, err := config.GlobalConfigPath()
	if err != nil {
		return err
	}

	cfgExists := config.FileExists(cfgPath)

	if forceInit || !cfgExists {
		// Create default Assern config (projects and settings only)
		defaultCfg := &config.Config{
			Servers:  map[string]*config.ServerConfig{}, // Empty - servers come from mcp.json
			Projects: map[string]*config.ProjectConfig{},
			Settings: config.DefaultSettings(),
		}

		if err := defaultCfg.Save(cfgPath); err != nil {
			return fmt.Errorf("saving config.yaml: %w", err)
		}

		cfgCreated = true

		if forceInit && cfgExists {
			fmt.Printf("  [overwrite] %s\n", cfgPath)
		} else {
			fmt.Printf("  [created]   %s\n", cfgPath)
		}
	} else {
		fmt.Printf("  [exists]    %s\n", cfgPath)
	}

	fmt.Println()

	// Summary message based on what happened
	if mcpCreated || cfgCreated {
		if forceInit && (mcpExists || cfgExists) {
			fmt.Println("Configuration reinitialized!")
		} else {
			fmt.Println("Configuration initialized!")
		}

		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Add MCP servers to mcp.json (can import from Claude Desktop)")
		fmt.Println("  2. Run 'assern config validate' to check configuration")
		fmt.Println("  3. Run 'assern list' to see available tools")
	} else {
		fmt.Println("Configuration already initialized. Use --force to reinitialize.")
	}

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

// configPathResolver adapts go-assern config functions to project.PathResolver interface.
type configPathResolver struct{}

func (r *configPathResolver) FindLocalConfigDir(startDir string) string {
	return config.FindLocalConfigDir(startDir)
}

func (r *configPathResolver) LocalConfigPath(localDir string) string {
	return config.LocalConfigPath(localDir)
}

func (r *configPathResolver) FileExists(path string) bool {
	return config.FileExists(path)
}

// detectProjectContext creates a project context for logging/display purposes.
// The actual config merging is done by LoadEffective.
func detectProjectContext(cfg *config.Config, cwd string, logger *slog.Logger) *project.Context {
	// Create path resolver
	resolver := &configPathResolver{}

	// Create registry from config projects
	registry := project.NewRegistry()
	for name, proj := range cfg.Projects {
		registry.Register(name, proj.Directories, nil)
	}

	// Create detector
	detector := project.NewDetector(resolver, ".assern", registry)

	// Set config loader for LocalProjectConfig
	detector.SetConfigLoader(func(path string) (interface{}, error) {
		return config.LoadLocalProject(path)
	})

	ctx, err := detector.DetectWithExplicit(cwd, projectFlag)
	if err != nil {
		logger.Debug("project detection failed", "error", err)

		return nil
	}

	return ctx
}

func configureLogger() {
	output := io.Discard
	if !quiet {
		output = os.Stderr
	}
	log.Configure(log.Options{
		Output:  output,
		Verbose: verbose,
	})
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

func loadGlobalEnv(logger *slog.Logger) *env.Loader {
	envLoader := env.NewLoader()
	globalEnvPath, err := config.GlobalEnvPath()
	if err != nil {
		logger.Debug("could not get global env path", "error", err)
	} else if err := envLoader.LoadDotenv(globalEnvPath); err != nil {
		logger.Debug("no global .env file", "error", err)
	}

	return envLoader
}
