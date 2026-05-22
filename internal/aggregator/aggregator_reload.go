package aggregator

import (
	"context"
	"fmt"

	"github.com/valksor/go-assern/internal/config"
)

// ReloadResult contains information about a reload operation.
type ReloadResult struct {
	Added   int      `json:"added"`
	Removed int      `json:"removed"`
	Errors  []string `json:"errors,omitempty"`
}

// Reload reloads the configuration from disk and updates servers.
// Added servers are started, removed servers are stopped.
// Modified servers are restarted (stopped then started).
func (a *Aggregator) Reload(ctx context.Context) (*ReloadResult, error) {
	// Prevent concurrent reloads
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()

	a.logger.Info("reloading configuration")

	// Load fresh config from disk
	newCfg, err := config.LoadEffective(a.workDir, a.projectName)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Compare configs
	diff := config.DiffConfigs(a.cfg, newCfg)

	if !diff.HasChanges() {
		a.logger.Info("no configuration changes detected")

		return &ReloadResult{}, nil
	}

	a.logger.Info(
		"configuration changes detected",
		"added", len(diff.Added),
		"removed", len(diff.Removed),
		"modified", len(diff.Modified),
	)

	result := &ReloadResult{}

	// Stop removed servers
	for _, name := range diff.Removed {
		if err := a.stopServer(name); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("stop %s: %v", name, err))
			a.logger.Error("failed to stop server", "server", name, "error", err)
		} else {
			result.Removed++
			a.logger.Info("stopped server", "server", name)
		}
	}

	// Stop modified servers (they will be restarted)
	for _, name := range diff.Modified {
		if err := a.stopServer(name); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("stop %s: %v", name, err))
			a.logger.Error("failed to stop server for restart", "server", name, "error", err)
		}
	}

	// Get effective servers from new config
	effectiveServers := config.GetEffectiveServers(newCfg)

	// Start added servers
	for _, name := range diff.Added {
		srvCfg := effectiveServers[name]
		if err := a.startServer(ctx, name, srvCfg); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("start %s: %v", name, err))
			a.logger.Error("failed to start server", "server", name, "error", err)
		} else {
			result.Added++
			a.addServerToolsToMCPServer(name)
			a.logger.Info("started server", "server", name)
		}
	}

	// Restart modified servers
	for _, name := range diff.Modified {
		srvCfg := effectiveServers[name]
		if err := a.startServer(ctx, name, srvCfg); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("restart %s: %v", name, err))
			a.logger.Error("failed to restart server", "server", name, "error", err)
		} else {
			a.addServerToolsToMCPServer(name)
			a.logger.Info("restarted server", "server", name)
		}
	}

	// Update config reference. Guarded because discovery/code-mode handlers
	// read a.cfg concurrently on MCP-call goroutines.
	a.cfgMu.Lock()
	a.cfg = newCfg
	a.cfgMu.Unlock()

	a.logger.Info(
		"reload completed",
		"added", result.Added,
		"removed", result.Removed,
		"errors", len(result.Errors),
	)

	return result, nil
}

// stopServer stops a single server and removes it from registries.
func (a *Aggregator) stopServer(name string) error {
	a.mu.Lock()
	srv, exists := a.servers[name]
	if exists {
		delete(a.servers, name)
	}
	a.mu.Unlock()

	if !exists {
		return nil
	}

	// Remove from registries
	a.tools.RemoveServer(name)
	a.resources.RemoveServer(name)
	a.prompts.RemoveServer(name)
	a.health.Clear()

	// Stop the server
	return srv.Stop()
}

// addServerToolsToMCPServer adds a server's tools to the MCP server.
// This is called after a new server is started during reload. In discovery
// mode the tools stay in the catalog (loaded per session on demand), so only
// pinned tools are exposed globally.
func (a *Aggregator) addServerToolsToMCPServer(serverName string) {
	if a.mcpServer == nil {
		return
	}

	a.mu.RLock()
	entries := a.tools.GetByServer(serverName)
	a.mu.RUnlock()

	discovery := a.DiscoveryEnabled()

	var pinned map[string]struct{}
	if discovery {
		pinned = a.pinnedSet()
	}

	for _, entry := range entries {
		if discovery {
			if _, ok := pinned[entry.PrefixedName]; !ok {
				continue
			}
		}

		a.addToolToServer(entry)
	}
}
