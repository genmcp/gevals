package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/genmcp/gevals/pkg/agent"
	"github.com/genmcp/gevals/pkg/llmjudge"
	"github.com/genmcp/gevals/pkg/steps"
	"github.com/genmcp/gevals/pkg/util"
)

type TaskRunner interface {
	Setup(ctx context.Context) (string, error)
	Cleanup(ctx context.Context) (string, error)
	RunAgent(ctx context.Context, agent agent.Runner) (string, error)
	Verify(ctx context.Context) (string, error)
}

type taskRunner struct {
	setup   []steps.StepRunner
	verify  []steps.StepRunner
	cleanup []steps.StepRunner
	prompt  string
	output  string
	baseDir string
}

func NewTaskRunner(cfg *TaskConfig, judge llmjudge.LLMJudge) (TaskRunner, error) {
	if cfg.Spec.Prompt.IsEmpty() {
		return nil, fmt.Errorf("prompt.inline or prompt.file must be set on a task to run it")
	}

	var err error
	r := &taskRunner{
		setup:   make([]steps.StepRunner, len(cfg.Spec.Setup)),
		verify:  make([]steps.StepRunner, len(cfg.Spec.Verify)),
		cleanup: make([]steps.StepRunner, len(cfg.Spec.Cleanup)),
		baseDir: cfg.basePath,
	}

	for i, stepCfg := range cfg.Spec.Setup {
		var stepErr error
		r.setup[i], stepErr = steps.DefaultRegistry.Parse(stepCfg)
		if stepErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to parse setup[%d]: %w", i, stepErr))
		}
	}

	for i, stepCfg := range cfg.Spec.Verify {
		var stepErr error
		r.setup[i], stepErr = steps.DefaultRegistry.Parse(stepCfg)
		if stepErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to parse verify[%d]: %w", i, stepErr))
		}
	}

	for i, stepCfg := range cfg.Spec.Cleanup {
		var stepErr error
		r.setup[i], stepErr = steps.DefaultRegistry.Parse(stepCfg)
		if stepErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to parse cleanup[%d]: %w", i, stepErr))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse task steps: %w", err)
	}

	r.prompt, err = cfg.Spec.Prompt.GetValue()
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt for task: %w", err)
	}

	return r, nil
}

func (r *taskRunner) Setup(ctx context.Context) (string, error) {
	for i, s := range r.setup {
		_, err := s.Execute(ctx, &steps.StepInput{
			Workdir: r.baseDir,
		})

		if err != nil {
			return "", fmt.Errorf("setup[%d] failed: %w", i, err)
		}
	}

	return "", nil
}

func (r *taskRunner) Cleanup(ctx context.Context) (string, error) {
	for i, s := range r.cleanup {
		_, err := s.Execute(ctx, &steps.StepInput{
			Workdir: r.baseDir,
		})

		if err != nil {
			return "", fmt.Errorf("cleanup[%d] failed: %w", i, err)
		}
	}

	return "", nil
}

func (r *taskRunner) RunAgent(ctx context.Context, agent agent.Runner) (string, error) {
	result, err := agent.RunTask(ctx, r.prompt)
	if err != nil {
		return "", fmt.Errorf("failed to run agent: %w", err)
	}

	output := result.GetOutput()

	r.output = output

	return output, nil
}

func (r *taskRunner) Verify(ctx context.Context) (string, error) {
	for i, s := range r.verify {
		_, err := s.Execute(ctx, &steps.StepInput{
			Agent: &steps.AgentContext{
				Prompt: r.prompt,
				Output: r.output,
			},
			Workdir: r.baseDir,
		})

		if err != nil {
			return "", fmt.Errorf("verify[%d] failed: %w", i, err)
		}
	}

	return "", nil
}
