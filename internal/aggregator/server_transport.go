package aggregator

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"

	"github.com/valksor/go-assern/internal/config"
)

// sharedHTTPTransport is a connection-pooled HTTP transport for all HTTP-based MCP servers.
// This enables connection reuse across multiple requests to the same backend servers.
var sharedHTTPTransport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 10,
	MaxConnsPerHost:     20,
	IdleConnTimeout:     90 * time.Second,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout: 10 * time.Second,
}

// sharedHTTPClient is the default HTTP client with connection pooling.
var sharedHTTPClient = &http.Client{
	Transport: sharedHTTPTransport,
	Timeout:   60 * time.Second,
}

// detectTransport determines the transport type from config.
func detectTransport(cfg *config.ServerConfig) TransportType {
	// Explicit transport takes precedence
	if cfg.Transport != "" {
		return TransportType(cfg.Transport)
	}

	// Auto-detect OAuth transports when OAuth config is present
	if cfg.OAuth != nil && cfg.URL != "" {
		return TransportOAuthHTTP // Default OAuth to HTTP (modern MCP standard)
	}

	// Auto-detect based on which fields are set
	if cfg.URL != "" {
		return TransportHTTP // Default URL-based to Streamable HTTP (modern MCP standard)
	}

	if cfg.Command != "" {
		return TransportStdio
	}

	return ""
}

// createStdioClient creates a stdio transport client.
func (s *ManagedServer) createStdioClient() (*client.Client, error) {
	// Ensure PATH is preserved even if mcp-go changes behavior
	env := s.env
	if !envContains(env, "PATH") {
		pathEnv := os.Getenv("PATH")
		if pathEnv != "" {
			env = append([]string{"PATH=" + pathEnv}, env...)
			s.logger.Debug("added PATH to server environment", "server", s.name)
		}
	}

	return client.NewStdioMCPClient(s.cfg.Command, env, s.cfg.Args...)
}

// envContains checks if a specific environment variable exists in the env slice.
func envContains(env []string, key string) bool {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}

	return false
}

// createSSEClient creates an SSE transport client with optional headers.
func (s *ManagedServer) createSSEClient() (*client.Client, error) {
	opts := []transport.ClientOption{
		transport.WithHTTPClient(sharedHTTPClient), // Use connection-pooled client
	}

	// Add custom headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(s.cfg.Headers))
	}

	return client.NewSSEMCPClient(s.cfg.URL, opts...)
}

// createHTTPClient creates a Streamable HTTP transport client with optional headers.
func (s *ManagedServer) createHTTPClient() (*client.Client, error) {
	opts := []transport.StreamableHTTPCOption{
		transport.WithHTTPBasicClient(sharedHTTPClient), // Use connection-pooled client
	}

	// Add custom headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(s.cfg.Headers))
	}

	return client.NewStreamableHttpClient(s.cfg.URL, opts...)
}

// buildOAuthConfig constructs the mcp-go OAuth config from the server's
// settings and attaches a file-backed token cache so tokens persist across
// runs (and are shared by servers referencing the same auth profile).
func (s *ManagedServer) buildOAuthConfig() transport.OAuthConfig {
	oauthCfg := transport.OAuthConfig{
		ClientID:              s.cfg.OAuth.ClientID,
		ClientSecret:          s.cfg.OAuth.ClientSecret,
		RedirectURI:           s.cfg.OAuth.RedirectURI,
		Scopes:                s.cfg.OAuth.Scopes,
		AuthServerMetadataURL: s.cfg.OAuth.AuthServerMetadataURL,
		PKCEEnabled:           s.cfg.OAuth.PKCEEnabled,
	}

	if dir, err := config.TokensDir(); err != nil {
		s.logger.Warn("oauth token cache unavailable; tokens will not persist", "error", err)
	} else {
		oauthCfg.TokenStore = newFileTokenStore(dir, s.tokenCacheKey())
	}

	return oauthCfg
}

// tokenCacheKey identifies the cached-token bucket: the shared OAuth profile
// reference when set, otherwise the server's own name.
func (s *ManagedServer) tokenCacheKey() string {
	if s.cfg.OAuthRef != "" {
		return s.cfg.OAuthRef
	}

	return s.name
}

// missingOAuthErr returns a descriptive error when OAuth config is absent,
// distinguishing an unresolved profile reference from a missing inline config.
func (s *ManagedServer) missingOAuthErr(transportName string) error {
	if s.cfg.OAuthRef != "" {
		return fmt.Errorf("%s transport: oauth_ref %q not found in auth profiles: %w", transportName, s.cfg.OAuthRef, ErrOAuthRequired)
	}

	return fmt.Errorf("%s transport: %w", transportName, ErrOAuthRequired)
}

// createOAuthSSEClient creates an SSE client with OAuth authentication.
func (s *ManagedServer) createOAuthSSEClient() (*client.Client, error) {
	if s.cfg.OAuth == nil {
		return nil, s.missingOAuthErr("oauth-sse")
	}

	oauthCfg := s.buildOAuthConfig()

	opts := []transport.ClientOption{}

	// Add additional headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(s.cfg.Headers))
	}

	return client.NewOAuthSSEClient(s.cfg.URL, oauthCfg, opts...)
}

// createOAuthHTTPClient creates a Streamable HTTP client with OAuth authentication.
func (s *ManagedServer) createOAuthHTTPClient() (*client.Client, error) {
	if s.cfg.OAuth == nil {
		return nil, s.missingOAuthErr("oauth-http")
	}

	oauthCfg := s.buildOAuthConfig()

	opts := []transport.StreamableHTTPCOption{}

	// Add additional headers if configured
	if len(s.cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(s.cfg.Headers))
	}

	return client.NewOAuthStreamableHttpClient(s.cfg.URL, oauthCfg, opts...)
}
