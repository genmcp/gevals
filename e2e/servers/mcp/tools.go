package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolHandler is a function that handles a tool call
type ToolHandler func(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error)

// ToolDef defines a tool to be registered with the mock MCP server
type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any // JSON Schema properties
	Required    []string       // Required property names

	// Response configuration (use one of these)
	Result  *mcp.CallToolResult // Static result to return
	Error   error               // Error to return
	Handler ToolHandler         // Dynamic handler function
}

// NewTool creates a new tool definition with the given name
func NewTool(name string) *ToolDef {
	return &ToolDef{
		Name:        name,
		InputSchema: make(map[string]any),
		Required:    make([]string, 0),
	}
}

// WithDescription sets the tool's description
func (t *ToolDef) WithDescription(desc string) *ToolDef {
	t.Description = desc
	return t
}

// WithStringParam adds a string parameter to the tool
func (t *ToolDef) WithStringParam(name, description string, required bool) *ToolDef {
	t.InputSchema[name] = map[string]any{
		"type":        "string",
		"description": description,
	}
	if required {
		t.Required = append(t.Required, name)
	}
	return t
}

// WithIntParam adds an integer parameter to the tool
func (t *ToolDef) WithIntParam(name, description string, required bool) *ToolDef {
	t.InputSchema[name] = map[string]any{
		"type":        "integer",
		"description": description,
	}
	if required {
		t.Required = append(t.Required, name)
	}
	return t
}

// WithBoolParam adds a boolean parameter to the tool
func (t *ToolDef) WithBoolParam(name, description string, required bool) *ToolDef {
	t.InputSchema[name] = map[string]any{
		"type":        "boolean",
		"description": description,
	}
	if required {
		t.Required = append(t.Required, name)
	}
	return t
}

// WithObjectParam adds an object parameter to the tool
func (t *ToolDef) WithObjectParam(name, description string, required bool) *ToolDef {
	t.InputSchema[name] = map[string]any{
		"type":        "object",
		"description": description,
	}
	if required {
		t.Required = append(t.Required, name)
	}
	return t
}

// WithArrayParam adds an array parameter to the tool
func (t *ToolDef) WithArrayParam(name, description string, itemType string, required bool) *ToolDef {
	t.InputSchema[name] = map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": itemType,
		},
	}
	if required {
		t.Required = append(t.Required, name)
	}
	return t
}

// Returns sets a static result for the tool to return
func (t *ToolDef) Returns(result *mcp.CallToolResult) *ToolDef {
	t.Result = result
	t.Error = nil
	t.Handler = nil
	return t
}

// ReturnsText sets the tool to return a text result
func (t *ToolDef) ReturnsText(text string) *ToolDef {
	return t.Returns(TextResult(text))
}

// ReturnsJSON sets the tool to return a JSON result
func (t *ToolDef) ReturnsJSON(data any) *ToolDef {
	return t.Returns(JSONResult(data))
}

// ReturnsError sets the tool to return an error
func (t *ToolDef) ReturnsError(err error) *ToolDef {
	t.Error = err
	t.Result = nil
	t.Handler = nil
	return t
}

// ReturnsErrorText sets the tool to return an error result (not a Go error)
func (t *ToolDef) ReturnsErrorText(message string) *ToolDef {
	return t.Returns(ErrorResult(message))
}

// WithHandler sets a dynamic handler for the tool
func (t *ToolDef) WithHandler(handler ToolHandler) *ToolDef {
	t.Handler = handler
	t.Result = nil
	t.Error = nil
	return t
}

// Result helper functions

// TextResult creates a text content result
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}
}

// JSONResult creates a JSON content result
func JSONResult(data any) *mcp.CallToolResult {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ErrorResult("failed to marshal JSON: " + err.Error())
	}
	return TextResult(string(jsonBytes))
}

// ErrorResult creates an error result (isError=true)
func ErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: message,
			},
		},
		IsError: true,
	}
}

// EmptyResult creates an empty result
func EmptyResult() *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{},
	}
}
