package task

import (
	"context"
	"fmt"

	"github.com/genmcp/gevals/pkg/agent"
)

type TaskRunner interface {
	Setup(ctx context.Context) (string, error)
	Cleanup(ctx context.Context) (string, error)
	RunAgent(ctx context.Context, agent agent.Runner) (string, error)
	Verify(ctx context.Context) (string, error)
}

type taskRunner struct {
	steps TaskSteps
}

func NewTaskRunner(cfg *TaskSpec) (TaskRunner, error) {
	if cfg.Steps.Prompt.IsEmpty() {
		return nil, fmt.Errorf("prompt.inline or prompt.file must be set on a task to run it")
	}
	if cfg.Steps.VerifyScript.IsEmpty() {
		return nil, fmt.Errorf("verify.inline or verify.file must be set on a task to run it")
	}

	return &taskRunner{
		steps: cfg.Steps,
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

	result, err := agent.RunTask(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to run agent: %w", err)
	}

	return result.GetOutput(), nil
}

func (r *taskRunner) Verify(ctx context.Context) (string, error) {
	// no need to verify that Verify is set - this is validated in NewTaskRunner
	return r.steps.VerifyScript.Run(ctx)
}
