package steps

import (
	"context"
	"fmt"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/llmjudge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLLMJudge struct {
	result *llmjudge.LLMJudgeResult
	err    error
	model  string
}

func (f *fakeLLMJudge) EvaluateText(ctx context.Context, judgeConfig *llmjudge.LLMJudgeStepConfig, prompt, output string) (*llmjudge.LLMJudgeResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeLLMJudge) ModelName() string {
	return f.model
}

func TestLLMJudgeStepConfig_Validate(t *testing.T) {
	tt := map[string]struct {
		config    *llmjudge.LLMJudgeStepConfig
		expectErr bool
	}{
		"valid contains config": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "expected content",
			},
			expectErr: false,
		},
		"valid exact config": {
			config: &llmjudge.LLMJudgeStepConfig{
				Exact: "exact match",
			},
			expectErr: false,
		},
		"invalid: both contains and exact set": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "content",
				Exact:    "exact",
			},
			expectErr: true,
		},
		"invalid: neither contains nor exact set": {
			config:    &llmjudge.LLMJudgeStepConfig{},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestNewLLMJudgeStep(t *testing.T) {
	tt := map[string]struct {
		config    *llmjudge.LLMJudgeStepConfig
		expectErr bool
	}{
		"valid config": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "expected",
			},
			expectErr: false,
		},
		"invalid config": {
			config:    &llmjudge.LLMJudgeStepConfig{},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			step, err := NewLLMJudgeStep(tc.config)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, step)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, step)
		})
	}
}

func TestLLMJudgeStep_Execute(t *testing.T) {
	tt := map[string]struct {
		config    *llmjudge.LLMJudgeStepConfig
		judge     *fakeLLMJudge
		input     *StepInput
		expected  *StepOutput
		expectErr bool
	}{
		"judge passes": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "expected content",
			},
			judge: &fakeLLMJudge{
				model: "test-model",
				result: &llmjudge.LLMJudgeResult{
					Passed:          true,
					Reason:          "output contains expected content",
					FailureCategory: "n/a",
				},
			},
			input: &StepInput{
				Agent: &AgentContext{
					Prompt: "test prompt",
					Output: "test output with expected content",
				},
			},
			expected: &StepOutput{
				Type:    "llmJudge",
				Success: true,
				Message: "output contains expected content",
			},
			expectErr: false,
		},
		"judge fails": {
			config: &llmjudge.LLMJudgeStepConfig{
				Exact: "exact match",
			},
			judge: &fakeLLMJudge{
				model: "test-model",
				result: &llmjudge.LLMJudgeResult{
					Passed:          false,
					Reason:          "output does not match exactly",
					FailureCategory: "semantic_mismatch",
				},
			},
			input: &StepInput{
				Agent: &AgentContext{
					Prompt: "test prompt",
					Output: "different output",
				},
			},
			expected: &StepOutput{
				Type:    "llmJudge",
				Success: false,
				Message: "output does not match exactly",
				Error:   "llm judge failed for reason 'semantic_mismatch': output does not match exactly",
			},
			expectErr: false,
		},
		"judge returns error": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "content",
			},
			judge: &fakeLLMJudge{
				model: "test-model",
				err:   fmt.Errorf("API error"),
			},
			input: &StepInput{
				Agent: &AgentContext{
					Prompt: "test prompt",
					Output: "test output",
				},
			},
			expectErr: true,
		},
		"no judge in context": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "content",
			},
			judge: nil,
			input: &StepInput{
				Agent: &AgentContext{
					Prompt: "test prompt",
					Output: "test output",
				},
			},
			expectErr: true,
		},
		"no agent output": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "content",
			},
			judge: &fakeLLMJudge{
				model: "test-model",
			},
			input:     &StepInput{},
			expectErr: true,
		},
		"agent output missing prompt": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "content",
			},
			judge: &fakeLLMJudge{
				model: "test-model",
			},
			input: &StepInput{
				Agent: &AgentContext{
					Output: "output only",
				},
			},
			expectErr: true,
		},
		"agent output missing output": {
			config: &llmjudge.LLMJudgeStepConfig{
				Contains: "content",
			},
			judge: &fakeLLMJudge{
				model: "test-model",
			},
			input: &StepInput{
				Agent: &AgentContext{
					Prompt: "prompt only",
				},
			},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			step, err := NewLLMJudgeStep(tc.config)
			require.NoError(t, err)

			ctx := context.Background()
			if tc.judge != nil {
				ctx = llmjudge.WithJudge(ctx, tc.judge)
			}

			got, err := step.Execute(ctx, tc.input)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
