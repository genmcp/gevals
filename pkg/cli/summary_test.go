package cli

import (
	"bytes"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/eval"
)

func TestSummaryCommand(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewSummaryCmd()
	cmd.SetArgs([]string{filePath})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("summary command failed: %v", err)
	}
}

func TestSummaryCommandWithTaskFilter(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewSummaryCmd()
	cmd.SetArgs([]string{filePath, "--task", "task-1"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("summary command with --task filter failed: %v", err)
	}
}

func TestSummaryCommandJSONOutput(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewSummaryCmd()
	cmd.SetArgs([]string{filePath, "--output", "json"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("summary command with --output json failed: %v", err)
	}
}

func TestSummaryCommandGitHubOutput(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewSummaryCmd()
	cmd.SetArgs([]string{filePath, "--github-output"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("summary command with --github-output failed: %v", err)
	}
}

func TestSummaryCommandEmptyResults(t *testing.T) {
	filePath := createTestResultsFile(t, []*eval.EvalResult{})

	cmd := NewSummaryCmd()
	cmd.SetArgs([]string{filePath})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("summary command with empty results failed: %v", err)
	}
}

func TestSummaryCommandFileNotFound(t *testing.T) {
	cmd := NewSummaryCmd()
	cmd.SetArgs([]string{"/nonexistent/path/results.json"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("summary command should fail with nonexistent file")
	}
}

func TestFilterResults(t *testing.T) {
	results := sampleResults()

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{"existing task", "task-1", 1},
		{"another task", "task-2", 1},
		{"nonexistent task", "task-999", 0},
		{"empty filter returns all", "", 3},
		{"partial match", "task", 3}, // substring matching
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterResults(results, tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("filterResults(%q) returned %d results, want %d", tt.filter, len(filtered), tt.expected)
			}
		})
	}
}

func TestBuildSummaryOutput(t *testing.T) {
	results := sampleResults()
	summary := buildSummaryOutput("test.json", results)

	if summary.TasksTotal != 3 {
		t.Errorf("TasksTotal = %d, want 3", summary.TasksTotal)
	}

	if summary.TasksPassed != 2 {
		t.Errorf("TasksPassed = %d, want 2", summary.TasksPassed)
	}

	if len(summary.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(summary.Tasks))
	}

	// Check first task
	if summary.Tasks[0].Name != "task-1" {
		t.Errorf("Tasks[0].Name = %s, want task-1", summary.Tasks[0].Name)
	}
	if !summary.Tasks[0].TaskPassed {
		t.Error("Tasks[0].TaskPassed should be true")
	}

	// Check failed task
	if summary.Tasks[2].TaskError == "" {
		t.Error("Tasks[2].TaskError should not be empty")
	}
}

func TestCollectFailedAssertions(t *testing.T) {
	results := &eval.CompositeAssertionResult{
		ToolsUsed:    &eval.SingleAssertionResult{Passed: false, Reason: "Tool not called"},
		MinToolCalls: &eval.SingleAssertionResult{Passed: true},
	}

	failures := collectFailedAssertions(results)

	if len(failures) != 1 {
		t.Errorf("len(failures) = %d, want 1", len(failures))
	}

	if len(failures) > 0 && failures[0] != "ToolsUsed: Tool not called" {
		t.Errorf("failures[0] = %s, want 'ToolsUsed: Tool not called'", failures[0])
	}
}

func TestOutputTextSummary(t *testing.T) {
	results := sampleResults()
	summary := buildSummaryOutput("test.json", results)

	// Just ensure it doesn't panic
	outputTextSummary(results, summary)
}

func TestOutputTextSummaryAllPassed(t *testing.T) {
	results := []*eval.EvalResult{
		{
			TaskName:            "task-1",
			TaskPassed:          true,
			AllAssertionsPassed: true,
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed: &eval.SingleAssertionResult{Passed: true},
			},
		},
	}
	summary := buildSummaryOutput("test.json", results)

	// Just ensure it doesn't panic
	outputTextSummary(results, summary)
}

func TestOutputTextSummaryAllFailed(t *testing.T) {
	results := []*eval.EvalResult{
		{
			TaskName:            "task-1",
			TaskPassed:          false,
			TaskError:           "something went wrong",
			AllAssertionsPassed: false,
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed: &eval.SingleAssertionResult{Passed: false, Reason: "Tool not called"},
			},
		},
	}
	summary := buildSummaryOutput("test.json", results)

	// Just ensure it doesn't panic
	outputTextSummary(results, summary)
}

func TestOutputTextSummaryAgentExecutionError(t *testing.T) {
	results := []*eval.EvalResult{
		{
			TaskName:            "task-1",
			TaskPassed:          false,
			AgentExecutionError: true,
			AllAssertionsPassed: false,
		},
	}
	summary := buildSummaryOutput("test.json", results)

	// Just ensure it doesn't panic
	outputTextSummary(results, summary)
}
