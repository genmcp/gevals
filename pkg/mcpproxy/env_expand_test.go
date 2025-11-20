package mcpproxy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		setupEnv    map[string]string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "simple value no expansion",
			value:       "simple-value",
			expected:    "simple-value",
			expectError: false,
		},
		{
			name:        "required var set",
			value:       "${TEST_VAR}",
			setupEnv:    map[string]string{"TEST_VAR": "test-value"},
			expected:    "test-value",
			expectError: false,
		},
		{
			name:        "required var not set",
			value:       "${TEST_VAR}",
			setupEnv:    map[string]string{},
			expectError: true,
			errorMsg:    "required environment variable(s) not set",
		},
		{
			name:        "required var empty",
			value:       "${TEST_VAR}",
			setupEnv:    map[string]string{"TEST_VAR": ""},
			expectError: true,
			errorMsg:    "required environment variable(s) not set",
		},
		{
			name:        "default var set",
			value:       "${TEST_VAR:-default-value}",
			setupEnv:    map[string]string{"TEST_VAR": "actual-value"},
			expected:    "actual-value",
			expectError: false,
		},
		{
			name:        "default var not set",
			value:       "${TEST_VAR:-default-value}",
			setupEnv:    map[string]string{},
			expected:    "default-value",
			expectError: false,
		},
		{
			name:        "default var empty",
			value:       "${TEST_VAR:-default-value}",
			setupEnv:    map[string]string{"TEST_VAR": ""},
			expected:    "default-value",
			expectError: false,
		},
		{
			name:        "default with empty string",
			value:       "${TEST_VAR:-}",
			setupEnv:    map[string]string{},
			expected:    "",
			expectError: false,
		},
		{
			name:        "multiple expansions",
			value:       "${VAR1}-${VAR2:-default}",
			setupEnv:    map[string]string{"VAR1": "value1"},
			expected:    "value1-default",
			expectError: false,
		},
		{
			name:        "mixed required and default",
			value:       "${REQUIRED_VAR}:${OPTIONAL_VAR:-optional}",
			setupEnv:    map[string]string{"REQUIRED_VAR": "required"},
			expected:    "required:optional",
			expectError: false,
		},
		{
			name:        "nested expansion in default",
			value:       "${VAR1:-${VAR2:-final-default}}",
			setupEnv:    map[string]string{},
			expected:    "final-default",
			expectError: false,
		},
		{
			name:        "literal dollar sign",
			value:       "cost: $100",
			expected:    "cost: $100",
			expectError: false,
		},
		{
			name:        "multiple required vars one missing",
			value:       "${VAR1}-${VAR2}",
			setupEnv:    map[string]string{"VAR1": "value1"},
			expectError: true,
			errorMsg:    "required environment variable(s) not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			originalEnv := make(map[string]string)
			for k, v := range tt.setupEnv {
				if orig, ok := os.LookupEnv(k); ok {
					originalEnv[k] = orig
				}
				os.Setenv(k, v)
			}
			defer func() {
				// Restore original env
				for k := range tt.setupEnv {
					if orig, ok := originalEnv[k]; ok {
						os.Setenv(k, orig)
					} else {
						os.Unsetenv(k)
					}
				}
			}()

			result, err := ExpandEnv(tt.value)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

