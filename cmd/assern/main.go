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
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/aggregator"
	asserncli "github.com/valksor/go-assern/internal/cli"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/instance"
	"github.com/valksor/go-assern/internal/transport"
	"github.com/valksor/go-toolkit/cli"
	"github.com/valksor/go-toolkit/cli/disambiguate"
	"github.com/valksor/go-toolkit/env"
	"github.com/valksor/go-toolkit/log"
	"github.com/valksor/go-toolkit/project"
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

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload MCP server configuration",
	Long: `Trigger a configuration reload on the running assern instance.

This command connects to the running instance and instructs it to:
- Re-read mcp.json files (global and local)
- Stop servers that were removed from config
- Start servers that were added to config
- Restart servers whose configuration changed

In-flight requests to unchanged servers are not disrupted.

Alternatively, you can send SIGHUP to the assern process.`,
	RunE: runReload,
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

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP server configurations",
	Long: `Interactive commands for adding, editing, deleting, and listing MCP servers.

Supports both global (~/.valksor/assern/mcp.json) and project-specific
(.assern/mcp.json) configurations.

Commands can be invoked with colon notation (e.g., mcp:add) or space notation (e.g., mcp add).`,
}

var mcpAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new MCP server",
	Long: `Interactively add a new MCP server configuration.

Prompts for server name, transport type, and transport-specific settings.
Allows choosing between global and project-specific scope.`,
	RunE: runMCPAdd,
}

var mcpEditCmd = &cobra.Command{
	Use:   "edit [server-name]",
	Short: "Edit an existing MCP server",
	Long: `Interactively edit an existing MCP server configuration.

If server-name is provided as argument, pre-selects that server.
Otherwise, prompts to select from available servers.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMCPEdit,
}

var mcpDeleteCmd = &cobra.Command{
	Use:   "delete [server-name]",
	Short: "Delete MCP server(s)",
	Long: `Delete one or more MCP server configurations.

Prompts for server selection with multi-select support.
Can delete from both global and project-specific configs.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMCPDelete,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List MCP servers",
	Long: `List all configured MCP servers with their configurations.

Shows transport type, scope (global/project), and key settings.
More detailed than the 'assern list' command.`,
	RunE: runMCPList,
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
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(cli.NewVersionCommand("assern"))

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)

	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpEditCmd)
	mcpCmd.AddCommand(mcpDeleteCmd)
	mcpCmd.AddCommand(mcpListCmd)

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
		WorkDir:      cwd,
		ProjectName:  projectFlag,
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
		sockServer := instance.NewServer(socketPath, mcpServer, agg, logger)
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

func runReload(cmd *cobra.Command, args []string) error {
	configureLogger()
	logger := log.Logger()

	// Check for running instance
	detector := instance.NewDetector(logger)
	existing, err := detector.DetectRunning()
	if err != nil {
		return fmt.Errorf("detecting instance: %w", err)
	}

	if existing == nil {
		return errors.New("no running assern instance found")
	}

	// Send reload command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := instance.Reload(ctx, existing.SocketPath)
	if err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}

	// Print results
	fmt.Printf("Configuration reloaded successfully\n")
	fmt.Printf("  Added:   %d servers\n", result.Added)
	fmt.Printf("  Removed: %d servers\n", result.Removed)

	if len(result.Errors) > 0 {
		fmt.Printf("  Errors:  %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("    - %s\n", e)
		}
	}

	return nil
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

// runMCPAdd adds a new MCP server interactively.
func runMCPAdd(cmd *cobra.Command, args []string) error {
	fmt.Println("Adding a new MCP server...")
	fmt.Println()

	// Create manager
	mgr, err := asserncli.NewMCPManager()
	if err != nil {
		return fmt.Errorf("creating MCP manager: %w", err)
	}

	// Run interactive prompts
	input, err := asserncli.PromptForMCPServer(nil)
	if err != nil {
		if err.Error() == "cancelled by user" {
			fmt.Println("Cancelled.")

			return nil
		}

		return err
	}

	// Add server
	if err := mgr.AddServer(input); err != nil {
		return fmt.Errorf("adding server: %w", err)
	}

	fmt.Printf("\nServer '%s' added successfully!\n", input.Name)

	return nil
}

// runMCPEdit edits an existing MCP server.
func runMCPEdit(cmd *cobra.Command, args []string) error {
	fmt.Println("Editing an MCP server...")
	fmt.Println()

	// Create manager
	mgr, err := asserncli.NewMCPManager()
	if err != nil {
		return fmt.Errorf("creating MCP manager: %w", err)
	}

	// Determine server name
	var serverName string
	if len(args) > 0 {
		serverName = args[0]
	} else {
		// List all servers
		allServers := mgr.ListServers()
		if len(allServers) == 0 {
			fmt.Println("No MCP servers configured.")

			return nil
		}

		// Get server names
		globalNames, localNames := mgr.ServerNames()
		allNames := append(globalNames, localNames...)

		// Select server
		selected, err := asserncli.SelectServer(allNames, "Select server to edit:")
		if err != nil {
			return err
		}
		serverName = selected
	}

	// Get existing server
	existingServer, scope, err := mgr.GetServer(serverName)
	if err != nil {
		return fmt.Errorf("getting server: %w", err)
	}

	// Convert to input format
	input := &asserncli.MCPInput{
		Name:      serverName,
		Scope:     scope,
		Transport: existingServer.Transport,
		Command:   existingServer.Command,
		Args:      existingServer.Args,
		Env:       existingServer.Env,
		WorkDir:   existingServer.WorkDir,
		URL:       existingServer.URL,
		Headers:   existingServer.Headers,
		OAuth:     existingServer.OAuth,
	}

	// Run interactive prompts
	updatedInput, err := asserncli.PromptForMCPServer(input)
	if err != nil {
		if err.Error() == "cancelled by user" {
			fmt.Println("Cancelled.")

			return nil
		}

		return err
	}

	// Update server
	if err := mgr.UpdateServer(serverName, updatedInput); err != nil {
		return fmt.Errorf("updating server: %w", err)
	}

	fmt.Printf("\nServer '%s' updated successfully!\n", serverName)

	return nil
}

// runMCPDelete deletes MCP server(s).
func runMCPDelete(cmd *cobra.Command, args []string) error {
	fmt.Println("Deleting MCP server(s)...")
	fmt.Println()

	// Create manager
	mgr, err := asserncli.NewMCPManager()
	if err != nil {
		return fmt.Errorf("creating MCP manager: %w", err)
	}

	// Get server names
	var toDelete []string
	if len(args) > 0 {
		toDelete = args
	} else {
		// List all servers
		allServers := mgr.ListServers()
		if len(allServers) == 0 {
			fmt.Println("No MCP servers configured.")

			return nil
		}

		// Get server names
		globalNames, localNames := mgr.ServerNames()
		allNames := append(globalNames, localNames...)

		// Select servers
		selected, err := asserncli.SelectServers(allNames, "Select server(s) to delete:")
		if err != nil {
			return err
		}
		toDelete = selected
	}

	// Confirm deletion
	if err := asserncli.ConfirmDelete(toDelete); err != nil {
		if err.Error() == "cancelled by user" {
			fmt.Println("Cancelled.")

			return nil
		}

		return err
	}

	// Delete servers
	if err := mgr.DeleteServer(toDelete); err != nil {
		return fmt.Errorf("deleting servers: %w", err)
	}

	fmt.Printf("\nDeleted %d server(s)\n", len(toDelete))

	return nil
}

// runMCPList lists all MCP servers.
func runMCPList(cmd *cobra.Command, args []string) error {
	// Create manager
	mgr, err := asserncli.NewMCPManager()
	if err != nil {
		return fmt.Errorf("creating MCP manager: %w", err)
	}

	// Get all servers
	servers := mgr.ListServers()

	// Format and display
	output := asserncli.FormatServerList(servers, verbose)
	fmt.Println(output)

	return nil
}
