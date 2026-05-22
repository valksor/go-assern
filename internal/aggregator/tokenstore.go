package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/mark3labs/mcp-go/client/transport"
)

// tokenKeySanitizer replaces any character that is unsafe in a filename so the
// token store key maps to a predictable, contained path.
var tokenKeySanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// fileTokenStore is a transport.TokenStore that persists OAuth tokens to disk so
// they survive across assern invocations. Tokens are keyed by OAuth profile (or
// server) name and written with owner-only permissions.
type fileTokenStore struct {
	path string
	mu   sync.Mutex
}

// newFileTokenStore returns a token store backed by dir/<sanitized key>.json.
func newFileTokenStore(dir, key string) *fileTokenStore {
	safe := tokenKeySanitizer.ReplaceAllString(key, "_")

	return &fileTokenStore{path: filepath.Join(dir, safe+".json")}
}

// GetToken loads the cached token, returning transport.ErrNoToken when none has
// been stored yet.
func (s *fileTokenStore) GetToken(ctx context.Context) (*transport.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, transport.ErrNoToken
		}

		return nil, fmt.Errorf("reading token: %w", err)
	}

	var token transport.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("decoding token: %w", err)
	}

	return &token, nil
}

// SaveToken persists the token, creating the token directory if needed. The
// file is written with 0600 permissions to protect the credentials.
func (s *fileTokenStore) SaveToken(ctx context.Context, token *transport.Token) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating token dir: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encoding token: %w", err)
	}

	// Write to a temp file then rename, so a crash mid-write never leaves a
	// truncated token file, and the final file always has fresh 0600 perms
	// regardless of any pre-existing file's mode.
	tmp, err := os.CreateTemp(dir, "token-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp token file: %w", err)
	}

	tmpPath := tmp.Name()

	if err := s.writeTemp(tmp, data); err != nil {
		_ = os.Remove(tmpPath)

		return err
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("finalizing token file: %w", err)
	}

	return nil
}

// writeTemp chmods, writes, and closes the temp file, returning the first error.
func (s *fileTokenStore) writeTemp(tmp *os.File, data []byte) error {
	chmodErr := tmp.Chmod(0o600)
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()

	switch {
	case chmodErr != nil:
		return fmt.Errorf("setting token file perms: %w", chmodErr)
	case writeErr != nil:
		return fmt.Errorf("writing token: %w", writeErr)
	case closeErr != nil:
		return fmt.Errorf("closing token file: %w", closeErr)
	default:
		return nil
	}
}

// Compile-time guarantee that fileTokenStore satisfies the mcp-go interface.
var _ transport.TokenStore = (*fileTokenStore)(nil)
