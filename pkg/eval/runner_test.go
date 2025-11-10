package eval

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAgentSpec(t *testing.T) {
	tests := map[string]struct {
		setupEnv    func()
		cleanupEnv  func()
		spec        *EvalSpec
		expectErr   bool
		errContains string
		validate    func(t *testing.T, runner *evalRunner)
	}{
		"inline agent - claude-code": {
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "claude-code",
					},
				},
			},
			validate: func(t *testing.T, runner *evalRunner) {
				agentSpec, err := runner.loadAgentSpec()
				require.NoError(t, err)
				require.NotNil(t, agentSpec)
				assert.Equal(t, "claude-code", agentSpec.Metadata.Name)
			},
		},
		"inline agent - openai-agent with valid env": {
			setupEnv: func() {
				os.Setenv("MODEL_BASE_URL", "https://api.openai.com/v1")
				os.Setenv("MODEL_KEY", "test-key")
			},
			cleanupEnv: func() {
				os.Unsetenv("MODEL_BASE_URL")
				os.Unsetenv("MODEL_KEY")
			},
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type:  "openai-agent",
						Model: "gpt-4",
					},
				},
			},
			validate: func(t *testing.T, runner *evalRunner) {
				agentSpec, err := runner.loadAgentSpec()
				require.NoError(t, err)
				require.NotNil(t, agentSpec)
				assert.Equal(t, "openai-agent-gpt-4", agentSpec.Metadata.Name)
				require.NotNil(t, agentSpec.Builtin)
				assert.Equal(t, "openai-agent", agentSpec.Builtin.Type)
				assert.Equal(t, "gpt-4", agentSpec.Builtin.Model)
			},
		},
		"inline agent - openai-agent without model": {
			setupEnv: func() {
				os.Setenv("MODEL_BASE_URL", "https://api.openai.com/v1")
				os.Setenv("MODEL_KEY", "test-key")
			},
			cleanupEnv: func() {
				os.Unsetenv("MODEL_BASE_URL")
				os.Unsetenv("MODEL_KEY")
			},
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "openai-agent",
					},
				},
			},
			expectErr:   true,
			errContains: "model is required",
		},
		"inline agent - unknown type": {
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "unknown-agent",
					},
				},
			},
			expectErr:   true,
			errContains: "unknown builtin agent type",
		},
		"no agent configuration": {
			spec: &EvalSpec{
				Config: EvalConfig{},
			},
			expectErr:   true,
			errContains: "either agentFile or agent must be specified",
		},
		"both agent and agentFile specified": {
			spec: &EvalSpec{
				Config: EvalConfig{
					AgentFile: "some/path/agent.yaml",
					Agent: &AgentRef{
						Type: "claude-code",
					},
				},
			},
			expectErr:   true,
			errContains: "cannot specify both agentFile and agent",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setupEnv != nil {
				tc.setupEnv()
			}
			if tc.cleanupEnv != nil {
				defer tc.cleanupEnv()
			}

			runner := &evalRunner{
				spec: tc.spec,
			}

			if tc.expectErr {
				_, err := runner.loadAgentSpec()
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			if tc.validate != nil {
				tc.validate(t, runner)
			}
		})
	}
}
