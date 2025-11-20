package agent

import (
	"fmt"
	"os/exec"
)

type ClaudeCodeAgent struct{}

func (a *ClaudeCodeAgent) Name() string {
	return "claude-code"
}

func (a *ClaudeCodeAgent) Description() string {
	return "Anthropic's Claude Code CLI"
}

func (a *ClaudeCodeAgent) RequiresModel() bool {
	return false // Claude Code manages its own model selection
}

func (a *ClaudeCodeAgent) ValidateEnvironment() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("'claude' binary not found in PATH")
	}
	return nil
}

func (a *ClaudeCodeAgent) GetDefaults(model string) (*AgentSpec, error) {
	separator := ","
	useVirtualHome := false
	return &AgentSpec{
		Metadata: AgentMetadata{
			Name: "claude-code",
		},
		Commands: AgentCommands{
			UseVirtualHome:            &useVirtualHome,
			ArgTemplateMcpServer:      "--mcp-config {{ .File }}",
			ArgTemplateAllowedTools:   "mcp__{{ .ServerName }}__{{ .ToolName }}",
			AllowedToolsJoinSeparator: &separator,
			RunPrompt:                 `claude {{ .McpServerFileArgs }} --strict-mcp-config --allowedTools "{{ .AllowedToolArgs }}" --print "{{ .Prompt }}"`,
		},
	}, nil
}
