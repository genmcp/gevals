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
		"inline agent - builtin.claude-code": {
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "builtin.claude-code",
					},
				},
			},
			validate: func(t *testing.T, runner *evalRunner) {
				agentSpec, err := runner.loadAgentSpec()
				// Note: This may fail with environment validation error if claude binary is not in PATH
				// That's expected behavior - the test will skip validation if claude is not available
				if err != nil {
					if assert.Contains(t, err.Error(), "environment validation failed") {
						t.Skip("claude binary not in PATH, skipping test")
					}
					require.NoError(t, err) // Fail if it's a different error
				}
				require.NotNil(t, agentSpec)
				assert.Equal(t, "claude-code", agentSpec.Metadata.Name)
			},
		},
		"inline agent - builtin.openai-agent with valid env": {
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
						Type:  "builtin.openai-agent",
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
		"inline agent - builtin.openai-agent without model": {
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
						Type: "builtin.openai-agent",
					},
				},
			},
			expectErr:   true,
			errContains: "requires a model to be specified",
		},
		"inline agent - unknown type": {
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "builtin.unknown-agent",
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
			errContains: "agent must be specified",
		},
		"file agent without path": {
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "file",
					},
				},
			},
			expectErr:   true,
			errContains: "path must be specified when agent type is 'file'",
		},
		"invalid agent type format": {
			spec: &EvalSpec{
				Config: EvalConfig{
					Agent: &AgentRef{
						Type: "invalid-format",
					},
				},
			},
			expectErr:   true,
			errContains: "agent type must be either 'file' or 'builtin.X' format",
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
func TestMatchesLabelSelector(t *testing.T) {
	tests := map[string]struct {
		taskLabels map[string]string
		selector   map[string]string
		expected   bool
	}{
		"empty selector matches any labels": {
			taskLabels: map[string]string{"suite": "kubernetes"},
			selector:   map[string]string{},
			expected:   true,
		},
		"nil selector matches any labels": {
			taskLabels: map[string]string{"suite": "kubernetes"},
			selector:   nil,
			expected:   true,
		},
		"exact match": {
			taskLabels: map[string]string{"suite": "kubernetes"},
			selector:   map[string]string{"suite": "kubernetes"},
			expected:   true,
		},
		"multiple labels all match": {
			taskLabels: map[string]string{
				"suite":    "kiali",
				"requires": "istio",
			},
			selector: map[string]string{
				"suite":    "kiali",
				"requires": "istio",
			},
			expected: true,
		},
		"selector has subset of task labels": {
			taskLabels: map[string]string{
				"suite":    "kubernetes",
				"category": "basic",
				"requires": "cluster",
			},
			selector: map[string]string{
				"suite": "kubernetes",
			},
			expected: true,
		},
		"task has subset of selector labels - no match": {
			taskLabels: map[string]string{
				"suite": "kubernetes",
			},
			selector: map[string]string{
				"suite":    "kubernetes",
				"requires": "istio",
			},
			expected: false,
		},
		"value mismatch": {
			taskLabels: map[string]string{"suite": "kubernetes"},
			selector:   map[string]string{"suite": "kiali"},
			expected:   false,
		},
		"key not present in task": {
			taskLabels: map[string]string{"suite": "kubernetes"},
			selector:   map[string]string{"category": "basic"},
			expected:   false,
		},
		"empty task labels with non-empty selector": {
			taskLabels: map[string]string{},
			selector:   map[string]string{"suite": "kubernetes"},
			expected:   false,
		},
		"nil task labels with non-empty selector": {
			taskLabels: nil,
			selector:   map[string]string{"suite": "kubernetes"},
			expected:   false,
		},
		"both empty - should match": {
			taskLabels: map[string]string{},
			selector:   map[string]string{},
			expected:   true,
		},
		"both nil - should match": {
			taskLabels: nil,
			selector:   nil,
			expected:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := matchesLabelSelector(tc.taskLabels, tc.selector)
			assert.Equal(t, tc.expected, result)
		})
	}
}
