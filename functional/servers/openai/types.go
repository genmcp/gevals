package openai

import (
	"encoding/json"
)

// ChatCompletionRequest matches the OpenAI SDK format
type ChatCompletionRequest struct {
	Model      string      `json:"model"`
	Messages   []Message   `json:"messages"`
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`
	Seed       *int64      `json:"seed,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool represents a tool definition in the request
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function tool
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall represents a tool call in a response message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded string
}

// ChatCompletionResponse matches the OpenAI API response format
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage contains token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolChoice can be a string or an object
// String values: "none", "auto", "required"
// Object formats:
//   - Function: {"type": "function", "function": {"name": "function_name"}}
//   - AllowedTools: {"type": "allowed_tools", "mode": "auto"|"required", "tools": [...]}
//   - Custom: {"type": "custom", "custom": {...}}
type ToolChoice struct {
	// If IsString is true, use StringValue
	IsString    bool
	StringValue string // "none", "auto", "required"

	// Object format fields
	Type     string              `json:"type,omitempty"`     // "function", "allowed_tools", "custom"
	Function *ToolChoiceFunction `json:"function,omitempty"` // For type="function"

	// For type="allowed_tools"
	AllowedTools *AllowedToolsChoice `json:"-"` // Handled specially in UnmarshalJSON

	// For type="custom"
	Custom map[string]any `json:"custom,omitempty"`
}

// ToolChoiceFunction specifies a function to force
type ToolChoiceFunction struct {
	Name string `json:"name"`
}

// AllowedToolsChoice constrains available tools
type AllowedToolsChoice struct {
	Mode  string           `json:"mode"`  // "auto" or "required"
	Tools []AllowedToolDef `json:"tools"` // List of allowed tool definitions
}

// AllowedToolDef defines an allowed tool
type AllowedToolDef struct {
	Type     string              `json:"type"`               // "function"
	Function *ToolChoiceFunction `json:"function,omitempty"` // {"name": "..."}
}

// UnmarshalJSON handles both string and object formats for ToolChoice
func (tc *ToolChoice) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		tc.IsString = true
		tc.StringValue = s
		return nil
	}

	// Try object - use intermediate struct to avoid recursion
	type toolChoiceObj struct {
		Type     string              `json:"type"`
		Function *ToolChoiceFunction `json:"function,omitempty"`
		Mode     string              `json:"mode,omitempty"`  // For allowed_tools at top level
		Tools    []AllowedToolDef    `json:"tools,omitempty"` // For allowed_tools at top level
		Custom   map[string]any      `json:"custom,omitempty"`
	}

	var obj toolChoiceObj
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	tc.IsString = false
	tc.Type = obj.Type
	tc.Function = obj.Function
	tc.Custom = obj.Custom

	// Handle allowed_tools - the mode and tools are at the top level alongside type
	if obj.Type == "allowed_tools" {
		tc.AllowedTools = &AllowedToolsChoice{
			Mode:  obj.Mode,
			Tools: obj.Tools,
		}
	}

	return nil
}

// MarshalJSON serializes ToolChoice back to JSON
func (tc ToolChoice) MarshalJSON() ([]byte, error) {
	if tc.IsString {
		return json.Marshal(tc.StringValue)
	}

	// For allowed_tools, we need to flatten the structure
	if tc.Type == "allowed_tools" && tc.AllowedTools != nil {
		return json.Marshal(map[string]any{
			"type":  tc.Type,
			"mode":  tc.AllowedTools.Mode,
			"tools": tc.AllowedTools.Tools,
		})
	}

	// For function type
	if tc.Type == "function" && tc.Function != nil {
		return json.Marshal(map[string]any{
			"type":     tc.Type,
			"function": tc.Function,
		})
	}

	// For custom type
	if tc.Type == "custom" && tc.Custom != nil {
		return json.Marshal(map[string]any{
			"type":   tc.Type,
			"custom": tc.Custom,
		})
	}

	// Fallback - just marshal the type
	return json.Marshal(map[string]any{
		"type": tc.Type,
	})
}

// IsNone returns true if tool_choice is "none"
func (tc *ToolChoice) IsNone() bool {
	return tc.IsString && tc.StringValue == "none"
}

// IsAuto returns true if tool_choice is "auto"
func (tc *ToolChoice) IsAuto() bool {
	return tc.IsString && tc.StringValue == "auto"
}

// IsRequired returns true if tool_choice is "required"
func (tc *ToolChoice) IsRequired() bool {
	return tc.IsString && tc.StringValue == "required"
}

// IsForcedFunction returns true if tool_choice forces a specific function
func (tc *ToolChoice) IsForcedFunction() bool {
	return !tc.IsString && tc.Type == "function" && tc.Function != nil
}

// ForcedFunctionName returns the forced function name, or empty string if not forcing
func (tc *ToolChoice) ForcedFunctionName() string {
	if tc.IsForcedFunction() {
		return tc.Function.Name
	}
	return ""
}

// IsAllowedTools returns true if tool_choice uses allowed_tools mode
func (tc *ToolChoice) IsAllowedTools() bool {
	return !tc.IsString && tc.Type == "allowed_tools" && tc.AllowedTools != nil
}

// AllowedToolNames returns the list of allowed tool names
func (tc *ToolChoice) AllowedToolNames() []string {
	if !tc.IsAllowedTools() {
		return nil
	}
	names := make([]string, 0, len(tc.AllowedTools.Tools))
	for _, tool := range tc.AllowedTools.Tools {
		if tool.Function != nil {
			names = append(names, tool.Function.Name)
		}
	}
	return names
}

// Helper functions for creating ToolChoice values

// ToolChoiceNone creates a "none" tool choice
func ToolChoiceNone() *ToolChoice {
	return &ToolChoice{IsString: true, StringValue: "none"}
}

// ToolChoiceAuto creates an "auto" tool choice
func ToolChoiceAuto() *ToolChoice {
	return &ToolChoice{IsString: true, StringValue: "auto"}
}

// ToolChoiceRequired creates a "required" tool choice
func ToolChoiceRequiredString() *ToolChoice {
	return &ToolChoice{IsString: true, StringValue: "required"}
}

// ToolChoiceForceFunction creates a tool choice that forces a specific function
func ToolChoiceForceFunction(name string) *ToolChoice {
	return &ToolChoice{
		IsString: false,
		Type:     "function",
		Function: &ToolChoiceFunction{Name: name},
	}
}

// ToolChoiceAllowed creates a tool choice with allowed tools
func ToolChoiceAllowed(mode string, toolNames ...string) *ToolChoice {
	tools := make([]AllowedToolDef, len(toolNames))
	for i, name := range toolNames {
		tools[i] = AllowedToolDef{
			Type:     "function",
			Function: &ToolChoiceFunction{Name: name},
		}
	}
	return &ToolChoice{
		IsString: false,
		Type:     "allowed_tools",
		AllowedTools: &AllowedToolsChoice{
			Mode:  mode,
			Tools: tools,
		},
	}
}
