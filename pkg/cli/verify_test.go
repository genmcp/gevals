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

func TestVerifyCommandLogsTaskThresholdFailure(t *testing.T) {
	evalResults := sampleResults()
	filePath := createTestResultsFile(t, evalResults)

	cmd := NewVerifyCmd()
	// Task pass rate is 2/3 = 0.667, setting threshold to 0.8 should fail
	cmd.SetArgs([]string{filePath, "--task", "0.8", "--assertion", "0.5"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	// Command should return error when thresholds not met (exit code 1 for CI)
	err := cmd.Execute()
	if err == nil {
		t.Errorf("verify command should return error when task threshold not met")
	}
}

func TestVerifyCommandLogsAssertionThresholdFailure(t *testing.T) {
	evalResults := sampleResults()
	filePath := createTestResultsFile(t, evalResults)

	cmd := NewVerifyCmd()
	// Assertion pass rate is 3/5 = 0.6, setting threshold to 0.8 should fail
	cmd.SetArgs([]string{filePath, "--task", "0.5", "--assertion", "0.8"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	// Command should return error when thresholds not met (exit code 1 for CI)
	err := cmd.Execute()
	if err == nil {
		t.Errorf("verify command should return error when assertion threshold not met")
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

func TestVerifyCommandNearThreshold(t *testing.T) {
	results := sampleResults()
	filePath := createTestResultsFile(t, results)

	cmd := NewVerifyCmd()
	// Task pass rate is 2/3 ≈ 0.667
	// Setting threshold just below should pass (>= comparison)
	cmd.SetArgs([]string{filePath, "--task", "0.66", "--assertion", "0.6"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command should pass with threshold just below actual rate, got error: %v", err)
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

