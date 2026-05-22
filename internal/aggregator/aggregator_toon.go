package aggregator

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/toon-format/toon-go"
)

// formatAsTOON converts a CallToolResult to TOON format.
func (a *Aggregator) formatAsTOON(result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	if result == nil {
		return &mcp.CallToolResult{}, nil
	}

	data := a.extractContentData(result)

	toonBytes, err := toon.Marshal(
		data,
		toon.WithLengthMarkers(true),
		toon.WithIndent(2),
	)
	if err != nil {
		return nil, fmt.Errorf("TOON marshal failed: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(toonBytes),
			},
		},
		IsError: result.IsError,
	}, nil
}

// extractContentData converts MCP content to a map structure for TOON encoding.
func (a *Aggregator) extractContentData(result *mcp.CallToolResult) map[string]any {
	data := make(map[string]any)

	if result.IsError {
		data["error"] = true
	}

	items := make([]map[string]any, 0, len(result.Content))
	for _, content := range result.Content {
		items = append(items, contentItemToMap(content))
	}

	data["content"] = items

	// Add metadata
	data["metadata"] = map[string]any{
		"format":       "toon",
		"contentCount": len(items),
	}

	return data
}

// contentItemToMap converts an MCP content item to a map for TOON encoding.
func contentItemToMap(content mcp.Content) map[string]any {
	item := make(map[string]any)

	switch c := content.(type) {
	case mcp.TextContent:
		item["type"] = "text"
		item["text"] = c.Text
	case mcp.ImageContent:
		item["type"] = "image"
		item["data"] = c.Data
		item["mimeType"] = c.MIMEType
	default:
		item["type"] = "unknown"
		item["data"] = fmt.Sprintf("%v", c)
	}

	return item
}
