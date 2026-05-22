package aggregator

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// charsPerToken is the rough number of characters per token used by the
// token-cost heuristic. It approximates the behaviour of cl100k-style
// tokenizers (~4 chars/token for English + JSON) without pulling in a real
// tokenizer dependency. The result is suitable for *relative* comparison
// (which servers/tools cost the most context), not for billing.
const charsPerToken = 4

// toolShape is the minimal tool definition a client sees in a tools/list
// response. Estimating over this shape keeps the fresh-discovery path and the
// running-instance path consistent, since both reduce to name + description +
// input schema.
type toolShape struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// EstimateToolTokens returns a rough estimate of the token cost of an exposed
// tool definition (name + description + input schema). It is a heuristic, not
// an exact tokenizer count.
func EstimateToolTokens(tool mcp.Tool) int {
	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		schema = nil
	}

	return EstimateRawToolTokens(tool.Name, tool.Description, schema)
}

// EstimateRawToolTokens estimates token cost from the raw fields of a tool
// definition. This is used by the instance socket path, where the input schema
// arrives as already-encoded JSON.
func EstimateRawToolTokens(name, description string, inputSchema json.RawMessage) int {
	data, err := json.Marshal(toolShape{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	})
	if err != nil {
		return 0
	}

	return estimateTokens(len(data))
}

// estimateTokens converts a byte length into an estimated token count,
// rounding up so that any non-empty content costs at least one token.
func estimateTokens(n int) int {
	if n <= 0 {
		return 0
	}

	return (n + charsPerToken - 1) / charsPerToken
}

// EstimateCatalogTokens groups estimated token costs by server name and returns
// the per-server breakdown alongside the total across all entries.
func EstimateCatalogTokens(entries []*ToolEntry) (map[string]int, int) {
	perServer := make(map[string]int)
	total := 0

	for _, e := range entries {
		if e == nil {
			continue
		}

		cost := EstimateToolTokens(e.ExposedTool())
		perServer[e.ServerName] += cost
		total += cost
	}

	return perServer, total
}
