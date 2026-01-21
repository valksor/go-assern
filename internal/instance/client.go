package instance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// ClientTimeout is the default timeout for client operations.
const ClientTimeout = 10 * time.Second

// ToolInfo represents tool information returned from a query.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListResult contains the result of querying a running instance.
type ListResult struct {
	Tools []ToolInfo
}

// Client connects to a running assern instance to query information.
type Client struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
	requestID  int
}

// NewClient creates a new client for the given socket path.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		requestID:  0,
	}
}

// Connect establishes connection to the instance.
func (c *Client) Connect(ctx context.Context) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("connect to socket: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// Initialize performs the MCP initialize handshake.
func (c *Client) Initialize(ctx context.Context) error {
	// Wait for handshake timeout to pass (server expects internal commands first)
	time.Sleep(handshakeTimeout + 10*time.Millisecond)

	c.requestID++
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      c.requestID,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "assern-client",
				"version": "1.0.0",
			},
		},
	}

	if err := c.sendRequest(initReq); err != nil {
		return fmt.Errorf("send initialize: %w", err)
	}

	// Read initialize response
	var initResp struct {
		ID     int `json:"id"`
		Result any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := c.readResponse(&initResp); err != nil {
		return fmt.Errorf("read initialize response: %w", err)
	}

	if initResp.Error != nil {
		return fmt.Errorf("initialize error: %s", initResp.Error.Message)
	}

	// Send initialized notification
	initializedNotif := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}

	if err := c.sendRequest(initializedNotif); err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}

	return nil
}

// ListTools queries the available tools from the running instance.
func (c *Client) ListTools(ctx context.Context) (*ListResult, error) {
	c.requestID++
	listReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      c.requestID,
		"method":  "tools/list",
		"params":  map[string]any{},
	}

	if err := c.sendRequest(listReq); err != nil {
		return nil, fmt.Errorf("send tools/list: %w", err)
	}

	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Tools []ToolInfo `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := c.readResponse(&resp); err != nil {
		return nil, fmt.Errorf("read tools/list response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	return &ListResult{
		Tools: resp.Result.Tools,
	}, nil
}

func (c *Client) sendRequest(req any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	if _, err := c.conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (c *Client) readResponse(resp any) error {
	if err := c.conn.SetReadDeadline(time.Now().Add(ClientTimeout)); err != nil {
		return err
	}
	defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }()

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return err
	}

	return json.Unmarshal(line, resp)
}

// QueryTools connects to a running instance and returns the available tools.
// This is a convenience function that handles the full connection lifecycle.
func QueryTools(ctx context.Context, socketPath string) (*ListResult, error) {
	client := NewClient(socketPath)

	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	if err := client.Initialize(ctx); err != nil {
		return nil, err
	}

	return client.ListTools(ctx)
}
