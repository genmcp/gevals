package agent

var builtinTypes = map[string]BuiltinAgent{
	"openai-agent": &OpenAIAgent{},
	"claude-code":  &ClaudeCodeAgent{},
	"gemini-cli":   &GeminiCLIAgent{},
}

// GetBuiltinType retrieves a builtin agent by name
func GetBuiltinType(name string) (BuiltinAgent, bool) {
	agent, ok := builtinTypes[name]
	return agent, ok
}

// ListBuiltinTypes returns all available builtin agent types
func ListBuiltinTypes() []BuiltinAgent {
	result := make([]BuiltinAgent, 0, len(builtinTypes))
	for _, agent := range builtinTypes {
		result = append(result, agent)
	}
	return result
}
