package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/aggregator"
	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/env"
	"github.com/valksor/go-assern/internal/instance"
	"github.com/valksor/go-assern/internal/log"
	"github.com/valksor/go-assern/internal/project"
	"github.com/valksor/go-assern/internal/transport"
)

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
		logger.Info(
			"running in PROXY MODE - forwarding to existing instance",
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
	detector.SetConfigLoader(func(path string) (any, error) {
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
