package eval

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/genmcp/gevals/pkg/agent"
	"github.com/genmcp/gevals/pkg/mcpproxy"
	"github.com/genmcp/gevals/pkg/task"
)

type EvalResult struct {
	TaskName            string                    `json:"taskName"`
	TaskPath            string                    `json:"taskPath"`
	TaskPassed          bool                      `json:"taskPassed"`
	TaskOutput          string                    `json:"taskOutput"`
	TaskError           string                    `json:"taskError,omitempty"`
	AgentExecutionError bool                      `json:"agentExecutionError,omitempty"` // True if agent failed to execute
	Difficulty          string                    `json:"difficulty"`
	AssertionResults    *CompositeAssertionResult `json:"assertionResults"`
	AllAssertionsPassed bool                      `json:"allAssertionsPassed"`
	CallHistory         *mcpproxy.CallHistory     `json:"callHistory"`
}

type EvalRunner interface {
	Run(ctx context.Context) ([]*EvalResult, error)
	RunWithProgress(ctx context.Context, callback ProgressCallback) ([]*EvalResult, error)
}

type evalRunner struct {
	spec             *EvalSpec
	mcpConfig        *mcpproxy.MCPConfig
	progressCallback ProgressCallback
}

var _ EvalRunner = &evalRunner{}

type taskConfig struct {
	path       string
	spec       *task.TaskSpec
	assertions *TaskAssertions
}

// NewRunner creates a new EvalRunner from an EvalSpec
func NewRunner(spec *EvalSpec) (EvalRunner, error) {
	if spec == nil {
		return nil, fmt.Errorf("eval spec cannot be nil")
	}

	return &evalRunner{
		spec:             spec,
		progressCallback: NoopProgressCallback,
	}, nil
}

func (r *evalRunner) Run(ctx context.Context) ([]*EvalResult, error) {
	return r.RunWithProgress(ctx, NoopProgressCallback)
}

func (r *evalRunner) RunWithProgress(ctx context.Context, callback ProgressCallback) ([]*EvalResult, error) {
	r.progressCallback = callback

	r.progressCallback(ProgressEvent{
		Type:    EventEvalStart,
		Message: "Starting evaluation",
	})

	mcpConfig, err := mcpproxy.ParseConfigFile(r.spec.Config.McpConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP config: %w", err)
	}

	r.mcpConfig = mcpConfig

	agentSpec, err := agent.FromFile(r.spec.Config.AgentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent spec: %w", err)
	}

	runner, err := agent.NewRunnerForSpec(agentSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent runner from spec: %w", err)
	}

	taskConfigs, err := r.collectTaskConfigs()
	if err != nil {
		return nil, err
	}

	results := make([]*EvalResult, 0, len(taskConfigs))
	var runErr error
	for _, tc := range taskConfigs {
		result, err := r.runTask(ctx, runner, mcpConfig, tc)
		if err != nil {
			runErr = errors.Join(runErr, err)
		} else {
			results = append(results, result)
		}
	}

	r.progressCallback(ProgressEvent{
		Type:    EventEvalComplete,
		Message: "Evaluation complete",
	})

	return results, runErr
}

func (r *evalRunner) collectTaskConfigs() ([]taskConfig, error) {
	taskConfigs := make([]taskConfig, 0)

	for _, ts := range r.spec.Config.TaskSets {
		var paths []string
		var err error

		if ts.Glob != "" {
			paths, err = filepath.Glob(ts.Glob)
			if err != nil {
				return nil, fmt.Errorf("failed to glob %s: %w", ts.Glob, err)
			}
		} else if ts.Path != "" {
			paths = []string{ts.Path}
		}

		for _, path := range paths {
			taskSpec, err := task.FromFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to load task at path %s: %w", path, err)
			}

			taskConfigs = append(taskConfigs, taskConfig{
				path:       path,
				spec:       taskSpec,
				assertions: ts.Assertions,
			})
		}
	}

	return taskConfigs, nil
}

func (r *evalRunner) runTask(
	ctx context.Context,
	agentRunner agent.Runner,
	mcpConfig *mcpproxy.MCPConfig,
	tc taskConfig,
) (*EvalResult, error) {
	result := &EvalResult{
		TaskName:   tc.spec.Metadata.Name,
		TaskPath:   tc.path,
		Difficulty: tc.spec.Metadata.Difficulty,
	}

	r.progressCallback(ProgressEvent{
		Type:    EventTaskStart,
		Message: fmt.Sprintf("Starting task: %s", tc.spec.Metadata.Name),
		Task:    result,
	})

	r.progressCallback(ProgressEvent{
		Type:    EventTaskSetup,
		Message: fmt.Sprintf("Setting up task: %s", tc.spec.Metadata.Name),
		Task:    result,
	})

	taskRunner, manager, cleanup, err := r.setupTaskResources(ctx, tc, mcpConfig)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	r.executeTaskSteps(ctx, taskRunner, agentRunner, manager, result)

	r.progressCallback(ProgressEvent{
		Type:    EventTaskAssertions,
		Message: fmt.Sprintf("Evaluating assertions for task: %s", tc.spec.Metadata.Name),
		Task:    result,
	})

	r.evaluateTaskAssertions(tc, manager, result)

	result.CallHistory = manager.GetAllCallHistory()

	r.progressCallback(ProgressEvent{
		Type:    EventTaskComplete,
		Message: fmt.Sprintf("Completed task: %s (passed: %v)", tc.spec.Metadata.Name, result.TaskPassed),
		Task:    result,
	})

	return result, nil
}

func (r *evalRunner) setupTaskResources(
	ctx context.Context,
	tc taskConfig,
	mcpConfig *mcpproxy.MCPConfig,
) (task.TaskRunner, mcpproxy.ServerManager, func(), error) {
	taskRunner, err := task.NewTaskRunner(tc.spec)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create task runner for task '%s': %w", tc.spec.Metadata.Name, err)
	}

	manager, err := mcpproxy.NewServerManger(ctx, mcpConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create mcp proxy server manager: %w", err)
	}

	if err := manager.Start(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start mcp proxy servers: %w", err)
	}

	out, err := taskRunner.Setup(ctx)
	if err != nil {
		manager.Close()
		return nil, nil, nil, fmt.Errorf("failed to setup task, got output %s: %w", out, err)
	}

	cleanup := func() {
		// TODO: find a way to surface cleanup failures
		_, _ = taskRunner.Cleanup(ctx)
		manager.Close()
	}

	return taskRunner, manager, cleanup, nil
}

func (r *evalRunner) executeTaskSteps(
	ctx context.Context,
	taskRunner task.TaskRunner,
	agentRunner agent.Runner,
	manager mcpproxy.ServerManager,
	result *EvalResult,
) {
	r.progressCallback(ProgressEvent{
		Type:    EventTaskRunning,
		Message: fmt.Sprintf("Running agent for task: %s", result.TaskName),
		Task:    result,
	})

	agentRunner = agentRunner.WithMcpServerInfo(manager)

	out, err := taskRunner.RunAgent(ctx, agentRunner)
	if err != nil {
		result.TaskPassed = false
		result.TaskOutput = out
		result.TaskError = err.Error()
		result.AgentExecutionError = true
		return
	}

	result.TaskOutput = out

	r.progressCallback(ProgressEvent{
		Type:    EventTaskVerifying,
		Message: fmt.Sprintf("Verifying task: %s", result.TaskName),
		Task:    result,
	})

	out, err = taskRunner.Verify(ctx)
	if err != nil {
		result.TaskPassed = false
		result.TaskError = fmt.Sprintf("verification script failed with output '%s': %s", out, err.Error())
	} else {
		result.TaskPassed = true
	}
}

func (r *evalRunner) evaluateTaskAssertions(
	tc taskConfig,
	manager mcpproxy.ServerManager,
	result *EvalResult,
) {
	if tc.assertions != nil {
		evaluator := NewCompositeAssertionEvaluator(tc.assertions)
		assertionResults := evaluator.Evaluate(manager.GetAllCallHistory())

		result.AssertionResults = assertionResults
		result.AllAssertionsPassed = assertionResults.Succeeded()
	} else {
		// No assertions = all pass
		result.AllAssertionsPassed = true
	}
}
