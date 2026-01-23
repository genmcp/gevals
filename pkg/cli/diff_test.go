package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/eval"
)

func TestDiffCommand(t *testing.T) {
	baseResults := sampleResults()
	currentResults := sampleResultsImproved()

	baseFile := createTestResultsFile(t, baseResults)
	currentFile := createTestResultsFile(t, currentResults)

	cmd := NewDiffCmd()
	cmd.SetArgs([]string{"--base", baseFile, "--current", currentFile})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("diff command failed: %v", err)
	}
}

func TestDiffCommandMarkdown(t *testing.T) {
	baseResults := sampleResults()
	currentResults := sampleResultsImproved()

	baseFile := createTestResultsFile(t, baseResults)
	currentFile := createTestResultsFile(t, currentResults)

	cmd := NewDiffCmd()
	cmd.SetArgs([]string{"--base", baseFile, "--current", currentFile, "--output", "markdown"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("diff command with --output markdown failed: %v", err)
	}
}

func TestDiffCommandBaseNotFound(t *testing.T) {
	currentResults := sampleResults()
	currentFile := createTestResultsFile(t, currentResults)

	cmd := NewDiffCmd()
	cmd.SetArgs([]string{"--base", "/nonexistent/path/base.json", "--current", currentFile})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("diff command should fail with nonexistent base file")
	}
}

func TestDiffCommandCurrentNotFound(t *testing.T) {
	baseResults := sampleResults()
	baseFile := createTestResultsFile(t, baseResults)

	cmd := NewDiffCmd()
	cmd.SetArgs([]string{"--base", baseFile, "--current", "/nonexistent/path/current.json"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("diff command should fail with nonexistent current file")
	}
}

func TestCalculateDiff(t *testing.T) {
	baseResults := sampleResults()
	headResults := sampleResultsImproved()

	diff := calculateDiff("base.json", "head.json", baseResults, headResults)

	// Check base stats
	if diff.BaseStats.TasksTotal != 3 {
		t.Errorf("BaseStats.TasksTotal = %d, want 3", diff.BaseStats.TasksTotal)
	}

	// Check head stats (improved results have 4 tasks)
	if diff.HeadStats.TasksTotal != 4 {
		t.Errorf("HeadStats.TasksTotal = %d, want 4", diff.HeadStats.TasksTotal)
	}

	// Should have 1 improvement (task-2 passes in head)
	if len(diff.Improvements) != 1 {
		t.Errorf("len(Improvements) = %d, want 1", len(diff.Improvements))
	}

	// Should have 1 new task
	if len(diff.New) != 1 {
		t.Errorf("len(New) = %d, want 1", len(diff.New))
	}
}

func TestCalculateDiffRegressions(t *testing.T) {
	// Swap base and head to test regressions
	baseResults := sampleResultsImproved()
	headResults := sampleResults()

	diff := calculateDiff("base.json", "head.json", baseResults, headResults)

	// Should have 1 regression (task-2 fails in head)
	if len(diff.Regressions) != 1 {
		t.Errorf("len(Regressions) = %d, want 1", len(diff.Regressions))
	}

	// Should have 1 removed task
	if len(diff.Removed) != 1 {
		t.Errorf("len(Removed) = %d, want 1", len(diff.Removed))
	}
}

func TestCalculateDiffNoChanges(t *testing.T) {
	results := sampleResults()

	diff := calculateDiff("base.json", "head.json", results, results)

	if len(diff.Regressions) != 0 {
		t.Errorf("len(Regressions) = %d, want 0", len(diff.Regressions))
	}

	if len(diff.Improvements) != 0 {
		t.Errorf("len(Improvements) = %d, want 0", len(diff.Improvements))
	}

	if len(diff.New) != 0 {
		t.Errorf("len(New) = %d, want 0", len(diff.New))
	}

	if len(diff.Removed) != 0 {
		t.Errorf("len(Removed) = %d, want 0", len(diff.Removed))
	}
}

func TestCalculateDiffEmptyBase(t *testing.T) {
	headResults := sampleResults()

	diff := calculateDiff("base.json", "head.json", []*eval.EvalResult{}, headResults)

	// All tasks in head should be "new"
	if len(diff.New) != 3 {
		t.Errorf("len(New) = %d, want 3", len(diff.New))
	}
}

func TestCalculateDiffEmptyHead(t *testing.T) {
	baseResults := sampleResults()

	diff := calculateDiff("base.json", "head.json", baseResults, []*eval.EvalResult{})

	// All tasks in base should be "removed"
	if len(diff.Removed) != 3 {
		t.Errorf("len(Removed) = %d, want 3", len(diff.Removed))
	}
}

func TestFormatChangeMarkdown(t *testing.T) {
	tests := []struct {
		change   float64
		contains string
	}{
		{0.1, "🟢"},
		{-0.1, "🔴"},
		{0.0, "➖"},
	}

	for _, tt := range tests {
		result := formatChangeMarkdown(tt.change)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("formatChangeMarkdown(%f) = %q, want to contain %q", tt.change, result, tt.contains)
		}
	}
}

// sampleResultsImproved returns improved results for diff testing
func sampleResultsImproved() []*eval.EvalResult {
	return []*eval.EvalResult{
		{
			TaskName:   "task-1",
			TaskPath:   "/path/to/task-1",
			TaskPassed: true,
			Difficulty: "easy",
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed:    &eval.SingleAssertionResult{Passed: true},
				MinToolCalls: &eval.SingleAssertionResult{Passed: true},
			},
			AllAssertionsPassed: true,
		},
		{
			TaskName:   "task-2",
			TaskPath:   "/path/to/task-2",
			TaskPassed: true,
			Difficulty: "medium",
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed:     &eval.SingleAssertionResult{Passed: true},
				ResourcesRead: &eval.SingleAssertionResult{Passed: true}, // Now passes
			},
			AllAssertionsPassed: true, // Now passes
		},
		{
			TaskName:   "task-3",
			TaskPath:   "/path/to/task-3",
			TaskPassed: false,
			TaskError:  "verification failed",
			Difficulty: "hard",
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed: &eval.SingleAssertionResult{Passed: false, Reason: "Tool not called"},
			},
			AllAssertionsPassed: false,
		},
		{
			TaskName:   "task-4",
			TaskPath:   "/path/to/task-4",
			TaskPassed: true,
			Difficulty: "easy",
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed: &eval.SingleAssertionResult{Passed: true},
			},
			AllAssertionsPassed: true,
		},
	}
}
