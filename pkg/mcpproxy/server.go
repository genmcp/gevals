package mcpproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server interface {
	Run(ctx context.Context) error
	GetConfig() (*ServerConfig, error)
	GetName() string
	GetAllowedToolNames() []string
	Close() error
	GetCallHistory() CallHistory
}

type server struct {
	name        string
	proxyServer *mcp.Server
	proxyClient *mcp.ClientSession
	cfg         *ServerConfig // TODO(Cali0707): see if we actually need this
	url         string

	// Call tracking
	recorder Recorder
}

var _ Server = &server{}

func NewProxyServerForConfig(ctx context.Context, name string, config *ServerConfig) (Server, error) {
	cs, err := createProxyClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy client for %+v: %w", config, err)
	}

	r := NewRecorder(name)

	s, err := createProxyServer(ctx, cs, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy server for %+v: %w", config, err)
	}

	return &server{
		name:        name,
		proxyServer: s,
		proxyClient: cs,
		cfg:         config,
		recorder:    r,
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
		cmd := exec.Command(config.Command, config.Args...)
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

func createProxyServer(ctx context.Context, cs *mcp.ClientSession, r Recorder) (*mcp.Server, error) {
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
				start := time.Now()
				res, err := cs.GetPrompt(ctx, gpr.Params)
				r.RecordPromptGet(gpr, res, err, start)
				return res, err
			})
		}
	}

	if opts.HasResources {
		for rr, err := range cs.Resources(ctx, &mcp.ListResourcesParams{}) {
			if err != nil {
				continue
			}
			s.AddResource(rr, func(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				start := time.Now()
				res, err := cs.ReadResource(ctx, rrr.Params)
				r.RecordResourceRead(rrr, res, err, start)
				return res, err
			})
		}

		for rt, err := range cs.ResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{}) {
			if err != nil {
				continue
			}
			s.AddResourceTemplate(rt, func(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				start := time.Now()
				res, err := cs.ReadResource(ctx, rrr.Params)
				r.RecordResourceRead(rrr, res, err, start)
				return res, err
			})
		}
	}

	if opts.HasTools {
		for t, err := range cs.Tools(ctx, &mcp.ListToolsParams{}) {
			if err != nil {
				continue
			}
			s.AddTool(t, func(ctx context.Context, ctr *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				start := time.Now()
				res, err := cs.CallTool(ctx, &mcp.CallToolParams{
					Meta:      ctr.Params.Meta,
					Name:      ctr.Params.Name,
					Arguments: ctr.Params.Arguments,
				})
				r.RecordToolCall(ctr, res, err, start)
				return res, err
			})
		}
	}

	return s, nil
}

// Run is a blocking call until ctx is cancelled
// Run will start the server in streamablehttp transport
// TODO(Cali0707): update this to support other transports
func (s *server) Run(ctx context.Context) error {
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

	httpServer := &http.Server{
		Handler: mux,
	}

	// Run server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Context cancelled, shutdown gracefully
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		return nil
	case err := <-serverErr:
		// Server error
		return err
	}
}

func (s *server) GetConfig() (*ServerConfig, error) {
	if s.url == "" {
		return nil, fmt.Errorf("url must be set for config to be valid, ensure Run() is called before GetConfig()")
	}

	return &ServerConfig{
		Type:    TransportTypeHttp,
		URL:     s.url,
		Headers: s.cfg.Headers,
	}, nil
}

func (s *server) GetName() string {
	return s.name
}

func (s *server) GetAllowedToolNames() []string {
	if s.cfg.EnableAllTools {
		toolNames := make([]string, 0)
		for t, err := range s.proxyClient.Tools(context.Background(), &mcp.ListToolsParams{}) {
			if err != nil {
				continue
			}

			toolNames = append(toolNames, t.Name)
		}

		return toolNames
	}

	return s.cfg.AlwaysAllow
}

func (s *server) Close() error {
	return s.proxyClient.Close()
}

func (s *server) GetCallHistory() CallHistory {
	return s.recorder.GetHistory()
}
