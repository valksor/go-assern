// Package cli provides interactive CLI components for assern.
package cli

import (
	"fmt"
	"strings"
)

// FormatServerList formats a list of servers for display.
func FormatServerList(servers []ServerInfo, verbose bool) string {
	if len(servers) == 0 {
		return "No MCP servers configured."
	}

	var sb strings.Builder

	// Group by scope
	globalServers := make([]ServerInfo, 0)
	projectServers := make(map[string][]ServerInfo)

	for _, srv := range servers {
		if srv.Scope == ScopeGlobal {
			globalServers = append(globalServers, srv)
		} else {
			projectName := srv.Project
			if projectName == "" {
				projectName = "(unnamed)"
			}
			projectServers[projectName] = append(projectServers[projectName], srv)
		}
	}

	// Print global servers
	if len(globalServers) > 0 {
		globalPath, _ := getGlobalPath()
		fmt.Fprintf(&sb, "Global Servers (%s):\n", globalPath)
		for _, srv := range globalServers {
			formatServer(&sb, srv, verbose)
		}
		sb.WriteString("\n")
	}

	// Print project servers
	for projectName, srvs := range projectServers {
		fmt.Fprintf(&sb, "Project: %s", projectName)
		if len(srvs) > 0 && srvs[0].Project != "" && srvs[0].Project != projectName {
			fmt.Fprintf(&sb, " (%s)", srvs[0].Project)
		}
		fmt.Fprintf(&sb, "\n")
		for _, srv := range srvs {
			formatServer(&sb, srv, verbose)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatServerDetail formats a single server's detailed configuration.
func FormatServerDetail(srv *ServerInfo) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Name: %s\n", srv.Name)
	fmt.Fprintf(&sb, "Scope: %s", srv.Scope)
	if srv.Scope == ScopeProject && srv.Project != "" {
		fmt.Fprintf(&sb, " (%s)", srv.Project)
	}
	fmt.Fprintf(&sb, "\n")
	fmt.Fprintf(&sb, "Transport: %s\n", srv.Transport)

	// Transport-specific details
	switch srv.Transport {
	case "stdio":
		if srv.Server.Command != "" {
			fmt.Fprintf(&sb, "  Command: %s\n", srv.Server.Command)
		}
		if len(srv.Server.Args) > 0 {
			fmt.Fprintf(&sb, "  Args: %s\n", strings.Join(srv.Server.Args, " "))
		}
		if srv.Server.WorkDir != "" {
			fmt.Fprintf(&sb, "  Working Directory: %s\n", srv.Server.WorkDir)
		}
	case "http", "sse", "oauth-http", "oauth-sse":
		if srv.Server.URL != "" {
			fmt.Fprintf(&sb, "  URL: %s\n", srv.Server.URL)
		}
	}

	// OAuth details
	if srv.Server.OAuth != nil {
		fmt.Fprintf(&sb, "  OAuth:\n")
		fmt.Fprintf(&sb, "    Client ID: %s\n", srv.Server.OAuth.ClientID)
		if srv.Server.OAuth.ClientSecret != "" {
			fmt.Fprintf(&sb, "    Client Secret: ***\n")
		}
		if len(srv.Server.OAuth.Scopes) > 0 {
			fmt.Fprintf(&sb, "    Scopes: %s\n", strings.Join(srv.Server.OAuth.Scopes, ", "))
		}
		if srv.Server.OAuth.AuthServerMetadataURL != "" {
			fmt.Fprintf(&sb, "    Auth Server: %s\n", srv.Server.OAuth.AuthServerMetadataURL)
		}
		if srv.Server.OAuth.PKCEEnabled {
			fmt.Fprintf(&sb, "    PKCE: enabled\n")
		}
	}

	// Headers
	if len(srv.Server.Headers) > 0 {
		fmt.Fprintf(&sb, "  Headers:\n")
		for k, v := range srv.Server.Headers {
			fmt.Fprintf(&sb, "    %s: %s\n", k, v)
		}
	}

	// Environment variables
	if len(srv.Server.Env) > 0 {
		fmt.Fprintf(&sb, "  Environment:\n")
		for k, v := range srv.Server.Env {
			fmt.Fprintf(&sb, "    %s: %s\n", k, v)
		}
	}

	return sb.String()
}

// formatServer formats a single server for list display.
func formatServer(sb *strings.Builder, srv ServerInfo, verbose bool) {
	status := "enabled"
	_ = srv.Server.OAuth // Reserved for future use

	fmt.Fprintf(sb, "  %-20s %-10s %s", srv.Name, srv.Transport, status)

	if verbose {
		switch srv.Transport {
		case "stdio":
			fmt.Fprintf(sb, " (%s", srv.Server.Command)
			if len(srv.Server.Args) > 0 {
				fmt.Fprintf(sb, " %s", strings.Join(srv.Server.Args, " "))
			}
			fmt.Fprintf(sb, ")")
		case "http", "sse", "oauth-http", "oauth-sse":
			fmt.Fprintf(sb, " (%s)", srv.Server.URL)
		}
	}

	fmt.Fprintf(sb, "\n")
}

// getGlobalPath returns the global MCP config path for display.
func getGlobalPath() (string, error) {
	// Import config package to get the path
	return "~/.valksor/assern/mcp.json", nil
}

// FormatDiff formats a diff between old and new server configurations.
func FormatDiff(oldName, newName string, oldServer, newServer *ServerInfo) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Changes to server '%s':\n", oldName)

	if oldName != newName {
		fmt.Fprintf(&sb, "  Name: %s -> %s\n", oldName, newName)
	}

	if oldServer.Transport != newServer.Transport {
		fmt.Fprintf(&sb, "  Transport: %s -> %s\n", oldServer.Transport, newServer.Transport)
	}

	if oldServer.Server.Command != newServer.Server.Command {
		fmt.Fprintf(&sb, "  Command: %s -> %s\n", oldServer.Server.Command, newServer.Server.Command)
	}

	if oldServer.Server.URL != newServer.Server.URL {
		fmt.Fprintf(&sb, "  URL: %s -> %s\n", oldServer.Server.URL, newServer.Server.URL)
	}

	// Note: This is a simplified diff. A full implementation would compare all fields.

	return sb.String()
}
