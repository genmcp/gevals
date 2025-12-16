package testcase

import (
	"strings"

	"github.com/genmcp/gevals/pkg/llmjudge"
	"github.com/genmcp/gevals/pkg/task"
	"github.com/genmcp/gevals/pkg/util"
)

// shellEscapeSingleQuote escapes a string for use within single quotes in a shell command.
// It replaces each ' with '\'' (end single quote, escaped single quote, start single quote).
func shellEscapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// TaskConfig provides a fluent API for building task configurations.
// A task defines what the agent should do and how to verify the result.
type TaskConfig struct {
	spec *task.TaskSpec
}

// NewTaskConfig creates a new task config builder
func NewTaskConfig() *TaskConfig {
	return &TaskConfig{
		spec: &task.TaskSpec{
			Steps: task.TaskSteps{
				VerifyScript: &task.VerifyStep{},
			},
		},
	}
}

// Name sets the task name
func (tc *TaskConfig) Name(name string) *TaskConfig {
	tc.spec.Metadata.Name = name
	return tc
}

// Difficulty sets the task difficulty level
func (tc *TaskConfig) Difficulty(difficulty string) *TaskConfig {
	tc.spec.Metadata.Difficulty = difficulty
	return tc
}

// Easy sets the difficulty to "easy"
func (tc *TaskConfig) Easy() *TaskConfig {
	return tc.Difficulty(task.DifficultyEasy)
}

// Medium sets the difficulty to "medium"
func (tc *TaskConfig) Medium() *TaskConfig {
	return tc.Difficulty(task.DifficultyMedium)
}

// Hard sets the difficulty to "hard"
func (tc *TaskConfig) Hard() *TaskConfig {
	return tc.Difficulty(task.DifficultyHard)
}

// Prompt sets the prompt text for the agent.
// The prompt is shell-escaped for single quotes since the agent spec template
// uses single quotes around the prompt argument.
func (tc *TaskConfig) Prompt(prompt string) *TaskConfig {
	tc.spec.Steps.Prompt = &util.Step{
		Inline: shellEscapeSingleQuote(prompt),
	}
	return tc
}

// PromptFile sets the prompt to be read from a file
func (tc *TaskConfig) PromptFile(path string) *TaskConfig {
	tc.spec.Steps.Prompt = &util.Step{
		File: path,
	}
	return tc
}

// SetupScript sets an inline setup script to run before the task
func (tc *TaskConfig) SetupScript(script string) *TaskConfig {
	tc.spec.Steps.SetupScript = &util.Step{
		Inline: script,
	}
	return tc
}

// SetupScriptFile sets the setup script to be read from a file
func (tc *TaskConfig) SetupScriptFile(path string) *TaskConfig {
	tc.spec.Steps.SetupScript = &util.Step{
		File: path,
	}
	return tc
}

// CleanupScript sets an inline cleanup script to run after the task
func (tc *TaskConfig) CleanupScript(script string) *TaskConfig {
	tc.spec.Steps.CleanupScript = &util.Step{
		Inline: script,
	}
	return tc
}

// CleanupScriptFile sets the cleanup script to be read from a file
func (tc *TaskConfig) CleanupScriptFile(path string) *TaskConfig {
	tc.spec.Steps.CleanupScript = &util.Step{
		File: path,
	}
	return tc
}

// VerifyScript sets an inline verification script
func (tc *TaskConfig) VerifyScript(script string) *TaskConfig {
	tc.spec.Steps.VerifyScript = &task.VerifyStep{
		Step: &util.Step{
			Inline: script,
		},
	}
	return tc
}

// VerifyScriptFile sets the verification script to be read from a file
func (tc *TaskConfig) VerifyScriptFile(path string) *TaskConfig {
	tc.spec.Steps.VerifyScript = &task.VerifyStep{
		Step: &util.Step{
			File: path,
		},
	}
	return tc
}

// VerifyContains sets LLM judge verification with CONTAINS mode.
// The judge will check if the agent output contains the expected content.
func (tc *TaskConfig) VerifyContains(expected string) *TaskConfig {
	tc.spec.Steps.VerifyScript = &task.VerifyStep{
		LLMJudgeTaskConfig: &llmjudge.LLMJudgeTaskConfig{
			Contains: expected,
		},
	}
	return tc
}

// VerifyExact sets LLM judge verification with EXACT mode.
// The judge will check if the agent output exactly matches the expected content.
func (tc *TaskConfig) VerifyExact(expected string) *TaskConfig {
	tc.spec.Steps.VerifyScript = &task.VerifyStep{
		LLMJudgeTaskConfig: &llmjudge.LLMJudgeTaskConfig{
			Exact: expected,
		},
	}
	return tc
}

// Build returns the task spec
func (tc *TaskConfig) Build() *task.TaskSpec {
	return tc.spec
}

// Re-export types for convenience
type (
	TaskSpec     = task.TaskSpec
	TaskMetadata = task.TaskMetadata
	TaskSteps    = task.TaskSteps
	VerifyStep   = task.VerifyStep
)

// Re-export difficulty constants
const (
	DifficultyEasy   = task.DifficultyEasy
	DifficultyMedium = task.DifficultyMedium
	DifficultyHard   = task.DifficultyHard
)
