package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeModelName(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"simple name": {
			input:    "gpt-4",
			expected: "GPT_4",
		},
		"name with version": {
			input:    "gemini-2.5-pro",
			expected: "GEMINI_2_5_PRO",
		},
		"name with dots": {
			input:    "granite-3.3-8b-instruct",
			expected: "GRANITE_3_3_8B_INSTRUCT",
		},
		"already uppercase": {
			input:    "GPT-4",
			expected: "GPT_4",
		},
		"mixed case": {
			input:    "GpT-4-TuRbO",
			expected: "GPT_4_TURBO",
		},
		"multiple dots and dashes": {
			input:    "model-1.2.3-alpha",
			expected: "MODEL_1_2_3_ALPHA",
		},
		"no special chars": {
			input:    "simplemodel",
			expected: "SIMPLEMODEL",
		},
		"empty string": {
			input:    "",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := normalizeModelName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
