package main

import (
	"github.com/spf13/cobra"
)

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
