// Package main is the entry point for the Assern CLI.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/valksor/go-assern/internal/cobracli"
	"github.com/valksor/go-assern/internal/disambiguate"
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
	rootCmd.AddCommand(cobracli.NewVersionCommand("assern"))

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
