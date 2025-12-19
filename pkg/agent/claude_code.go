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
	dangerouslySkipPermissions := false
	return &AgentSpec{
		Metadata: AgentMetadata{
			Name: "claude-code",
		},
		Commands: AgentCommands{
			UseVirtualHome:             &useVirtualHome,
			ArgTemplateMcpServer:       "--mcp-config {{ .File }}",
			ArgTemplateAllowedTools:    "mcp__{{ .ServerName }}__{{ .ToolName }}",
			AllowedToolsJoinSeparator:  &separator,
			DangerouslySkipPermissions: &dangerouslySkipPermissions,
			RunPrompt:                  `claude {{ .McpServerFileArgs }} --strict-mcp-config --allowedTools "{{ .AllowedToolArgs }}" -p "{{ .Prompt }}"{{ if .DangerouslySkipPermissions }} --dangerously-skip-permissions{{ end }} --output-format stream-json --verbose`,
		},
	}, nil
}
