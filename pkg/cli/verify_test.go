package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mcpchecker/mcpchecker/pkg/eval"
)

// createTestResultsFile creates a temporary results file for testing
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

// sampleResults returns a set of sample results for testing
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

func TestVerifyCommandPassesThresholds(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	// Task pass rate is 2/3 = 0.667, assertion pass rate is 3/5 = 0.6
	// Setting thresholds below these should pass
	cmd.SetArgs([]string{filePath, "--task", "0.5", "--assertion", "0.5"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command should pass with low thresholds, got error: %v", err)
	}
}

func TestVerifyCommandFailsTaskThreshold(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	// Task pass rate is 2/3 = 0.667, setting threshold to 0.8 should fail
	cmd.SetArgs([]string{filePath, "--task", "0.8", "--assertion", "0.5"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("check command should fail with high task threshold")
	}
}

func TestVerifyCommandFailsAssertionThreshold(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	// Assertion pass rate is 3/5 = 0.6, setting threshold to 0.8 should fail
	cmd.SetArgs([]string{filePath, "--task", "0.5", "--assertion", "0.8"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("check command should fail with high assertion threshold")
	}
}

func TestVerifyCommandDefaultThresholds(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	// Default thresholds are 0.0, should always pass
	cmd.SetArgs([]string{filePath})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command should pass with default thresholds, got error: %v", err)
	}
}

func TestVerifyCommandExactThreshold(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	// Task pass rate is exactly 2/3 = 0.6666...
	// Setting threshold to same value should pass (>= comparison)
	taskRate := 2.0 / 3.0
	cmd.SetArgs([]string{filePath, "--task", "0.6666666666666666", "--assertion", "0.6"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command should pass with exact threshold %f, got error: %v", taskRate, err)
	}
}

func TestVerifyCommandFileNotFound(t *testing.T) {
	cmd := NewVerifyCmd()
	cmd.SetArgs([]string{"/nonexistent/path/results.json", "--task", "0.5"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("check command should fail with nonexistent file")
	}
}

func TestVerifyCommandAllPassed(t *testing.T) {
	// Create results where everything passes (including assertions)
	results := []*eval.EvalResult{
		{
			TaskName:            "task-1",
			TaskPassed:          true,
			AllAssertionsPassed: true,
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed: &eval.SingleAssertionResult{Passed: true},
			},
		},
		{
			TaskName:            "task-2",
			TaskPassed:          true,
			AllAssertionsPassed: true,
			AssertionResults: &eval.CompositeAssertionResult{
				ToolsUsed: &eval.SingleAssertionResult{Passed: true},
			},
		},
	}

	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	cmd.SetArgs([]string{filePath, "--task", "1.0", "--assertion", "1.0"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command should pass when all tasks pass, got error: %v", err)
	}
}

func TestCalculateStats(t *testing.T) {
	results := sampleResults()

	stats := calculateStats("test.json", results)

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
	stats := calculateStats("empty.json", []*eval.EvalResult{})

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

func TestLoadResultsFile(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	loaded, err := loadEvalResults(filePath)
	if err != nil {
		t.Fatalf("loadEvalResults failed: %v", err)
	}

	if len(loaded) != len(results) {
		t.Errorf("loaded %d results, want %d", len(loaded), len(results))
	}

	if loaded[0].TaskName != "task-1" {
		t.Errorf("first task name = %s, want task-1", loaded[0].TaskName)
	}
}

func TestLoadResultsFileNotFound(t *testing.T) {
	_, err := loadEvalResults("/nonexistent/path/results.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadResultsFileInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(filePath, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := loadEvalResults(filePath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
