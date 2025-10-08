package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/shared"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client wraps MCP SDK functionality for tool calling
type Client struct {
	session *mcpsdk.ClientSession
	tools   []mcpsdk.Tool
}

// NewClient creates a new MCP client connection over HTTP
func NewClient(ctx context.Context, serverURL string) (*Client, error) {
	// Create MCP client with implementation info
	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "gevals-agent",
		Version: "1.0.0",
	}, nil)

	// Create the streamable HTTP transport
	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: serverURL,
	}

	// Connect to the server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	return &Client{
		session: session,
	}, nil
}

// LoadTools fetches available tools from the MCP server
func (c *Client) LoadTools(ctx context.Context) error {
	result, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert from []*Tool to []Tool
	c.tools = make([]mcpsdk.Tool, len(result.Tools))
	for i, tool := range result.Tools {
		if tool != nil {
			c.tools[i] = *tool
		}
	}
	return nil
}

// GetTools returns the available tools as OpenAI function definitions
func (c *Client) GetTools() []openai.ChatCompletionToolUnionParam {
	var openaiTools []openai.ChatCompletionToolUnionParam

	for _, tool := range c.tools {
		openaiTool := convertMCPToolToOpenAI(tool)
		openaiTools = append(openaiTools, openaiTool)
	}

	return openaiTools
}

// CallTool executes a tool call through the MCP server
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (string, error) {
	result, err := c.session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	// Convert result to string representation
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool result: %w", err)
	}

	return string(resultBytes), nil
}

// Close closes the MCP client connection
func (c *Client) Close() error {
	return c.session.Close()
}

// convertMCPToolToOpenAI converts an MCP tool definition to OpenAI function calling format
func convertMCPToolToOpenAI(tool mcpsdk.Tool) openai.ChatCompletionToolUnionParam {
	// Create function definition
	function := shared.FunctionDefinitionParam{
		Name: tool.Name,
	}

	// Add description if available
	if tool.Description != "" {
		function.Description = openai.String(tool.Description)
	}

	// If the tool has input schema, convert it to OpenAI parameters format
	if tool.InputSchema != nil {
		// The MCP tool schema should be compatible with JSON Schema
		// which OpenAI function calling expects
		if params, ok := tool.InputSchema.(map[string]interface{}); ok {
			function.Parameters = shared.FunctionParameters(params)
		}
	}

	// Use the helper function to create the tool
	return openai.ChatCompletionFunctionTool(function)
}