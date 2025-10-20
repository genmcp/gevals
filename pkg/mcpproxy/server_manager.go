package mcpproxy

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"golang.org/x/sync/errgroup"
)

const (
	mcpServerFileName = "mcp-server.json"
)

type ServerManager interface {
	GetMcpServerFiles() ([]string, error)
	GetMcpServers() []Server
	// Start is non-blocking. Caller must ensure this is only called once, and called before Close
	Start(ctx context.Context) error
	// Close closes associated server resrouces. Caller must ensure this is only called once, and called after Start
	Close() error
}

type serverManager struct {
	servers map[string]Server
	tmpDir  string

	cancel context.CancelFunc
	eg     *errgroup.Group
}

func NewServerManger(ctx context.Context, cfg *MCPConfig) (ServerManager, error) {
	servers := make(map[string]Server, len(cfg.MCPServers))
	for n, cfg := range cfg.MCPServers {
		s, err := NewProxyServerForConfig(ctx, n, cfg)
		if err != nil {
			return nil, err
		}

		servers[n] = s
	}

	return &serverManager{
		servers: servers,
	}, nil
}

func (m *serverManager) GetMcpServerFiles() ([]string, error) {
	if m.tmpDir != "" {
		return []string{fmt.Sprintf("%s/%s", m.tmpDir, mcpServerFileName)}, nil
	}

	cfg, err := m.getMcpServers()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}

	err = cfg.ToFile(fmt.Sprintf("%s/%s", tmpDir, mcpServerFileName))
	if err != nil {
		rmErr := os.Remove(tmpDir)
		if rmErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to remove temp dir '%s': %w", tmpDir, rmErr))
		}

		return nil, err
	}

	m.tmpDir = tmpDir

	return []string{fmt.Sprintf("%s/%s", tmpDir, mcpServerFileName)}, nil

}

func (m *serverManager) GetMcpServers() []Server {
	return slices.Collect(maps.Values(m.servers))
}

func (m *serverManager) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	// Use errgroup to start all servers concurrently
	g, gctx := errgroup.WithContext(ctx)
	m.eg = g

	// Start all servers
	for name, srv := range m.servers {
		g.Go(func() error {
			if err := srv.Run(gctx); err != nil {
				return fmt.Errorf("server %s failed: %w", name, err)
			}
			return nil
		})
	}

	return nil
}

func (m *serverManager) Close() error {
	// Signal all servers to stop
	m.cancel()

	// Wait for all servers to finish
	var errs []error
	if err := m.eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		errs = append(errs, err)
	}

	// Close all servers (cleanup connections, etc.)
	for name, srv := range m.servers {
		if err := srv.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close server %s: %w", name, err))
		}
	}

	// Clean up temp directory
	if m.tmpDir != "" {
		if err := os.RemoveAll(m.tmpDir); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove temp dir: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (m *serverManager) getMcpServers() (*MCPConfig, error) {
	cfg := &MCPConfig{
		MCPServers: make(map[string]*ServerConfig),
	}
	for n, s := range m.servers {
		serverCfg, err := s.GetConfig()
		if err != nil {
			return nil, err
		}

		cfg.MCPServers[n] = serverCfg
	}

	return cfg, nil
}
