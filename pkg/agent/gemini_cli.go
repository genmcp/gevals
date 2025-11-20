package agent

import (
	"fmt"
	"os/exec"
)

type GeminiCLIAgent struct{}

func (a *GeminiCLIAgent) Name() string {
	return "gemini-cli"
}

func (a *GeminiCLIAgent) Description() string {
	return "Google's Gemini CLI"
}

func (a *GeminiCLIAgent) RequiresModel() bool {
	return false // Gemini CLI manages its own model selection
}

func (a *GeminiCLIAgent) ValidateEnvironment() error {
	if _, err := exec.LookPath("gemini"); err != nil {
		return fmt.Errorf("'gemini' binary not found in PATH")
	}
	return nil
}

func (a *GeminiCLIAgent) GetDefaults(model string) (*AgentSpec, error) {
	separator := ","
	useVirtualHome := false
	return &AgentSpec{
		Metadata: AgentMetadata{
			Name: "gemini-cli",
		},
		Commands: AgentCommands{
			UseVirtualHome:            &useVirtualHome,
			ArgTemplateMcpServer:      "{{ .URL }}",
			ArgTemplateAllowedTools:   "{{ .ToolName }}",
			AllowedToolsJoinSeparator: &separator,
			RunPrompt:                 `sh -c 'SERVER_NAME="mcp-eval-$$" && gemini mcp add "$SERVER_NAME" {{ .McpServerFileArgs }} --scope project --transport http --trust >/dev/null 2>&1 && trap "gemini mcp remove \"$SERVER_NAME\" >/dev/null 2>&1 || true" EXIT && gemini --allowed-mcp-server-names "$SERVER_NAME" --allowed-tools "{{ .AllowedToolArgs }}" --approval-mode yolo --output-format text --prompt "{{ .Prompt }}"'`,
		},
	}, nil
}
