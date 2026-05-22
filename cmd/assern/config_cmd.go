package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/config"
)

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
