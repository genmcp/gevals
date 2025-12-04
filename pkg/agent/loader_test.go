package agent

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithBuiltins(t *testing.T) {
	tests := map[string]struct {
		file        string
		setupEnv    func()
		cleanupEnv  func()
		expectErr   bool
		errContains string
		validate    func(t *testing.T, spec *AgentSpec)
		shouldSkip  bool
	}{
		"claude-code builtin": {
			file: "builtin-claude-code.yaml",
			validate: func(t *testing.T, spec *AgentSpec) {
				assert.Equal(t, "claude-code", spec.Metadata.Name)
				require.NotNil(t, spec.Commands.UseVirtualHome)
				assert.False(t, *spec.Commands.UseVirtualHome)
				assert.Contains(t, spec.Commands.RunPrompt, "claude")
			},
			shouldSkip: func() bool {
				_, err := exec.LookPath("claude")
				return err != nil
			}(),
		},
		"openai-agent builtin with valid env": {
			file: "builtin-openai-agent.yaml",
			setupEnv: func() {
				os.Setenv("MODEL_BASE_URL", "https://api.openai.com/v1")
				os.Setenv("MODEL_KEY", "test-key")
			},
			cleanupEnv: func() {
				os.Unsetenv("MODEL_BASE_URL")
				os.Unsetenv("MODEL_KEY")
			},
			validate: func(t *testing.T, spec *AgentSpec) {
				assert.Equal(t, "openai-agent-gpt-4", spec.Metadata.Name)
				// Check builtin configuration is present
				require.NotNil(t, spec.Builtin)
				assert.Equal(t, "openai-agent", spec.Builtin.Type)
				assert.Equal(t, "gpt-4", spec.Builtin.Model)
				assert.Equal(t, "https://api.openai.com/v1", spec.Builtin.BaseURL)
				assert.Equal(t, "test-key", spec.Builtin.APIKey)
			},
		},
		"builtin with overrides": {
			file: "builtin-with-overrides.yaml",
			setupEnv: func() {
				os.Setenv("MODEL_BASE_URL", "https://api.openai.com/v1")
				os.Setenv("MODEL_KEY", "test-key")
			},
			cleanupEnv: func() {
				os.Unsetenv("MODEL_BASE_URL")
				os.Unsetenv("MODEL_KEY")
			},
			validate: func(t *testing.T, spec *AgentSpec) {
				// Name should be overridden
				assert.Equal(t, "custom-openai", spec.Metadata.Name)
				// UseVirtualHome should be true as specified in the YAML override
				require.NotNil(t, spec.Commands.UseVirtualHome)
				assert.True(t, *spec.Commands.UseVirtualHome)
				// Builtin configuration should be present
				require.NotNil(t, spec.Builtin)
				assert.Equal(t, "openai-agent", spec.Builtin.Type)
			},
		},
		"non-builtin agent (no builtin field)": {
			file: "claude-agent.yaml",
			validate: func(t *testing.T, spec *AgentSpec) {
				// Should load normally without builtin processing
				assert.Equal(t, "claude", spec.Metadata.Name)
				assert.NotNil(t, spec.Metadata.Version)
				assert.Equal(t, "2.0.x", *spec.Metadata.Version)
			},
		},
		"invalid builtin type": {
			file:        "builtin-invalid-type.yaml",
			expectErr:   true,
			errContains: "unknown builtin type",
		},
		"builtin requires model but not provided": {
			file:        "builtin-openai-no-model.yaml",
			expectErr:   true,
			errContains: "requires a model",
		},
		"builtin with missing environment variables": {
			file: "builtin-openai-agent.yaml",
			setupEnv: func() {
				// Deliberately don't set env vars
				os.Unsetenv("MODEL_BASE_URL")
				os.Unsetenv("MODEL_KEY")
			},
			expectErr:   true,
			errContains: "MODEL_BASE_URL",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.shouldSkip {
				t.Skipf("skipping test %s because shouldSkip=true", name)
			}

			if tc.setupEnv != nil {
				tc.setupEnv()
			}
			if tc.cleanupEnv != nil {
				defer tc.cleanupEnv()
			}

			spec, err := LoadWithBuiltins(basePath + "/" + tc.file)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, spec)

			if tc.validate != nil {
				tc.validate(t, spec)
			}
		})
	}
}

func TestMergeAgentSpecs(t *testing.T) {
	t.Run("override metadata name", func(t *testing.T) {
		base := &AgentSpec{
			Metadata: AgentMetadata{Name: "base"},
		}
		override := &AgentSpec{
			Metadata: AgentMetadata{Name: "override"},
		}
		result := mergeAgentSpecs(base, override)
		assert.Equal(t, "override", result.Metadata.Name)
	})

	t.Run("override commands", func(t *testing.T) {
		baseUseVirtualHome := false
		overrideUseVirtualHome := true
		base := &AgentSpec{
			Commands: AgentCommands{
				UseVirtualHome:       &baseUseVirtualHome,
				ArgTemplateMcpServer: "{{ .File }}",
				RunPrompt:            "base command",
			},
		}
		override := &AgentSpec{
			Commands: AgentCommands{
				UseVirtualHome: &overrideUseVirtualHome,
				RunPrompt:      "override command",
			},
		}
		result := mergeAgentSpecs(base, override)

		// Overridden fields
		require.NotNil(t, result.Commands.UseVirtualHome)
		assert.True(t, *result.Commands.UseVirtualHome)
		assert.Equal(t, "override command", result.Commands.RunPrompt)

		// Non-overridden fields should keep base value
		assert.Equal(t, "{{ .File }}", result.Commands.ArgTemplateMcpServer)
	})

	t.Run("override preserves base when override is empty", func(t *testing.T) {
		base := &AgentSpec{
			Metadata: AgentMetadata{Name: "base"},
			Commands: AgentCommands{
				ArgTemplateMcpServer: "{{ .File }}",
				RunPrompt:            "base command",
			},
		}
		override := &AgentSpec{
			Metadata: AgentMetadata{Name: "override"},
			// Commands not specified
		}
		result := mergeAgentSpecs(base, override)

		assert.Equal(t, "override", result.Metadata.Name)
		assert.Equal(t, "{{ .File }}", result.Commands.ArgTemplateMcpServer)
		assert.Equal(t, "base command", result.Commands.RunPrompt)
	})
}
