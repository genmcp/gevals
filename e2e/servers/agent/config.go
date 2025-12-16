package agent

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config defines the mock agent's behavior.
// This is serialized to a file by the test framework and read by the agent binary.
type Config struct {
	// Behaviors define how the agent responds to different prompts.
	// Behaviors are matched in order; the first match wins.
	Behaviors []Behavior `json:"behaviors"`

	// DefaultResponse is returned when no behavior matches
	DefaultResponse string `json:"defaultResponse,omitempty"`

	// DefaultError causes the agent to exit with an error when no behavior matches
	DefaultError string `json:"defaultError,omitempty"`

	// VerifyMCPConnectivity if true, the agent will verify it can connect to
	// all configured MCP servers and list their tools before processing
	VerifyMCPConnectivity bool `json:"verifyMCPConnectivity,omitempty"`

	// VerifyAllowedTools if set, the agent will verify all these tools are
	// available from the MCP servers before processing
	VerifyAllowedTools []string `json:"verifyAllowedTools,omitempty"`
}

// Behavior defines a response pattern for the mock agent
type Behavior struct {
	// Name is an optional identifier for debugging
	Name string `json:"name,omitempty"`

	// Match conditions (at least one should be set)
	PromptContains string `json:"promptContains,omitempty"`
	PromptMatches  string `json:"promptMatches,omitempty"` // Regex pattern
	MatchAny       bool   `json:"matchAny,omitempty"`      // Match any prompt

	// ToolCalls to make before responding
	ToolCalls []ToolCallSpec `json:"toolCalls,omitempty"`

	// Response to output after tool calls complete
	Response string `json:"response"`

	// Error causes the agent to exit with an error instead of responding
	Error string `json:"error,omitempty"`
}

// ToolCallSpec defines a tool call to make to an MCP server
type ToolCallSpec struct {
	// Server is the MCP server name (optional, uses first server if not set)
	Server string `json:"server,omitempty"`

	// Name is the tool name to call
	Name string `json:"name"`

	// Arguments to pass to the tool
	Arguments map[string]any `json:"arguments,omitempty"`

	// ExpectError if true, the tool call is expected to return an error
	ExpectError bool `json:"expectError,omitempty"`
}

// LoadConfig reads a config from a JSON file
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveConfig writes a config to a JSON file
func SaveConfig(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// NewConfig creates a new empty config
func NewConfig() *Config {
	return &Config{
		Behaviors: make([]Behavior, 0),
	}
}

// WithDefaultResponse sets the default response
func (c *Config) WithDefaultResponse(response string) *Config {
	c.DefaultResponse = response
	return c
}

// WithDefaultError sets the default error
func (c *Config) WithDefaultError(err string) *Config {
	c.DefaultError = err
	return c
}

// WithVerifyMCPConnectivity enables MCP connectivity verification
func (c *Config) WithVerifyMCPConnectivity() *Config {
	c.VerifyMCPConnectivity = true
	return c
}

// WithVerifyAllowedTools sets tools that must be available
func (c *Config) WithVerifyAllowedTools(tools ...string) *Config {
	c.VerifyAllowedTools = tools
	return c
}

// AddBehavior adds a behavior to the config
func (c *Config) AddBehavior(b Behavior) *Config {
	c.Behaviors = append(c.Behaviors, b)
	return c
}

// NewBehavior creates a new behavior builder
func NewBehavior() *Behavior {
	return &Behavior{
		ToolCalls: make([]ToolCallSpec, 0),
	}
}

// WithName sets the behavior name
func (b *Behavior) WithName(name string) *Behavior {
	b.Name = name
	return b
}

// OnPromptContaining sets the prompt contains matcher
func (b *Behavior) OnPromptContaining(substring string) *Behavior {
	b.PromptContains = substring
	return b
}

// OnPromptMatching sets the prompt regex matcher
func (b *Behavior) OnPromptMatching(pattern string) *Behavior {
	b.PromptMatches = pattern
	return b
}

// OnAnyPrompt matches any prompt
func (b *Behavior) OnAnyPrompt() *Behavior {
	b.MatchAny = true
	return b
}

// CallTool adds a tool call to the behavior
func (b *Behavior) CallTool(name string, args map[string]any) *Behavior {
	b.ToolCalls = append(b.ToolCalls, ToolCallSpec{
		Name:      name,
		Arguments: args,
	})
	return b
}

// CallToolOnServer adds a tool call to a specific server
func (b *Behavior) CallToolOnServer(server, name string, args map[string]any) *Behavior {
	b.ToolCalls = append(b.ToolCalls, ToolCallSpec{
		Server:    server,
		Name:      name,
		Arguments: args,
	})
	return b
}

// ThenRespond sets the response
func (b *Behavior) ThenRespond(response string) *Behavior {
	b.Response = response
	return b
}

// ThenFail sets an error response
func (b *Behavior) ThenFail(err string) *Behavior {
	b.Error = err
	return b
}
