package codemode

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"go.starlark.net/starlark"
)

// truncMarker is appended once when output is truncated. Its length is reserved
// against the buffer limit so total output never exceeds the configured cap.
const truncMarker = "\n…(output truncated)"

// maxConvertDepth bounds recursion when converting Starlark values to Go,
// guarding against a script building a deeply nested structure that could
// otherwise overflow the Go stack during conversion.
const maxConvertDepth = 64

// errTooDeep is returned when an argument structure exceeds maxConvertDepth.
var errTooDeep = fmt.Errorf("argument nesting too deep (max %d)", maxConvertDepth)

// valueToArgs converts a Starlark value (expected to be a dict or None) into a
// Go argument map suitable for an MCP tool call.
func valueToArgs(v starlark.Value) (map[string]any, error) {
	if v == nil || v == starlark.None {
		return map[string]any{}, nil
	}

	dict, ok := v.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("args must be a dict, got %s", v.Type())
	}

	return dictToMap(dict, 0)
}

// dictToMap converts a Starlark dict with string keys into a Go map.
func dictToMap(dict *starlark.Dict, depth int) (map[string]any, error) {
	if depth > maxConvertDepth {
		return nil, errTooDeep
	}

	out := make(map[string]any, dict.Len())

	for _, item := range dict.Items() {
		key, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %s", item[0].Type())
		}

		val, err := starlarkToGo(item[1], depth+1)
		if err != nil {
			return nil, err
		}

		out[key] = val
	}

	return out, nil
}

// starlarkToGo converts a Starlark value into its natural Go representation.
func starlarkToGo(v starlark.Value, depth int) (any, error) {
	if depth > maxConvertDepth {
		return nil, errTooDeep
	}

	switch t := v.(type) {
	case starlark.NoneType:
		//nolint:nilnil // Starlark None maps to a nil Go value (JSON null); not an error.
		return nil, nil
	case starlark.Bool:
		return bool(t), nil
	case starlark.Int:
		if i, ok := t.Int64(); ok {
			return i, nil
		}

		return t.String(), nil
	case starlark.Float:
		return float64(t), nil
	case starlark.String:
		return string(t), nil
	case *starlark.List:
		return iterableToGo(t, depth)
	case starlark.Tuple:
		return iterableToGo(t, depth)
	case *starlark.Dict:
		return dictToMap(t, depth)
	default:
		return nil, fmt.Errorf("unsupported argument type %s", v.Type())
	}
}

// iterableToGo converts a Starlark iterable into a Go slice.
func iterableToGo(it starlark.Iterable, depth int) ([]any, error) {
	if depth > maxConvertDepth {
		return nil, errTooDeep
	}

	iter := it.Iterate()
	defer iter.Done()

	var (
		out  []any
		elem starlark.Value
	)

	for iter.Next(&elem) {
		val, err := starlarkToGo(elem, depth+1)
		if err != nil {
			return nil, err
		}

		out = append(out, val)
	}

	return out, nil
}

// boundedBuffer accumulates newline-terminated output up to a byte limit,
// appending a truncation marker once the limit is reached.
type boundedBuffer struct {
	mu    sync.Mutex
	buf   strings.Builder
	limit int
	full  bool
}

func (b *boundedBuffer) writeLine(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.full {
		return
	}

	// Reserve room for the marker so content + marker never exceeds limit.
	budget := max(b.limit-len(truncMarker), 0)

	remaining := budget - b.buf.Len()
	if remaining <= 0 {
		b.markFull()

		return
	}

	line := s + "\n"
	if len(line) <= remaining {
		b.buf.WriteString(line)

		return
	}

	// Truncate within the remaining budget at a UTF-8 boundary.
	b.buf.WriteString(line[:validCut(line, remaining)])
	b.markFull()
}

// markFull appends the truncation marker exactly once.
func (b *boundedBuffer) markFull() {
	b.buf.WriteString(truncMarker)
	b.full = true
}

// validCut returns the largest index <= maxBytes that does not split a UTF-8
// rune, so s[:validCut] is always valid UTF-8.
func validCut(s string, maxBytes int) int {
	if maxBytes >= len(s) {
		return len(s)
	}

	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}

	return cut
}

func (b *boundedBuffer) string() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}
