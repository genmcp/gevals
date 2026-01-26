package agent

import (
	"fmt"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

const (
	basePath = "testdata"
)

func TestFromFile(t *testing.T) {
	tt := map[string]struct {
		file      string
		expected  *AgentSpec
		expectErr bool
	}{
		"claude": {
			file: "claude-agent.yaml",
			expected: &AgentSpec{
				TypeMeta: util.TypeMeta{
					Kind: KindAgent,
				},
				Metadata: AgentMetadata{
					Name:    "claude",
					Version: ptr.To("2.0.x"),
				},
				Commands: AgentCommands{
					ArgTemplateMcpServer:    "{{ .File }}",
					ArgTemplateAllowedTools: "mcp__{{ .ServerName }}__{{ .ToolName }}",
					RunPrompt:               "claude --mcp-config {{ .McpServerFileArgs }} --strict-mcp-config --allowedTools {{ .AllowedToolArgs }} --print {{ .Prompt }}",
				},
			},
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			got, err := FromFile(fmt.Sprintf("%s/%s", basePath, tc.file))
			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
