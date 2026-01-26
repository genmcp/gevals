// Package cli provides commands for rendering and inspecting evaluation results.
package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mcpchecker/mcpchecker/pkg/results"
	"github.com/spf13/cobra"
)

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
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resultsFile := args[0]

			evalResults, err := results.Load(resultsFile)
			if err != nil {
				return fmt.Errorf("failed to load results file: %w", err)
			}

			stats := results.CalculateStats(resultsFile, evalResults)

			taskThresholdMet := stats.TaskPassRate >= taskThreshold
			// If no assertions exist, skip the assertion threshold check
			assertionThresholdMet := stats.AssertionsTotal == 0 || stats.AssertionPassRate >= assertionThreshold
			passed := taskThresholdMet && assertionThresholdMet

			outputVerifyResults(stats, taskThreshold, assertionThreshold, taskThresholdMet, assertionThresholdMet, passed)

			if !passed {
				// silent error (SilenceErrors: true), sets exit code 1
				return fmt.Errorf("thresholds not met")
			}

			return nil
		},
	}

	cmd.Flags().Float64Var(&taskThreshold, "task", 0.0, "Minimum task pass rate (0.0-1.0)")
	cmd.Flags().Float64Var(&assertionThreshold, "assertion", 0.0, "Minimum assertion pass rate (0.0-1.0)")

	return cmd
}

func outputVerifyResults(stats results.Stats, taskThreshold, assertionThreshold float64, taskMet, assertionMet, passed bool) {
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
	if stats.AssertionsTotal == 0 {
		fmt.Println("Assertion Pass Rate: N/A (no assertions defined)")
	} else if assertionMet {
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
