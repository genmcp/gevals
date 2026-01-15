package steps

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner is a simple StepRunner for testing
type mockRunner struct {
	name string
}

func (m *mockRunner) Execute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	return &StepOutput{Success: true, Message: m.name}, nil
}

func TestRegistry_Register(t *testing.T) {
	tt := map[string]struct {
		registerFirst string
		registerAgain string
		expectErr     bool
	}{
		"register new parser": {
			registerFirst: "newtype",
			expectErr:     false,
		},
		"register duplicate fails": {
			registerFirst: "duptype",
			registerAgain: "duptype",
			expectErr:     true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			reg := &Registry{
				parsers:       make(map[string]Parser),
				prefixParsers: make(map[string]PrefixParser),
			}

			parser := func(raw json.RawMessage) (StepRunner, error) {
				return &mockRunner{name: "test"}, nil
			}

			err := reg.Register(tc.registerFirst, parser)
			require.NoError(t, err)

			if tc.registerAgain != "" {
				err = reg.Register(tc.registerAgain, parser)
				if tc.expectErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "already exists")
				}
			}
		})
	}
}

func TestRegistry_RegisterPrefix(t *testing.T) {
	tt := map[string]struct {
		registerFirst string
		registerAgain string
		expectErr     bool
	}{
		"register new prefix parser": {
			registerFirst: "ext",
			expectErr:     false,
		},
		"register duplicate prefix fails": {
			registerFirst: "ext",
			registerAgain: "ext",
			expectErr:     true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			reg := &Registry{
				parsers:       make(map[string]Parser),
				prefixParsers: make(map[string]PrefixParser),
			}

			parser := func(suffix string, raw json.RawMessage) (StepRunner, error) {
				return &mockRunner{name: suffix}, nil
			}

			err := reg.RegisterPrefix(tc.registerFirst, parser)
			require.NoError(t, err)

			if tc.registerAgain != "" {
				err = reg.RegisterPrefix(tc.registerAgain, parser)
				if tc.expectErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "already exists")
				}
			}
		})
	}
}

func TestRegistry_Parse(t *testing.T) {
	// Create a registry with test parsers
	reg := &Registry{
		parsers:       make(map[string]Parser),
		prefixParsers: make(map[string]PrefixParser),
	}

	// Register a standard parser
	reg.parsers["script"] = func(raw json.RawMessage) (StepRunner, error) {
		return &mockRunner{name: "script-runner"}, nil
	}

	// Register a prefix parser
	reg.prefixParsers["k8s"] = func(suffix string, raw json.RawMessage) (StepRunner, error) {
		return &mockRunner{name: "k8s." + suffix}, nil
	}

	tt := map[string]struct {
		config       StepConfig
		expectedName string
		expectErr    bool
		errMsg       string
	}{
		"parse standard step": {
			config:       StepConfig{"script": json.RawMessage(`{"inline": "echo hello"}`)},
			expectedName: "script-runner",
			expectErr:    false,
		},
		"parse prefix step": {
			config:       StepConfig{"k8s.apply": json.RawMessage(`{"file": "deploy.yaml"}`)},
			expectedName: "k8s.apply",
			expectErr:    false,
		},
		"unknown step type": {
			config:    StepConfig{"unknown": json.RawMessage(`{}`)},
			expectErr: true,
			errMsg:    "unknown step type",
		},
		"unknown prefix": {
			config:    StepConfig{"unknown.operation": json.RawMessage(`{}`)},
			expectErr: true,
			errMsg:    "unknown step type",
		},
		"empty config": {
			config:    StepConfig{},
			expectErr: true,
			errMsg:    "exactly one type",
		},
		"multiple types in config": {
			config: StepConfig{
				"script": json.RawMessage(`{}`),
				"http":   json.RawMessage(`{}`),
			},
			expectErr: true,
			errMsg:    "exactly one type",
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			runner, err := reg.Parse(tc.config)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, runner)

			// Execute to verify it's the right runner
			output, err := runner.Execute(context.Background(), &StepInput{})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedName, output.Message)
		})
	}
}

func TestRegistry_WithExtensions(t *testing.T) {
	// Create a base registry
	baseReg := &Registry{
		parsers:       make(map[string]Parser),
		prefixParsers: make(map[string]PrefixParser),
	}

	baseReg.parsers["script"] = func(raw json.RawMessage) (StepRunner, error) {
		return &mockRunner{name: "script"}, nil
	}

	baseReg.prefixParsers["existing"] = func(suffix string, raw json.RawMessage) (StepRunner, error) {
		return &mockRunner{name: "existing." + suffix}, nil
	}

	tt := map[string]struct {
		aliases              []string
		expectParsersCount   int
		expectPrefixCount    int
		expectBaseUnchanged  bool
	}{
		"add single extension": {
			aliases:             []string{"k8s"},
			expectParsersCount:  1,
			expectPrefixCount:   2, // existing + k8s
			expectBaseUnchanged: true,
		},
		"add multiple extensions": {
			aliases:             []string{"k8s", "db", "git"},
			expectParsersCount:  1,
			expectPrefixCount:   4, // existing + 3 new
			expectBaseUnchanged: true,
		},
		"empty aliases": {
			aliases:             []string{},
			expectParsersCount:  1,
			expectPrefixCount:   1, // just existing
			expectBaseUnchanged: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			// Note: WithExtensions requires a context with ExtensionManager
			// For this test, we just verify the structure is created correctly
			// The actual extension parsing is tested in extension_test.go

			ctx := context.Background()
			newReg := baseReg.WithExtensions(ctx, tc.aliases)

			assert.Len(t, newReg.parsers, tc.expectParsersCount)
			assert.Len(t, newReg.prefixParsers, tc.expectPrefixCount)

			// Verify base registry is unchanged
			if tc.expectBaseUnchanged {
				assert.Len(t, baseReg.parsers, 1)
				assert.Len(t, baseReg.prefixParsers, 1)
			}

			// Verify all aliases are registered as prefix parsers
			for _, alias := range tc.aliases {
				_, exists := newReg.prefixParsers[alias]
				assert.True(t, exists, "alias %q should be registered", alias)
			}
		})
	}
}
