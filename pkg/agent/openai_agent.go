package agent

import (
	"fmt"
	"os"
)

type OpenAIAgent struct{}

func (a *OpenAIAgent) Name() string {
	return "openai-agent"
}

func (a *OpenAIAgent) Description() string {
	return "OpenAI-compatible agent using direct API calls"
}

func (a *OpenAIAgent) RequiresModel() bool {
	return true
}

func (a *OpenAIAgent) ValidateEnvironment() error {
	// No external binary required - we use the openaiagent package directly
	return nil
}

func (a *OpenAIAgent) GetDefaults(model string) (*AgentSpec, error) {
	if model == "" {
		return nil, fmt.Errorf("model is required for openai-agent")
	}

	// Get API configuration from environment using generic MODEL_ variables
	baseURL := os.Getenv("MODEL_BASE_URL")
	apiKey := os.Getenv("MODEL_KEY")

	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("environment variables MODEL_BASE_URL and MODEL_KEY must be set")
	}

	useVirtualHome := false
	return &AgentSpec{
		Metadata: AgentMetadata{
			Name: fmt.Sprintf("openai-agent-%s", model),
		},
		// Store the OpenAI configuration in the spec
		// The runner will be created specially for OpenAI agents
		Builtin: &BuiltinRef{
			Type:    "openai-agent",
			Model:   model,
			BaseURL: baseURL,
			APIKey:  apiKey,
		},
		Commands: AgentCommands{
			UseVirtualHome:       &useVirtualHome,
			ArgTemplateMcpServer: "{{ .URL }}",
			// RunPrompt is not used for OpenAI agents - they use a custom runner
			RunPrompt: "",
		},
	}, nil
}
