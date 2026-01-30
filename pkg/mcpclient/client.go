package mcpclient

import (
	"context"
	"net/http"
	"os/exec"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Client struct {
	*mcp.ClientSession
	cfg *ServerConfig
}

func Connect(ctx context.Context, cfg *ServerConfig) (*Client, error) {
	var transport mcp.Transport
	if cfg.IsHttp() {
		client := &http.Client{
			Transport: NewHeaderRoundTripper(cfg.Headers, nil),
		}

		transport = &mcp.StreamableClientTransport{
			Endpoint:   cfg.URL,
			HTTPClient: client,
		}
	} else {
		cmd := exec.Command(cfg.Command, cfg.Args...)
		transport = &mcp.CommandTransport{Command: cmd}
	}

	// TODO: revisit the client options, we probably want to leverage many
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcpchecker-client",
		Version: "0.0.0",
	}, nil)

	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientSession: cs,
		cfg:           cfg,
	}, nil
}

func (c *Client) GetAllowedTools(ctx context.Context) []*mcp.Tool {
	allowed := []*mcp.Tool{}
	for t, err := range c.Tools(context.Background(), &mcp.ListToolsParams{}) {
		if err != nil {
			continue
		}

		if c.cfg.EnableAllTools {
			allowed = append(allowed, t)
		} else if slices.Contains(c.cfg.AlwaysAllow, t.Name) {
			allowed = append(allowed, t)
		}
	}

	return allowed
}

func (c *Client) GetConfig() *ServerConfig {
	return c.cfg
}
