package results

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/eval"
)

// createTestResultsFile creates a temporary results file for testing.
func createTestResultsFile(t *testing.T, results []*eval.EvalResult) string {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "results.json")

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal results: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write results file: %v", err)
	}

	return filePath
}

// sampleResults returns a set of sample results for testing.
func sampleResults() []*eval.EvalResult {
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
				ResourcesRead: &eval.SingleAssertionResult{Passed: false, Reason: "Resource not read"},
			},
			AllAssertionsPassed: false,
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
	}
}

func TestCalculateStats(t *testing.T) {
	evalResults := sampleResults()

	stats := CalculateStats("test.json", evalResults)

	if stats.TasksTotal != 3 {
		t.Errorf("TasksTotal = %d, want 3", stats.TasksTotal)
	}

	if stats.TasksPassed != 2 {
		t.Errorf("TasksPassed = %d, want 2", stats.TasksPassed)
	}

	if stats.AssertionsTotal != 5 {
		t.Errorf("AssertionsTotal = %d, want 5", stats.AssertionsTotal)
	}

	if stats.AssertionsPassed != 3 {
		t.Errorf("AssertionsPassed = %d, want 3", stats.AssertionsPassed)
	}

	expectedTaskRate := 2.0 / 3.0
	if stats.TaskPassRate != expectedTaskRate {
		t.Errorf("TaskPassRate = %f, want %f", stats.TaskPassRate, expectedTaskRate)
	}

	expectedAssertionRate := 3.0 / 5.0
	if stats.AssertionPassRate != expectedAssertionRate {
		t.Errorf("AssertionPassRate = %f, want %f", stats.AssertionPassRate, expectedAssertionRate)
	}
}

func TestCalculateStatsEmptyResults(t *testing.T) {
	stats := CalculateStats("empty.json", []*eval.EvalResult{})

	if stats.TasksTotal != 0 {
		t.Errorf("TasksTotal = %d, want 0", stats.TasksTotal)
	}

	if stats.TaskPassRate != 0 {
		t.Errorf("TaskPassRate = %f, want 0", stats.TaskPassRate)
	}

	if stats.AssertionPassRate != 0 {
		t.Errorf("AssertionPassRate = %f, want 0", stats.AssertionPassRate)
	}
}

func TestLoad(t *testing.T) {
	evalResults := sampleResults()
	filePath := createTestResultsFile(t, evalResults)

	loaded, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded) != len(evalResults) {
		t.Errorf("loaded %d results, want %d", len(loaded), len(evalResults))
	}

	if loaded[0].TaskName != "task-1" {
		t.Errorf("first task name = %s, want task-1", loaded[0].TaskName)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/results.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(filePath, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := Load(filePath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestFilter(t *testing.T) {
	evalResults := sampleResults()

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{"existing task", "task-1", 1},
		{"another task", "task-2", 1},
		{"nonexistent task", "task-999", 0},
		{"empty filter returns all", "", 3},
		{"partial match", "task", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := Filter(evalResults, tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("Filter(%q) returned %d results, want %d", tt.filter, len(filtered), tt.expected)
			}
		})
	}
}

func TestCollectFailedAssertions(t *testing.T) {
	assertionResults := &eval.CompositeAssertionResult{
		ToolsUsed:    &eval.SingleAssertionResult{Passed: false, Reason: "Tool not called"},
		MinToolCalls: &eval.SingleAssertionResult{Passed: true},
	}

	failures := CollectFailedAssertions(assertionResults)

	if len(failures) != 1 {
		t.Errorf("len(failures) = %d, want 1", len(failures))
	}

	if len(failures) > 0 && failures[0] != "ToolsUsed: Tool not called" {
		t.Errorf("failures[0] = %s, want 'ToolsUsed: Tool not called'", failures[0])
	}
}
