package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/genmcp/gevals/pkg/util"
	"sigs.k8s.io/yaml"
)

const (
	KindAgent = "Agent"
)

type AgentSpec struct {
	Metadata AgentMetadata `json:"metadata"`
	Commands AgentCommands `json:"commands"`
}

type AgentMetadata struct {
	// Name of the agent
	Name string `json:"name"`

	// Version of the agent - used if Commads.GetVersion is not set
	Version *string `json:"version,omitempty"`
}

type AgentCommands struct {
	// Whether or not to create a virtual $HOME for executing the agent without existing config
	UseVirtualHome bool `json:"useVirtualHome"`

	// A template for how the mcp servers config files should be provided to the prompt
	// the server file will be in {{ .File }}
	ArgTemplateMcpServer string `json:"argTemplateMcpServer"`

	// A template for how the mcp agents allowed tools should be provided to the prompt
	// the server name will be in {{ .ServerName }}
	// the tool name will be in {{ .ToolName }}
	ArgTemplateAllowedTools string `json:"argTemplateAllowedTools"`

	// A template command to run the agent with a prompt and some mcp servers
	// the prompt will be in {{ .Prompt }}
	// the servers will be in {{ .McpServerFileArgs }}
	// the allowed tools will be in {{ .AllowedTools }}
	RunPrompt string `json:"runPrompt"`

	// An optional command to get the version of the agent
	// useful for generic agents such as claude code that may autoupdate/have different versions on different machines
	GetVersion *string `json:"getVersion,omitempty"`
}

func (a *AgentSpec) UnmarshalJSON(data []byte) error {
	return util.UnmarshalWithKind(data, a, KindAgent)
}

func Read(data []byte) (*AgentSpec, error) {
	spec := &AgentSpec{}

	err := yaml.Unmarshal(data, spec)
	if err != nil {
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
