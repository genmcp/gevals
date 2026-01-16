package agent

import (
	"fmt"
	"os"
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
	// Check for GCP credentials (for Vertex AI users)
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		if _, err := exec.LookPath("gcloud"); err == nil {
			// gcloud exists, check if ADC is configured
			cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: No GCP credentials found. If using Vertex AI, run 'gcloud auth application-default login'\n")
			}
		}
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
			RunPrompt:                 `claude {{ .McpServerFileArgs }} --strict-mcp-config --allowedTools "{{ .AllowedToolArgs }}" -p "{{ .Prompt }}" --dangerously-skip-permissions --output-format stream-json --verbose`,
		},
	}, nil
}
