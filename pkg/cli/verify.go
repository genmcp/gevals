// Package cli provides commands for rendering and inspecting evaluation results.
package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mcpchecker/mcpchecker/pkg/eval"
	"github.com/spf13/cobra"
)

// Stats holds computed statistics from evaluation results
type Stats struct {
	ResultsFile       string  `json:"resultsFile"`
	TasksTotal        int     `json:"tasksTotal"`
	TasksPassed       int     `json:"tasksPassed"`
	TaskPassRate      float64 `json:"taskPassRate"`
	AssertionsTotal   int     `json:"assertionsTotal"`
	AssertionsPassed  int     `json:"assertionsPassed"`
	AssertionPassRate float64 `json:"assertionPassRate"`
}

func calculateStats(resultsFile string, results []*eval.EvalResult) Stats {
	stats := Stats{
		ResultsFile: resultsFile,
		TasksTotal:  len(results),
	}

	for _, result := range results {
		if result.TaskPassed {
			stats.TasksPassed++
		}

		if result.AssertionResults != nil {
			stats.AssertionsTotal += result.AssertionResults.TotalAssertions()
			stats.AssertionsPassed += result.AssertionResults.PassedAssertions()
		}
	}

	// Calculate pass rates
	if stats.TasksTotal > 0 {
		stats.TaskPassRate = float64(stats.TasksPassed) / float64(stats.TasksTotal)
	}
	if stats.AssertionsTotal > 0 {
		stats.AssertionPassRate = float64(stats.AssertionsPassed) / float64(stats.AssertionsTotal)
	}

	return stats
}

// NewVerifyCmd creates the verify command
func NewVerifyCmd() *cobra.Command {
	var taskThreshold float64
	var assertionThreshold float64

	cmd := &cobra.Command{
		Use:   "verify <results-file>",
		Short: "Verify evaluation results meet thresholds",
		Long: `Verify that evaluation results meet minimum pass rate thresholds.

Exits with code 0 if all thresholds are met, code 1 otherwise.
Use 'mcpchecker summary' to view detailed results.`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resultsFile := args[0]

			results, err := loadEvalResults(resultsFile)
			if err != nil {
				return fmt.Errorf("failed to load results file: %w", err)
			}

			stats := calculateStats(resultsFile, results)

			taskThresholdMet := stats.TaskPassRate >= taskThreshold
			assertionThresholdMet := stats.AssertionPassRate >= assertionThreshold
			passed := taskThresholdMet && assertionThresholdMet

			outputVerifyResults(stats, taskThreshold, assertionThreshold, taskThresholdMet, assertionThresholdMet, passed)

			if !passed {
				return fmt.Errorf("thresholds not met")
			}

			return nil
		},
	}

	cmd.Flags().Float64Var(&taskThreshold, "task", 0.0, "Minimum task pass rate (0.0-1.0)")
	cmd.Flags().Float64Var(&assertionThreshold, "assertion", 0.0, "Minimum assertion pass rate (0.0-1.0)")

	return cmd
}

func outputVerifyResults(stats Stats, taskThreshold, assertionThreshold float64, taskMet, assertionMet, passed bool) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	bold := color.New(color.Bold)

	_, _ = bold.Println("=== Threshold Verification ===")
	fmt.Println()

	// Task threshold
	if taskMet {
		_, _ = green.Printf("Task Pass Rate:      %.2f%% >= %.2f%% ✓\n",
			stats.TaskPassRate*100, taskThreshold*100)
	} else {
		_, _ = red.Printf("Task Pass Rate:      %.2f%% < %.2f%% ✗\n",
			stats.TaskPassRate*100, taskThreshold*100)
	}

	// Assertion threshold
	if assertionMet {
		_, _ = green.Printf("Assertion Pass Rate: %.2f%% >= %.2f%% ✓\n",
			stats.AssertionPassRate*100, assertionThreshold*100)
	} else {
		_, _ = red.Printf("Assertion Pass Rate: %.2f%% < %.2f%% ✗\n",
			stats.AssertionPassRate*100, assertionThreshold*100)
	}

	fmt.Println()
	if passed {
		_, _ = green.Println("Result: PASSED")
	} else {
		_, _ = red.Println("Result: FAILED")
	}
}
