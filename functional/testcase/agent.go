package testcase

import (
	"github.com/mcpchecker/mcpchecker/functional/servers/agent"
)

// AgentBuilder provides a fluent API for configuring mock agent behavior.
// It wraps agent.Config and provides a test-friendly interface.
type AgentBuilder struct {
	config *agent.Config
}

// NewAgentBuilder creates a new agent builder
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		config: agent.NewConfig(),
	}
}

// OnPromptContaining adds a behavior that matches when the prompt contains a substring.
// Returns a BehaviorBuilder to configure the behavior's actions.
func (b *AgentBuilder) OnPromptContaining(substring string) *BehaviorBuilder {
	behavior := agent.NewBehavior().OnPromptContaining(substring)
	return &BehaviorBuilder{agentBuilder: b, behavior: behavior}
}

// OnPromptMatching adds a behavior that matches when the prompt matches a regex pattern.
// Returns a BehaviorBuilder to configure the behavior's actions.
func (b *AgentBuilder) OnPromptMatching(pattern string) *BehaviorBuilder {
	behavior := agent.NewBehavior().OnPromptMatching(pattern)
	return &BehaviorBuilder{agentBuilder: b, behavior: behavior}
}

// OnAnyPrompt adds a behavior that matches any prompt (catch-all).
// Returns a BehaviorBuilder to configure the behavior's actions.
func (b *AgentBuilder) OnAnyPrompt() *BehaviorBuilder {
	behavior := agent.NewBehavior().OnAnyPrompt()
	return &BehaviorBuilder{agentBuilder: b, behavior: behavior}
}

// DefaultResponse sets the response when no behavior matches
func (b *AgentBuilder) DefaultResponse(response string) *AgentBuilder {
	b.config.WithDefaultResponse(response)
	return b
}

// DefaultError sets an error to return when no behavior matches
func (b *AgentBuilder) DefaultError(err string) *AgentBuilder {
	b.config.WithDefaultError(err)
	return b
}

// VerifyMCPConnectivity enables MCP connectivity verification before processing.
// The agent will connect to all configured MCP servers and list their tools.
func (b *AgentBuilder) VerifyMCPConnectivity() *AgentBuilder {
	b.config.WithVerifyMCPConnectivity()
	return b
}

// VerifyAllowedTools sets tools that must be available from MCP servers.
// The agent will verify all specified tools exist before processing.
func (b *AgentBuilder) VerifyAllowedTools(tools ...string) *AgentBuilder {
	b.config.WithVerifyAllowedTools(tools...)
	return b
}

// Build returns the agent configuration
func (b *AgentBuilder) Build() *agent.Config {
	return b.config
}

// BehaviorBuilder provides a fluent API for building a single behavior.
// Call ThenRespond() or ThenFail() to finalize and return to the AgentBuilder.
type BehaviorBuilder struct {
	agentBuilder *AgentBuilder
	behavior     *agent.Behavior
}

// WithName sets an optional name for this behavior (useful for debugging)
func (bb *BehaviorBuilder) WithName(name string) *BehaviorBuilder {
	bb.behavior.WithName(name)
	return bb
}

// CallTool adds a tool call to this behavior.
// The tool will be called on the first available MCP server.
func (bb *BehaviorBuilder) CallTool(name string, args map[string]any) *BehaviorBuilder {
	bb.behavior.CallTool(name, args)
	return bb
}

// CallToolOnServer adds a tool call to a specific MCP server
func (bb *BehaviorBuilder) CallToolOnServer(server, name string, args map[string]any) *BehaviorBuilder {
	bb.behavior.CallToolOnServer(server, name, args)
	return bb
}

// CallToolExpectingError adds a tool call that is expected to return an error
func (bb *BehaviorBuilder) CallToolExpectingError(name string, args map[string]any) *BehaviorBuilder {
	bb.behavior.ToolCalls = append(bb.behavior.ToolCalls, agent.ToolCallSpec{
		Name:        name,
		Arguments:   args,
		ExpectError: true,
	})
	return bb
}

// ThenRespond sets the response and finalizes this behavior.
// Returns the AgentBuilder to continue configuration.
func (bb *BehaviorBuilder) ThenRespond(response string) *AgentBuilder {
	bb.behavior.ThenRespond(response)
	bb.agentBuilder.config.AddBehavior(*bb.behavior)
	return bb.agentBuilder
}

// ThenFail sets an error response and finalizes this behavior.
// The agent will exit with an error instead of responding.
// Returns the AgentBuilder to continue configuration.
func (bb *BehaviorBuilder) ThenFail(err string) *AgentBuilder {
	bb.behavior.ThenFail(err)
	bb.agentBuilder.config.AddBehavior(*bb.behavior)
	return bb.agentBuilder
}

// Re-export types from agent package for convenience
type (
	AgentConfig  = agent.Config
	Behavior     = agent.Behavior
	ToolCallSpec = agent.ToolCallSpec
	MCPConfig    = agent.MCPConfig
	ServerConfig = agent.ServerConfig
)

// Re-export constants and helpers from agent package
const EnvAgentConfigPath = agent.EnvConfigPath

var (
	NewAgentConfig  = agent.NewConfig
	NewBehavior     = agent.NewBehavior
	LoadAgentConfig = agent.LoadConfig
	SaveAgentConfig = agent.SaveConfig
)
