package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Manager interface {
	Get(name string) (*mcp.ClientSession, bool)
	Close(ctx context.Context) error
}

var _ Manager = &manager{}

type manager struct {
	sessions map[string]*mcp.ClientSession
}

func NewManager(ctx context.Context, config *MCPConfig) (Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("no config provided")
	}

	servers := config.GetEnabledServers()

	if len(servers) == 0 {
		return nil, fmt.Errorf("no enabled mcp servers found in config")
	}

	m := &manager{
		sessions: make(map[string]*mcp.ClientSession, len(servers)),
	}

	var err error
	for name, cfg := range servers {
		cs, connErr := Connect(ctx, cfg)
		if connErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to connect to mcp server %q: %w", name, connErr))
			continue
		}

		m.sessions[name] = cs
	}

	if err != nil {
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		err = errors.Join(err, m.Close(ctx)) // clean up any successfully made connections
		cancel()

		return nil, err
	}

	return m, nil
}

func (m *manager) Get(name string) (*mcp.ClientSession, bool) {
	cs, ok := m.sessions[name]
	return cs, ok
}

func (m *manager) Close(ctx context.Context) error {
	results := make(chan error, len(m.sessions))

	for _, cs := range m.sessions {
		go func() {
			results <- cs.Close()
		}()
	}

	var err error
	for range len(m.sessions) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case closeErr := <-results:
			err = errors.Join(closeErr)
		}
	}

	return err
}
