package instance

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
)

// Proxy connects to an existing assern instance and bridges stdio to it.
type Proxy struct {
	socketPath string
	logger     *slog.Logger
	conn       net.Conn
}

// NewProxy creates a new proxy to an existing instance.
func NewProxy(socketPath string, logger *slog.Logger) *Proxy {
	return &Proxy{
		socketPath: socketPath,
		logger:     logger,
	}
}

// Connect establishes connection to the primary instance.
func (p *Proxy) Connect(ctx context.Context) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", p.socketPath)
	if err != nil {
		return err
	}

	p.conn = conn

	return nil
}

// Close closes the connection to the primary instance.
func (p *Proxy) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}

	return nil
}

// ServeStdio bridges stdin/stdout to the socket connection.
// This makes the proxy transparent to the calling LLM.
func (p *Proxy) ServeStdio(ctx context.Context) error {
	if p.conn == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	p.logger.Info("proxy connected - forwarding stdio to primary instance")

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// stdin -> socket
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(p.conn, os.Stdin)
		if err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	// socket -> stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(os.Stdout, p.conn)
		if err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	// Wait for context cancellation or connection close
	select {
	case <-ctx.Done():
		_ = p.conn.Close()
	case err := <-errCh:
		p.logger.Debug("proxy connection closed", "error", err)
		_ = p.conn.Close()
	}

	wg.Wait()

	return nil
}
