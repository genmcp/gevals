package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mcpchecker/mcpchecker/pkg/llmjudge"
	"github.com/mcpchecker/mcpchecker/pkg/util"
)

type LLMJudgeStep struct {
	cfg *llmjudge.LLMJudgeStepConfig
}

var _ StepRunner = &LLMJudgeStep{}

func ParseLLMJudgeStep(raw json.RawMessage) (StepRunner, error) {
	cfg := &llmjudge.LLMJudgeStepConfig{}

	err := json.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}

	return NewLLMJudgeStep(cfg)
}

func NewLLMJudgeStep(cfg *llmjudge.LLMJudgeStepConfig) (*LLMJudgeStep, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &LLMJudgeStep{cfg: cfg}, nil
}

func (s *LLMJudgeStep) Execute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	judge, ok := llmjudge.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no llm judge configured for llmJudge step")
	}

	if input.Agent == nil || input.Agent.Prompt == "" || input.Agent.Output == "" {
		return nil, fmt.Errorf("cannot run llmJudge step before agent (must be in verification)")
	}

	if util.IsVerbose(ctx) {
		fmt.Printf("  → LLM judge '%s' is evaluating…\n", judge.ModelName())
	}

	res, err := judge.EvaluateText(ctx, s.cfg, input.Agent.Prompt, input.Agent.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to call llm judge: %w", err)
	}

	out := &StepOutput{
		Type:    "llmJudge",
		Success: res.Passed,
		Message: res.Reason,
	}

	if !res.Passed {
		out.Error = fmt.Sprintf("llm judge failed for reason '%s': %s", res.FailureCategory, res.Reason)
	}

	return out, nil
}
