package aggregator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mark3labs/mcp-go/client/transport"
)

func TestFileTokenStoreRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := newFileTokenStore(dir, "github")
	ctx := context.Background()

	// No token yet.
	if _, err := store.GetToken(ctx); !errors.Is(err, transport.ErrNoToken) {
		t.Fatalf("GetToken on empty store = %v, want ErrNoToken", err)
	}

	want := &transport.Token{AccessToken: "secret", TokenType: "Bearer", RefreshToken: "r"}
	if err := store.SaveToken(ctx, want); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err := store.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken after save: %v", err)
	}

	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestFileTokenStorePermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("file mode bits are not meaningful on Windows")
	}

	dir := t.TempDir()
	store := newFileTokenStore(dir, "github")

	if err := store.SaveToken(context.Background(), &transport.Token{AccessToken: "x"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "github.json"))
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token file perms = %o, want 600", perm)
	}
}

func TestFileTokenStoreSanitizesKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// A key with path separators must not escape the token directory.
	store := newFileTokenStore(dir, "../../etc/passwd")

	if err := store.SaveToken(context.Background(), &transport.Token{AccessToken: "x"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected exactly one token file in dir, got %d", len(entries))
	}

	// Path separators become underscores so the key cannot escape the dir.
	// Dots are allowed in filenames, so they are preserved.
	if got := entries[0].Name(); got != ".._.._etc_passwd.json" {
		t.Errorf("sanitized filename = %q, want .._.._etc_passwd.json", got)
	}
}
