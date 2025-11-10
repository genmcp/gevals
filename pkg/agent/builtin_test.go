package agent

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBuiltinType(t *testing.T) {
	tests := map[string]struct {
		agentType    string
		shouldExist  bool
		expectedName string
	}{
		"openai-agent exists": {
			agentType:    "openai-agent",
			shouldExist:  true,
			expectedName: "openai-agent",
		},
		"claude-code exists": {
			agentType:    "claude-code",
			shouldExist:  true,
			expectedName: "claude-code",
		},
		"non-existent agent": {
			agentType:   "non-existent",
			shouldExist: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			agent, ok := GetBuiltinType(tc.agentType)
			if tc.shouldExist {
				require.True(t, ok)
				require.NotNil(t, agent)
				assert.Equal(t, tc.expectedName, agent.Name())
			} else {
				assert.False(t, ok)
				assert.Nil(t, agent)
			}
		})
	}
}

func TestListBuiltinTypes(t *testing.T) {
	agents := ListBuiltinTypes()

	// Should have at least 2 builtin agents
	assert.GreaterOrEqual(t, len(agents), 2)

	// Check that expected agents are present
	expectedAgents := map[string]bool{
		"openai-agent": false,
		"claude-code":  false,
	}

	for _, agent := range agents {
		if _, ok := expectedAgents[agent.Name()]; ok {
			expectedAgents[agent.Name()] = true
		}
	}

	for name, found := range expectedAgents {
		assert.True(t, found, "Expected builtin agent %s not found", name)
	}
}

func TestOpenAIAgent(t *testing.T) {
	agent := &OpenAIAgent{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "openai-agent", agent.Name())
	})

	t.Run("Description", func(t *testing.T) {
		desc := agent.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "OpenAI")
	})

	t.Run("RequiresModel", func(t *testing.T) {
		assert.True(t, agent.RequiresModel())
	})

	t.Run("ValidateEnvironment", func(t *testing.T) {
		// Should always succeed - no external binary required
		err := agent.ValidateEnvironment()
		assert.NoError(t, err)
	})

	t.Run("GetDefaults requires model", func(t *testing.T) {
		spec, err := agent.GetDefaults("")
		assert.Error(t, err)
		assert.Nil(t, spec)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("GetDefaults requires environment variables", func(t *testing.T) {
		// Clear any existing env vars
		oldBaseURL := os.Getenv("MODEL_BASE_URL")
		oldAPIKey := os.Getenv("MODEL_KEY")
		defer func() {
			if oldBaseURL != "" {
				os.Setenv("MODEL_BASE_URL", oldBaseURL)
			}
			if oldAPIKey != "" {
				os.Setenv("MODEL_KEY", oldAPIKey)
			}
		}()
		os.Unsetenv("MODEL_BASE_URL")
		os.Unsetenv("MODEL_KEY")

		spec, err := agent.GetDefaults("gpt-4")
		assert.Error(t, err)
		assert.Nil(t, spec)
		assert.Contains(t, err.Error(), "MODEL_BASE_URL")
		assert.Contains(t, err.Error(), "MODEL_KEY")
	})

	t.Run("GetDefaults with valid environment", func(t *testing.T) {
		// Set up environment variables
		os.Setenv("MODEL_BASE_URL", "https://api.openai.com/v1")
		os.Setenv("MODEL_KEY", "test-key")
		defer func() {
			os.Unsetenv("MODEL_BASE_URL")
			os.Unsetenv("MODEL_KEY")
		}()

		spec, err := agent.GetDefaults("gpt-4")
		require.NoError(t, err)
		require.NotNil(t, spec)

		// Check metadata
		assert.Equal(t, "openai-agent-gpt-4", spec.Metadata.Name)

		// Check builtin configuration is stored
		require.NotNil(t, spec.Builtin)
		assert.Equal(t, "openai-agent", spec.Builtin.Type)
		assert.Equal(t, "gpt-4", spec.Builtin.Model)
		assert.Equal(t, "https://api.openai.com/v1", spec.Builtin.BaseURL)
		assert.Equal(t, "test-key", spec.Builtin.APIKey)

		// Check commands
		assert.False(t, spec.Commands.UseVirtualHome)
		assert.Equal(t, "{{ .URL }}", spec.Commands.ArgTemplateMcpServer)
		// RunPrompt is empty for OpenAI agents - they use a custom runner
		assert.Empty(t, spec.Commands.RunPrompt)
	})
}

func TestClaudeCodeAgent(t *testing.T) {
	agent := &ClaudeCodeAgent{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "claude-code", agent.Name())
	})

	t.Run("Description", func(t *testing.T) {
		desc := agent.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "Claude")
	})

	t.Run("RequiresModel", func(t *testing.T) {
		assert.False(t, agent.RequiresModel())
	})

	t.Run("GetDefaults without model", func(t *testing.T) {
		spec, err := agent.GetDefaults("")
		require.NoError(t, err)
		require.NotNil(t, spec)

		// Check metadata
		assert.Equal(t, "claude-code", spec.Metadata.Name)

		// Check commands
		assert.False(t, spec.Commands.UseVirtualHome)
		assert.Equal(t, "--mcp-config {{ .File }}", spec.Commands.ArgTemplateMcpServer)
		assert.Equal(t, "mcp__{{ .ServerName }}__{{ .ToolName }}", spec.Commands.ArgTemplateAllowedTools)
		assert.Contains(t, spec.Commands.RunPrompt, "claude")
		assert.Contains(t, spec.Commands.RunPrompt, "{{ .McpServerFileArgs }}")
		assert.Contains(t, spec.Commands.RunPrompt, "{{ .AllowedToolArgs }}")
		assert.Contains(t, spec.Commands.RunPrompt, "{{ .Prompt }}")
	})

	t.Run("GetDefaults with model (ignored)", func(t *testing.T) {
		spec, err := agent.GetDefaults("some-model")
		require.NoError(t, err)
		require.NotNil(t, spec)

		// Model should be ignored for Claude Code
		assert.Equal(t, "claude-code", spec.Metadata.Name)
	})
}
