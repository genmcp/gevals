package task

import (
	"context"
	"fmt"

	"github.com/genmcp/gevals/pkg/agent"
	"github.com/genmcp/gevals/pkg/llmjudge"
)

type TaskRunner interface {
	Setup(ctx context.Context) (string, error)
	Cleanup(ctx context.Context) (string, error)
	RunAgent(ctx context.Context, agent agent.Runner) (string, error)
	Verify(ctx context.Context) (string, error)
	GetJudgeResult() (*llmjudge.LLMJudgeResult, error)
}

type taskRunner struct {
	steps        TaskSteps
	judge        llmjudge.LLMJudge
	judgeCfg     *llmjudge.LLMJudgeTaskConfig
	prompt       string
	output       string
	judgeResult  *llmjudge.LLMJudgeResult
	judgeError   error
}

func NewTaskRunner(cfg *TaskSpec, judge llmjudge.LLMJudge) (TaskRunner, error) {
	if cfg.Steps.Prompt.IsEmpty() {
		return nil, fmt.Errorf("prompt.inline or prompt.file must be set on a task to run it")
	}

	// Validate the verify step
	if err := cfg.Steps.VerifyScript.Validate(); err != nil {
		return nil, err
	}

	// If judge is nil and there is a llm judge task config, report an error
	if cfg.Steps.VerifyScript.LLMJudgeTaskConfig != nil && judge == nil {
		return nil, fmt.Errorf("verify.exact and verify.contains require that the eval contains an llm judge config")
	}

	return &taskRunner{
		steps:    cfg.Steps,
		judge:    judge,
		judgeCfg: cfg.Steps.VerifyScript.LLMJudgeTaskConfig,
	}, nil
}

func (r *taskRunner) Setup(ctx context.Context) (string, error) {
	if r.steps.SetupScript.IsEmpty() {
		return "no setup", nil
	}

	return r.steps.SetupScript.Run(ctx)
}

func (r *taskRunner) Cleanup(ctx context.Context) (string, error) {
	if r.steps.CleanupScript.IsEmpty() {
		return "no cleanup", nil
	}

	return r.steps.CleanupScript.Run(ctx)
}

func (r *taskRunner) RunAgent(ctx context.Context, agent agent.Runner) (string, error) {
	prompt, err := r.steps.Prompt.GetValue()
	if err != nil {
		return "", fmt.Errorf("failed to get prompt: %w", err)
	}

	r.prompt = prompt

	result, err := agent.RunTask(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to run agent: %w", err)
	}

	output := result.GetOutput()

	r.output = output

	return output, nil
}

func (r *taskRunner) Verify(ctx context.Context) (string, error) {
	// no need to verify that Verify is set - this is validated in NewTaskRunner
	if r.steps.VerifyScript.Step != nil && !r.steps.VerifyScript.Step.IsEmpty() {
		// Script-based verification
		return r.steps.VerifyScript.Step.Run(ctx)
	}

	// Using LLM judge - validate that state exists
	if r.prompt == "" || r.output == "" {
		return "", fmt.Errorf("cannot run LLM judge verification: RunAgent() must be called before Verify()")
	}

	out, err := r.judge.EvaluateText(ctx, r.judgeCfg, r.prompt, r.output)
	if err != nil {
		// Store error from judge API call
		r.judgeError = err
		return "", err
	}

	// Store judge result (both success and failure cases)
	r.judgeResult = out

	if !out.Passed {
		return "", fmt.Errorf("evaluation failed for reason '%s' because '%s'", out.FailureCategory, out.Reason)
	}

	return "", nil
}

func (r *taskRunner) GetJudgeResult() (*llmjudge.LLMJudgeResult, error) {
	// Return stored judge result and error
	// Returns (nil, nil) when no judge was used (script-based verification)
	return r.judgeResult, r.judgeError
}
