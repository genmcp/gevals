package mcpproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	proxyServer *mcp.Server
	proxyClient *mcp.ClientSession
	cfg         *ServerConfig // TODO(Cali0707): see if we actually need this
	url         string
}

func NewProxyServerForConfig(ctx context.Context, config *ServerConfig) (*Server, error) {
	cs, err := createProxyClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy client for %+v: %w", config, err)
	}

	s, err := createProxyServer(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy server for %+v: %w", config, err)
	}

	return &Server{
		proxyServer: s,
		proxyClient: cs,
	}, nil
}

func createProxyClient(ctx context.Context, config *ServerConfig) (*mcp.ClientSession, error) {
	var transport mcp.Transport
	if config.IsHttp() {
		client := &http.Client{
			Transport: NewHeaderRoundTripper(config.Headers, nil),
		}

		transport = &mcp.StreamableClientTransport{
			Endpoint:   config.URL,
			HTTPClient: client,
		}
	} else {
		cmd := exec.Command(config.Args[0], config.Args[1:]...)
		transport = &mcp.CommandTransport{Command: cmd}
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "gevals-proxy-client",
		Version: "0.0.0",
	}, nil)

	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func createProxyServer(ctx context.Context, cs *mcp.ClientSession) (*mcp.Server, error) {
	opts := &mcp.ServerOptions{
		Instructions: cs.InitializeResult().Instructions,
		HasPrompts:   cs.InitializeResult().Capabilities.Prompts != nil,
		HasResources: cs.InitializeResult().Capabilities.Resources != nil,
		HasTools:     cs.InitializeResult().Capabilities.Tools != nil,
	}
	s := mcp.NewServer(
		cs.InitializeResult().ServerInfo,
		opts,
	)

	if opts.HasPrompts {
		for p, err := range cs.Prompts(ctx, &mcp.ListPromptsParams{}) {
			if err != nil {
				continue
			}
			s.AddPrompt(p, func(ctx context.Context, gpr *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				return cs.GetPrompt(ctx, gpr.Params)
			})
		}
	}

	if opts.HasResources {
		for r, err := range cs.Resources(ctx, &mcp.ListResourcesParams{}) {
			if err != nil {
				continue
			}
			s.AddResource(r, func(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				return cs.ReadResource(ctx, rrr.Params)
			})
		}

		for rt, err := range cs.ResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{}) {
			if err != nil {
				continue
			}
			s.AddResourceTemplate(rt, func(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				return cs.ReadResource(ctx, rrr.Params)
			})
		}
	}

	if opts.HasTools {
		for t, err := range cs.Tools(ctx, &mcp.ListToolsParams{}) {
			if err != nil {
				continue
			}
			s.AddTool(t, func(ctx context.Context, ctr *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return cs.CallTool(ctx, &mcp.CallToolParams{
					Meta:      ctr.Params.Meta,
					Name:      ctr.Params.Name,
					Arguments: ctr.Params.Arguments,
				})
			})
		}
	}

	return s, nil
}

// Run is a blocking call until ctx is cancelled
// Run will start the server in streamablehttp transport
// TODO(Cali0707): update this to support other transports
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.proxyServer
	}, &mcp.StreamableHTTPOptions{})

	mux.Handle("/mcp", handler)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("failed to start listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	s.url = fmt.Sprintf("http://localhost:%d/mcp", port)

	return http.Serve(listener, mux)
}

func (s *Server) GetConfig() (*ServerConfig, error) {
	if s.url == "" {
		return nil, fmt.Errorf("url must be set for config to be valid, ensure Run() is called before GetConfig()")
	}

	return &ServerConfig{
		Type:    TransportTypeHttp,
		URL:     s.url,
		Headers: s.cfg.Headers,
	}, nil
}

func (s *Server) Close() error {
	return s.proxyClient.Close()
}
