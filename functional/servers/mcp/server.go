package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MockMCPServer implements a mock MCP server using Streamable HTTP transport
type MockMCPServer struct {
	mu       sync.Mutex
	name     string
	tools    []*ToolDef
	calls    []CapturedToolCall
	server   *mcp.Server
	listener net.Listener
	httpSrv  *http.Server
	ready    chan struct{}
}

// CapturedToolCall stores details of a tool invocation for assertions
type CapturedToolCall struct {
	ToolName  string
	Arguments map[string]any
	Result    *mcp.CallToolResult
	Error     error
	Timestamp time.Time
}

// NewMockMCPServer creates a new mock MCP server with the given name
func NewMockMCPServer(name string) *MockMCPServer {
	return &MockMCPServer{
		name:  name,
		tools: make([]*ToolDef, 0),
		calls: make([]CapturedToolCall, 0),
		ready: make(chan struct{}),
	}
}

// AddTool registers a tool with the mock server
func (s *MockMCPServer) AddTool(tool *ToolDef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
}

// Start starts the server on a random available port and returns the URL
func (s *MockMCPServer) Start() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create the MCP server
	s.server = mcp.NewServer(
		&mcp.Implementation{
			Name:    s.name,
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			HasTools: len(s.tools) > 0,
		},
	)

	// Register all tools
	for _, toolDef := range s.tools {
		s.registerTool(toolDef)
	}

	// Listen on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to listen on random port: %w", err)
	}
	s.listener = listener

	// Create HTTP handler for MCP
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.server
	}, &mcp.StreamableHTTPOptions{})

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	s.httpSrv = &http.Server{
		Handler: mux,
	}

	// Start serving
	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP mock server %q error: %v\n", s.name, err)
		}
	}()

	close(s.ready)
	return s.URL(), nil
}

// registerTool adds a tool to the MCP server
func (s *MockMCPServer) registerTool(toolDef *ToolDef) {
	// Build the input schema as a map
	inputSchema := map[string]any{
		"type":       "object",
		"properties": toolDef.InputSchema,
	}
	if len(toolDef.Required) > 0 {
		inputSchema["required"] = toolDef.Required
	}

	// Build the MCP tool definition
	mcpTool := &mcp.Tool{
		Name:        toolDef.Name,
		Description: toolDef.Description,
		InputSchema: inputSchema,
	}

	// Create handler that captures calls and returns configured response
	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse arguments from the request
		args := parseArguments(req.Params.Arguments)

		captured := CapturedToolCall{
			ToolName:  req.Params.Name,
			Arguments: args,
			Timestamp: time.Now(),
		}

		var result *mcp.CallToolResult
		var err error

		// Use custom handler if provided, otherwise use static result
		if toolDef.Handler != nil {
			result, err = toolDef.Handler(ctx, args)
		} else if toolDef.Result != nil {
			result = toolDef.Result
		} else if toolDef.Error != nil {
			err = toolDef.Error
		} else {
			// Default empty result
			result = &mcp.CallToolResult{
				Content: []mcp.Content{},
			}
		}

		captured.Result = result
		captured.Error = err

		s.mu.Lock()
		s.calls = append(s.calls, captured)
		s.mu.Unlock()

		return result, err
	}

	s.server.AddTool(mcpTool, handler)
}

// parseArguments converts the Arguments (which can be any) to map[string]any.
// Returns an empty map if conversion fails, logging a warning for debugging.
func parseArguments(args any) map[string]any {
	if args == nil {
		return make(map[string]any)
	}

	// If it's already a map, return it
	if m, ok := args.(map[string]any); ok {
		return m
	}

	// Try to convert via JSON
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to marshal tool arguments to JSON: %v\n", err)
		return make(map[string]any)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to unmarshal tool arguments from JSON: %v\n", err)
		return make(map[string]any)
	}

	return result
}

// Stop stops the server
func (s *MockMCPServer) Stop() error {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

// URL returns the server's MCP endpoint URL (e.g., "http://127.0.0.1:12345/mcp")
func (s *MockMCPServer) URL() string {
	if s.listener == nil {
		return ""
	}
	return fmt.Sprintf("http://%s/mcp", s.listener.Addr().String())
}

// Name returns the server's name
func (s *MockMCPServer) Name() string {
	return s.name
}

// Calls returns all captured tool calls
func (s *MockMCPServer) Calls() []CapturedToolCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]CapturedToolCall, len(s.calls))
	copy(result, s.calls)
	return result
}

// CallCount returns the number of captured tool calls
func (s *MockMCPServer) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// CallsForTool returns all calls to a specific tool
func (s *MockMCPServer) CallsForTool(toolName string) []CapturedToolCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]CapturedToolCall, 0)
	for _, call := range s.calls {
		if call.ToolName == toolName {
			result = append(result, call)
		}
	}
	return result
}

// LastCall returns the most recent captured call, or nil if none
func (s *MockMCPServer) LastCall() *CapturedToolCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		return nil
	}
	call := s.calls[len(s.calls)-1]
	return &call
}

// Reset clears all captured calls
func (s *MockMCPServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = make([]CapturedToolCall, 0)
}

// WaitReady blocks until the server is ready
func (s *MockMCPServer) WaitReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
