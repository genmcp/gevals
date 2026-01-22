package agent

import (
	"fmt"
	"os"

	"github.com/genmcp/gevals/pkg/util"
	"sigs.k8s.io/yaml"
)

const (
	KindAgent = "Agent"
)

type AgentSpec struct {
	util.TypeMeta `json:",inline"`
	Metadata      AgentMetadata `json:"metadata"`
	Builtin       *BuiltinRef   `json:"builtin,omitempty"`
	Commands      AgentCommands `json:"commands"`
}

// BuiltinRef references a built-in agent type with optional model
type BuiltinRef struct {
	// Type is the built-in agent type (e.g., "openai-agent", "claude-code")
	Type string `json:"type"`

	// Model is the AI model to use (required for some types like openai-agent)
	Model string `json:"model,omitempty"`

	// BaseURL overrides the default API base URL
	BaseURL string `json:"baseUrl,omitempty"`

	// APIKey overrides the default API key (from environment)
	APIKey string `json:"apiKey,omitempty"`
}

type AgentMetadata struct {
	// Name of the agent
	Name string `json:"name"`

	// Version of the agent - used if Commands.GetVersion is not set
	Version *string `json:"version,omitempty"`
}

type AgentCommands struct {
	// Whether or not to create a virtual $HOME for executing the agent without existing config
	UseVirtualHome *bool `json:"useVirtualHome,omitempty"`

	// A template for how the mcp servers config files should be provided to the prompt
	// the server file will be in {{ .File }}
	// the server URL will be in {{ .URL }}
	ArgTemplateMcpServer string `json:"argTemplateMcpServer"`

	// A template for how the mcp agents allowed tools should be provided to the prompt
	// the server name will be in {{ .ServerName }}
	// the tool name will be in {{ .ToolName }}
	ArgTemplateAllowedTools string `json:"argTemplateAllowedTools"`

	// The separator to use when joining allowed tools together
	// Defaults to " " (space) if not specified
	AllowedToolsJoinSeparator *string `json:"allowedToolsJoinSeparator,omitempty"`

	// A template command to run the agent with a prompt and some mcp servers
	// the prompt will be in {{ .Prompt }}
	// the servers will be in {{ .McpServerFileArgs }}
	// the allowed tools will be in {{ .AllowedToolArgs }}
	// {{ .DangerouslySkipPermissions }} will be true if the flag should be included
	RunPrompt string `json:"runPrompt"`

	// Whether to skip permission prompts (unsafe, use only for automated testing)
	// Defaults to false. When true, agents may include flags like --dangerously-skip-permissions
	DangerouslySkipPermissions *bool `json:"dangerouslySkipPermissions,omitempty"`

	// An optional command to get the version of the agent
	// useful for generic agents such as claude code that may autoupdate/have different versions on different machines
	GetVersion *string `json:"getVersion,omitempty"`
}

func Read(data []byte) (*AgentSpec, error) {
	spec := &AgentSpec{}

	err := yaml.Unmarshal(data, spec)
	if err != nil {
		return nil, err
	}

	if err := spec.TypeMeta.Validate(KindAgent); err != nil {
		return nil, err
	}

	return spec, nil
}

func FromFile(path string) (*AgentSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s' for agentspec: %w", path, err)
	}

	return Read(data)
}
