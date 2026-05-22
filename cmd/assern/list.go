package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/config"
	"github.com/valksor/go-assern/internal/instance"
	"github.com/valksor/go-assern/internal/log"
	"github.com/valksor/go-assern/internal/project"
)

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

			printTokenSummary(result.TokensByServer, result.TotalTokens, len(result.Tools))

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

	logger.Debug(
		"found running instance, querying tools",
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

	tools := agg.ListTools()
	byServer, totalTokens := agg.TokenStats()

	fmt.Println()
	fmt.Println("Tools:")

	for _, tool := range tools {
		summary := tool.Summarize()
		fmt.Printf("  - %s (%s)\n", summary.PrefixedName, summary.Description)
	}

	printTokenSummary(byServer, totalTokens, len(tools))

	return nil
}

// formatTokens renders an estimated token count compactly (e.g. "~3.4k").
func formatTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("~%.1fk", float64(n)/1000)
	}

	return fmt.Sprintf("~%d", n)
}

// printTokenSummary prints the total tool count and estimated token cost,
// plus a per-server breakdown when more than one server is present. Token
// figures are a relative heuristic, not an exact tokenizer count.
func printTokenSummary(byServer map[string]int, totalTokens, totalTools int) {
	fmt.Println()
	fmt.Printf("Total: %d tools, %s tokens (estimated, not exact)\n", totalTools, formatTokens(totalTokens))

	if len(byServer) <= 1 {
		return
	}

	names := make([]string, 0, len(byServer))
	for name := range byServer {
		names = append(names, name)
	}

	sort.Strings(names)

	fmt.Println("By server:")

	for _, name := range names {
		fmt.Printf("  - %-20s %s tokens\n", name, formatTokens(byServer[name]))
	}
}
