package mcpclient

import (
	"context"
	"net/http"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Connect(ctx context.Context, cfg *ServerConfig) (*mcp.ClientSession, error) {
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

	return cs, nil
}
