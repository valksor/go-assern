package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/valksor/go-assern/internal/config"
)

func BenchmarkToolRegistry_Register(b *testing.B) {
	b.ReportAllocs()

	for range b.N {
		registry := NewToolRegistry()
		for j := range 100 {
			registry.Register("server"+string(rune('0'+j%10)), mcp.Tool{
				Name:        "tool" + string(rune('0'+j)),
				Description: "Test tool",
			}, nil)
		}
	}
}

func BenchmarkToolRegistry_Get(b *testing.B) {
	registry := NewToolRegistry()
	for i := range 100 {
		registry.Register("server"+string(rune('0'+i%10)), mcp.Tool{
			Name:        "tool" + string(rune('0'+i)),
			Description: "Test tool",
		}, nil)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		registry.Get("server5_tool5")
	}
}

func BenchmarkToolRegistry_GetWithAlias(b *testing.B) {
	registry := NewToolRegistry()
	for i := range 100 {
		registry.Register("server"+string(rune('0'+i%10)), mcp.Tool{
			Name:        "tool" + string(rune('0'+i)),
			Description: "Test tool",
		}, nil)
	}
	registry.SetAliases(map[string]string{
		"shortcut": "server5_tool5",
	})

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		registry.Get("shortcut")
	}
}

func BenchmarkToolRegistry_All(b *testing.B) {
	registry := NewToolRegistry()
	for i := range 100 {
		registry.Register("server"+string(rune('0'+i%10)), mcp.Tool{
			Name:        "tool" + string(rune('0'+i)),
			Description: "Test tool",
		}, nil)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = registry.All()
	}
}

func BenchmarkToolRegistry_All_Cached(b *testing.B) {
	registry := NewToolRegistry()
	for i := range 100 {
		registry.Register("server"+string(rune('0'+i%10)), mcp.Tool{
			Name:        "tool" + string(rune('0'+i)),
			Description: "Test tool",
		}, nil)
	}

	// Prime the cache
	_ = registry.All()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = registry.All()
	}
}

func BenchmarkPrefixToolName(b *testing.B) {
	b.ReportAllocs()

	for range b.N {
		PrefixToolName("my-server-name", "my-tool-name")
	}
}

func BenchmarkParsePrefixedName(b *testing.B) {
	b.ReportAllocs()

	for range b.N {
		_, _, _ = ParsePrefixedName("server_tool_name")
	}
}

func BenchmarkHealthTracker_RecordSuccess(b *testing.B) {
	ht := NewHealthTracker(3)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		ht.RecordSuccess("server1")
	}
}

func BenchmarkHealthTracker_RecordFailure(b *testing.B) {
	ht := NewHealthTracker(1000) // High threshold to avoid state change overhead

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		ht.RecordFailure("server1")
	}
}

func BenchmarkHealthTracker_AllStats(b *testing.B) {
	ht := NewHealthTracker(3)
	for i := range 50 {
		server := "server" + string(rune('0'+i%10))
		ht.RecordSuccess(server)
		ht.RecordFailure(server)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = ht.AllStats()
	}
}

func BenchmarkWithRetry_NoRetry(b *testing.B) {
	ctx := context.Background()
	fn := func(ctx context.Context, attempt int) (string, error) {
		return "ok", nil
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, _ = WithRetry(ctx, nil, fn)
	}
}

func BenchmarkWithRetry_WithConfig(b *testing.B) {
	ctx := context.Background()
	cfg := &config.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}
	fn := func(ctx context.Context, attempt int) (string, error) {
		return "ok", nil
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, _ = WithRetry(ctx, cfg, fn)
	}
}

func BenchmarkResourceRegistry_All(b *testing.B) {
	registry := NewResourceRegistry()
	for i := range 100 {
		registry.Register("server"+string(rune('0'+i%10)), mcp.Resource{
			URI:  "file:///path/to/resource" + string(rune('0'+i)),
			Name: "resource" + string(rune('0'+i)),
		})
	}

	// Prime cache
	_ = registry.All()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = registry.All()
	}
}

func BenchmarkPromptRegistry_All(b *testing.B) {
	registry := NewPromptRegistry()
	for i := range 100 {
		registry.Register("server"+string(rune('0'+i%10)), mcp.Prompt{
			Name:        "prompt" + string(rune('0'+i)),
			Description: "Test prompt",
		})
	}

	// Prime cache
	_ = registry.All()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = registry.All()
	}
}
