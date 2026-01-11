package aggregator

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestExtractContentData(t *testing.T) {
	t.Parallel()

	agg := &Aggregator{}

	t.Run("text content", func(t *testing.T) {
		t.Parallel()

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Hello, World!"},
			},
		}

		data := agg.extractContentData(result)

		content, ok := data["content"].([]map[string]any)
		if !ok {
			t.Fatal("content should be []map[string]any")
		}

		if len(content) != 1 {
			t.Errorf("expected 1 content item, got %d", len(content))
		}

		if content[0]["type"] != "text" {
			t.Errorf("expected type 'text', got %v", content[0]["type"])
		}

		if content[0]["text"] != "Hello, World!" {
			t.Errorf("expected text 'Hello, World!', got %v", content[0]["text"])
		}

		// Check metadata
		metadata, ok := data["metadata"].(map[string]any)
		if !ok {
			t.Fatal("metadata should be map[string]any")
		}

		if metadata["format"] != "toon" {
			t.Errorf("expected format 'toon', got %v", metadata["format"])
		}

		if metadata["contentCount"] != 1 {
			t.Errorf("expected contentCount 1, got %v", metadata["contentCount"])
		}
	})

	t.Run("image content", func(t *testing.T) {
		t.Parallel()

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.ImageContent{
					Type:     "image",
					Data:     "base64data==",
					MIMEType: "image/png",
				},
			},
		}

		data := agg.extractContentData(result)

		content, ok := data["content"].([]map[string]any)
		if !ok {
			t.Fatal("content should be []map[string]any")
		}

		if content[0]["type"] != "image" {
			t.Errorf("expected type 'image', got %v", content[0]["type"])
		}

		if content[0]["data"] != "base64data==" {
			t.Errorf("expected data 'base64data==', got %v", content[0]["data"])
		}

		if content[0]["mimeType"] != "image/png" {
			t.Errorf("expected mimeType 'image/png', got %v", content[0]["mimeType"])
		}
	})

	t.Run("error result", func(t *testing.T) {
		t.Parallel()

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Error occurred"},
			},
			IsError: true,
		}

		data := agg.extractContentData(result)

		if data["error"] != true {
			t.Error("expected error=true")
		}
	})

	t.Run("multiple content items", func(t *testing.T) {
		t.Parallel()

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "First"},
				mcp.TextContent{Type: "text", Text: "Second"},
				mcp.TextContent{Type: "text", Text: "Third"},
			},
		}

		data := agg.extractContentData(result)

		content, ok := data["content"].([]map[string]any)
		if !ok {
			t.Fatal("content should be []map[string]any")
		}

		if len(content) != 3 {
			t.Errorf("expected 3 content items, got %d", len(content))
		}

		metadata, ok := data["metadata"].(map[string]any)
		if !ok {
			t.Fatal("metadata should be map[string]any")
		}
		if metadata["contentCount"] != 3 {
			t.Errorf("expected contentCount 3, got %v", metadata["contentCount"])
		}
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()

		result := &mcp.CallToolResult{
			Content: []mcp.Content{},
		}

		data := agg.extractContentData(result)

		content, ok := data["content"].([]map[string]any)
		if !ok {
			t.Fatal("content should be []map[string]any")
		}

		if len(content) != 0 {
			t.Errorf("expected 0 content items, got %d", len(content))
		}
	})

	t.Run("mixed content types", func(t *testing.T) {
		t.Parallel()

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Text content"},
				mcp.ImageContent{Type: "image", Data: "imgdata", MIMEType: "image/jpeg"},
			},
		}

		data := agg.extractContentData(result)

		content, ok := data["content"].([]map[string]any)
		if !ok {
			t.Fatal("content should be []map[string]any")
		}
		if len(content) != 2 {
			t.Errorf("expected 2 content items, got %d", len(content))
		}

		if content[0]["type"] != "text" {
			t.Errorf("expected first item type 'text', got %v", content[0]["type"])
		}

		if content[1]["type"] != "image" {
			t.Errorf("expected second item type 'image', got %v", content[1]["type"])
		}
	})
}

func TestFormatAsTOON(t *testing.T) {
	t.Parallel()

	agg := &Aggregator{}

	t.Run("nil result", func(t *testing.T) {
		t.Parallel()

		result, err := agg.formatAsTOON(nil)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("text content produces valid TOON", func(t *testing.T) {
		t.Parallel()

		input := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Hello"},
			},
		}

		result, err := agg.formatAsTOON(input)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content item, got %d", len(result.Content))
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		// TOON output should contain the text
		if !strings.Contains(textContent.Text, "Hello") {
			t.Errorf("TOON output should contain 'Hello', got: %s", textContent.Text)
		}

		// TOON output should have format metadata
		if !strings.Contains(textContent.Text, "toon") {
			t.Errorf("TOON output should contain 'toon' format marker, got: %s", textContent.Text)
		}
	})

	t.Run("preserves IsError flag", func(t *testing.T) {
		t.Parallel()

		input := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Error message"},
			},
			IsError: true,
		}

		result, err := agg.formatAsTOON(input)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		if !result.IsError {
			t.Error("IsError flag should be preserved")
		}
	})

	t.Run("handles large payload", func(t *testing.T) {
		t.Parallel()

		// Create a large text content
		largeText := strings.Repeat("x", 100000)

		input := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: largeText},
			},
		}

		result, err := agg.formatAsTOON(input)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		if len(result.Content) != 1 {
			t.Errorf("expected 1 content item, got %d", len(result.Content))
		}
	})

	t.Run("handles special characters", func(t *testing.T) {
		t.Parallel()

		// Note: null character \x00 is not supported by TOON
		input := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Special: \n\t\r\"'\\"},
			},
		}

		result, err := agg.formatAsTOON(input)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		if len(result.Content) != 1 {
			t.Errorf("expected 1 content item, got %d", len(result.Content))
		}
	})

	t.Run("handles unicode", func(t *testing.T) {
		t.Parallel()

		input := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: "Unicode: 你好世界 🎉 αβγ"},
			},
		}

		result, err := agg.formatAsTOON(input)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatal("content should be TextContent")
		}

		// Should contain the unicode text
		if !strings.Contains(textContent.Text, "你好世界") {
			t.Errorf("should preserve unicode, got: %s", textContent.Text)
		}
	})

	t.Run("handles JSON-like content", func(t *testing.T) {
		t.Parallel()

		input := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: `{"key": "value", "array": [1, 2, 3]}`},
			},
		}

		result, err := agg.formatAsTOON(input)
		if err != nil {
			t.Fatalf("formatAsTOON() error = %v", err)
		}

		if len(result.Content) != 1 {
			t.Errorf("expected 1 content item, got %d", len(result.Content))
		}
	})
}

func TestFormatAsTOON_OutputFormat(t *testing.T) {
	t.Parallel()

	agg := &Aggregator{outputFormat: "toon"}

	input := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "Test content"},
		},
	}

	result, err := agg.formatAsTOON(input)
	if err != nil {
		t.Fatalf("formatAsTOON() error = %v", err)
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("content should be TextContent")
	}

	// TOON format should include length markers and structure
	// This is a basic check that TOON formatting is applied
	if textContent.Type != "text" {
		t.Errorf("expected type 'text', got %s", textContent.Type)
	}

	// Should have metadata section
	if !strings.Contains(textContent.Text, "metadata") {
		t.Errorf("TOON output should include metadata section")
	}

	// Should have content section
	if !strings.Contains(textContent.Text, "content") {
		t.Errorf("TOON output should include content section")
	}
}
