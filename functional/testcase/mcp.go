package testcase

import (
	"github.com/genmcp/gevals/functional/servers/mcp"
)

// MCPServerBuilder builds a mock MCP server configuration
type MCPServerBuilder struct {
	name  string
	tools []*mcp.ToolDef
}

// NewMCPServerBuilder creates a new MCP server builder
func NewMCPServerBuilder(name string) *MCPServerBuilder {
	return &MCPServerBuilder{
		name:  name,
		tools: make([]*mcp.ToolDef, 0),
	}
}

// Tool adds a tool to the MCP server using a fluent configuration callback.
// The callback receives a *mcp.ToolDef which has methods like:
//   - WithDescription(desc string)
//   - WithStringParam(name, description string, required bool)
//   - WithIntParam(name, description string, required bool)
//   - WithBoolParam(name, description string, required bool)
//   - WithObjectParam(name, description string, required bool)
//   - WithArrayParam(name, description, itemType string, required bool)
//   - ReturnsText(text string)
//   - ReturnsJSON(data any)
//   - ReturnsErrorText(message string)
//   - ReturnsError(err error)
//   - WithHandler(handler ToolHandler)
func (b *MCPServerBuilder) Tool(name string, configure func(*mcp.ToolDef)) *MCPServerBuilder {
	tool := mcp.NewTool(name)
	configure(tool)
	b.tools = append(b.tools, tool)
	return b
}

// AddTool adds a pre-configured tool definition
func (b *MCPServerBuilder) AddTool(tool *mcp.ToolDef) *MCPServerBuilder {
	b.tools = append(b.tools, tool)
	return b
}

// Build creates the mock MCP server with all configured tools
func (b *MCPServerBuilder) Build() *mcp.MockMCPServer {
	server := mcp.NewMockMCPServer(b.name)
	for _, tool := range b.tools {
		server.AddTool(tool)
	}
	return server
}

// Re-export types and helpers from mcp package for convenience
type (
	ToolDef        = mcp.ToolDef
	ToolHandler    = mcp.ToolHandler
	MockMCPServer  = mcp.MockMCPServer
	CapturedToolCall = mcp.CapturedToolCall
)

// Re-export result helpers for convenience
var (
	NewTool     = mcp.NewTool
	TextResult  = mcp.TextResult
	JSONResult  = mcp.JSONResult
	ErrorResult = mcp.ErrorResult
	EmptyResult = mcp.EmptyResult
)
