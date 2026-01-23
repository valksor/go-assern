// Package transport provides MCP transport implementations.
package transport

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/valksor/go-assern/internal/aggregator"
)

// ServeStdio starts the aggregator as an MCP server over stdio.
func ServeStdio(ctx context.Context, agg *aggregator.Aggregator, logger *slog.Logger) error {
	// Start the aggregator (connect to all backend servers)
	if err := agg.Start(ctx); err != nil {
		return fmt.Errorf("starting aggregator: %w", err)
	}

	// Create the MCP server
	mcpServer := agg.CreateMCPServer()

	return ServeStdioWithServer(ctx, agg, mcpServer, logger)
}

// ServeStdioWithServer serves an existing MCP server over stdio.
// This allows the MCP server to be shared with other transports (e.g., socket).
func ServeStdioWithServer(ctx context.Context, agg *aggregator.Aggregator, mcpServer *server.MCPServer, logger *slog.Logger) error {
	// Setup signal handlers
	shutdownCh := make(chan os.Signal, 1)
	reloadCh := make(chan os.Signal, 1)

	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)
	signal.Notify(reloadCh, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-shutdownCh:
				logger.Info("received shutdown signal")

				if err := agg.Stop(); err != nil {
					logger.Error("error stopping aggregator", "error", err)
				}

				os.Exit(0)
			case <-reloadCh:
				logger.Info("received SIGHUP, reloading configuration")

				result, err := agg.Reload(ctx)
				if err != nil {
					logger.Error("configuration reload failed", "error", err)
				} else {
					logger.Info("configuration reload completed",
						"added", result.Added,
						"removed", result.Removed,
						"errors", len(result.Errors),
					)
				}
			}
		}
	}()

	// Redirect any log output to stderr to keep stdout clean for MCP protocol
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	logger.Info("starting MCP server on stdio",
		"project", agg.ProjectName(),
		"servers", len(agg.ServerNames()),
		"tools", len(agg.ListTools()),
	)

	// Start serving
	if err := server.ServeStdio(mcpServer); err != nil {
		return fmt.Errorf("serving stdio: %w", err)
	}

	return nil
}
