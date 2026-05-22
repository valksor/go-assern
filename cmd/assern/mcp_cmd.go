package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/valksor/go-assern/internal/cli"
)

// runMCPAdd adds a new MCP server interactively.
func runMCPAdd(cmd *cobra.Command, args []string) error {
	fmt.Println("Adding a new MCP server...")
	fmt.Println()

	// Create manager
	mgr, err := cli.NewMCPManager()
	if err != nil {
		return fmt.Errorf("creating MCP manager: %w", err)
	}

	// Run interactive prompts
	input, err := cli.PromptForMCPServer(nil)
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
	mgr, err := cli.NewMCPManager()
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
		selected, err := cli.SelectServer(allNames, "Select server to edit:")
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
	input := &cli.MCPInput{
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
	updatedInput, err := cli.PromptForMCPServer(input)
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
	mgr, err := cli.NewMCPManager()
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
		selected, err := cli.SelectServers(allNames, "Select server(s) to delete:")
		if err != nil {
			return err
		}
		toDelete = selected
	}

	// Confirm deletion
	if err := cli.ConfirmDelete(toDelete); err != nil {
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
	mgr, err := cli.NewMCPManager()
	if err != nil {
		return fmt.Errorf("creating MCP manager: %w", err)
	}

	// Get all servers
	servers := mgr.ListServers()

	// Format and display
	output := cli.FormatServerList(servers, verbose)
	fmt.Println(output)

	return nil
}
