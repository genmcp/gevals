package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mcpchecker/mcpchecker/pkg/eval"
	"github.com/mcpchecker/mcpchecker/pkg/results"
	"github.com/spf13/cobra"
)

// DiffResult holds the comparison between two evaluation runs
type DiffResult struct {
	BaseStats    results.Stats
	HeadStats    results.Stats
	Regressions  []TaskDiff
	Improvements []TaskDiff
	New          []TaskDiff
	Removed      []TaskDiff
}

// TaskDiff holds the diff for a single task
type TaskDiff struct {
	TaskName           string
	BasePassed         bool
	HeadPassed         bool
	BaseAssertions     int
	HeadAssertions     int
	BaseAssertionTotal int
	HeadAssertionTotal int
	FailureReason      string
}

// NewDiffCmd creates the diff command
func NewDiffCmd() *cobra.Command {
	var outputFormat string
	var baseFile string
	var currentFile string

	cmd := &cobra.Command{
		Use:   "diff --base <results-file> --current <results-file>",
		Short: "Compare two evaluation results",
		Long: `Compare evaluation results between two runs (e.g., main vs PR).

Shows regressions, improvements, and overall pass rate changes.
Useful for posting on pull requests to show impact of changes.

Example:
  mcpchecker diff --base results-main.json --current results-pr.json
  mcpchecker diff --base results-main.json --current results-pr.json --output markdown`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseResults, err := results.Load(baseFile)
			if err != nil {
				return fmt.Errorf("failed to load base results: %w", err)
			}

			currentResults, err := results.Load(currentFile)
			if err != nil {
				return fmt.Errorf("failed to load current results: %w", err)
			}

			diff := calculateDiff(baseFile, currentFile, baseResults, currentResults)

			switch outputFormat {
			case "text":
				outputTextDiff(diff)
			case "markdown":
				outputMarkdownDiff(diff)
			default:
				return fmt.Errorf("unknown output format: %s", outputFormat)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&baseFile, "base", "", "Base results file (e.g., main branch)")
	cmd.Flags().StringVar(&currentFile, "current", "", "Current results file (e.g., PR branch)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, markdown)")

	_ = cmd.MarkFlagRequired("base")
	_ = cmd.MarkFlagRequired("current")

	return cmd
}

func calculateDiff(baseFile, currentFile string, baseResults, currentResults []*eval.EvalResult) DiffResult {
	diff := DiffResult{
		BaseStats:    results.CalculateStats(baseFile, baseResults),
		HeadStats:    results.CalculateStats(currentFile, currentResults),
		Regressions:  make([]TaskDiff, 0),
		Improvements: make([]TaskDiff, 0),
		New:          make([]TaskDiff, 0),
		Removed:      make([]TaskDiff, 0),
	}

	baseMap := make(map[string]*eval.EvalResult)
	for _, r := range baseResults {
		baseMap[r.TaskName] = r
	}

	currentMap := make(map[string]*eval.EvalResult)
	for _, r := range currentResults {
		currentMap[r.TaskName] = r
	}

	for _, current := range currentResults {
		base, exists := baseMap[current.TaskName]
		if !exists {
			diff.New = append(diff.New, TaskDiff{
				TaskName:           current.TaskName,
				HeadPassed:         current.TaskPassed && current.AllAssertionsPassed,
				HeadAssertions:     results.PassedAssertions(current),
				HeadAssertionTotal: results.TotalAssertions(current),
			})
			continue
		}

		basePassed := base.TaskPassed && base.AllAssertionsPassed
		currentPassed := current.TaskPassed && current.AllAssertionsPassed

		taskDiff := TaskDiff{
			TaskName:           current.TaskName,
			BasePassed:         basePassed,
			HeadPassed:         currentPassed,
			BaseAssertions:     results.PassedAssertions(base),
			HeadAssertions:     results.PassedAssertions(current),
			BaseAssertionTotal: results.TotalAssertions(base),
			HeadAssertionTotal: results.TotalAssertions(current),
			FailureReason:      results.FailureReason(current),
		}

		if basePassed && !currentPassed {
			diff.Regressions = append(diff.Regressions, taskDiff)
		} else if !basePassed && currentPassed {
			diff.Improvements = append(diff.Improvements, taskDiff)
		}
	}

	for _, base := range baseResults {
		if _, exists := currentMap[base.TaskName]; !exists {
			diff.Removed = append(diff.Removed, TaskDiff{
				TaskName:           base.TaskName,
				BasePassed:         base.TaskPassed && base.AllAssertionsPassed,
				BaseAssertions:     results.PassedAssertions(base),
				BaseAssertionTotal: results.TotalAssertions(base),
			})
		}
	}

	return diff
}

func outputTextDiff(diff DiffResult) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	bold := color.New(color.Bold)

	_, _ = bold.Println("=== Evaluation Diff ===")
	fmt.Println()

	// Regressions
	if len(diff.Regressions) > 0 {
		_, _ = red.Printf("Regressions (%d):\n", len(diff.Regressions))
		for _, r := range diff.Regressions {
			_, _ = red.Printf("  ✗ %s: PASSED → FAILED\n", r.TaskName)
			if r.FailureReason != "" {
				fmt.Printf("      %s\n", r.FailureReason)
			}
		}
		fmt.Println()
	}

	// Improvements
	if len(diff.Improvements) > 0 {
		_, _ = green.Printf("Improvements (%d):\n", len(diff.Improvements))
		for _, r := range diff.Improvements {
			_, _ = green.Printf("  ✓ %s: FAILED → PASSED\n", r.TaskName)
		}
		fmt.Println()
	}

	// New tasks
	if len(diff.New) > 0 {
		_, _ = yellow.Printf("New Tasks (%d):\n", len(diff.New))
		for _, r := range diff.New {
			if r.HeadPassed {
				_, _ = green.Printf("  + %s: PASSED\n", r.TaskName)
			} else {
				_, _ = red.Printf("  + %s: FAILED\n", r.TaskName)
			}
		}
		fmt.Println()
	}

	// Removed tasks
	if len(diff.Removed) > 0 {
		_, _ = yellow.Printf("Removed Tasks (%d):\n", len(diff.Removed))
		for _, r := range diff.Removed {
			fmt.Printf("  - %s\n", r.TaskName)
		}
		fmt.Println()
	}

	// Summary table
	_, _ = bold.Println("=== Summary ===")
	fmt.Println()

	taskChange := diff.HeadStats.TaskPassRate - diff.BaseStats.TaskPassRate
	assertionChange := diff.HeadStats.AssertionPassRate - diff.BaseStats.AssertionPassRate

	fmt.Printf("             Base        Head        Change\n")
	fmt.Printf("Tasks:       %d/%-8d %d/%-8d ",
		diff.BaseStats.TasksPassed, diff.BaseStats.TasksTotal,
		diff.HeadStats.TasksPassed, diff.HeadStats.TasksTotal)
	printChange(taskChange)

	fmt.Printf("Assertions:  %d/%-8d %d/%-8d ",
		diff.BaseStats.AssertionsPassed, diff.BaseStats.AssertionsTotal,
		diff.HeadStats.AssertionsPassed, diff.HeadStats.AssertionsTotal)
	printChange(assertionChange)
}

func printChange(change float64) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	if change > 0 {
		_, _ = green.Printf("+%.1f%%\n", change*100)
	} else if change < 0 {
		_, _ = red.Printf("%.1f%%\n", change*100)
	} else {
		fmt.Println("0.0%")
	}
}

func outputMarkdownDiff(diff DiffResult) {
	taskChange := diff.HeadStats.TaskPassRate - diff.BaseStats.TaskPassRate
	assertionChange := diff.HeadStats.AssertionPassRate - diff.BaseStats.AssertionPassRate

	fmt.Println("### 📊 Evaluation Results")
	fmt.Println()
	fmt.Println("| Metric | Base | Head | Change |")
	fmt.Println("|--------|------|------|--------|")
	fmt.Printf("| Tasks | %d/%d (%.1f%%) | %d/%d (%.1f%%) | %s |\n",
		diff.BaseStats.TasksPassed, diff.BaseStats.TasksTotal, diff.BaseStats.TaskPassRate*100,
		diff.HeadStats.TasksPassed, diff.HeadStats.TasksTotal, diff.HeadStats.TaskPassRate*100,
		formatChangeMarkdown(taskChange))
	fmt.Printf("| Assertions | %d/%d (%.1f%%) | %d/%d (%.1f%%) | %s |\n",
		diff.BaseStats.AssertionsPassed, diff.BaseStats.AssertionsTotal, diff.BaseStats.AssertionPassRate*100,
		diff.HeadStats.AssertionsPassed, diff.HeadStats.AssertionsTotal, diff.HeadStats.AssertionPassRate*100,
		formatChangeMarkdown(assertionChange))

	// Regressions
	if len(diff.Regressions) > 0 {
		fmt.Println()
		fmt.Printf("#### ❌ Regressions (%d)\n", len(diff.Regressions))
		for _, r := range diff.Regressions {
			fmt.Printf("- `%s`: PASSED → FAILED", r.TaskName)
			if r.FailureReason != "" {
				fmt.Printf(" - %s", r.FailureReason)
			}
			fmt.Println()
		}
	}

	// Improvements
	if len(diff.Improvements) > 0 {
		fmt.Println()
		fmt.Printf("#### ✅ Improvements (%d)\n", len(diff.Improvements))
		for _, r := range diff.Improvements {
			fmt.Printf("- `%s`: FAILED → PASSED\n", r.TaskName)
		}
	}

	// New tasks
	if len(diff.New) > 0 {
		fmt.Println()
		fmt.Printf("#### 🆕 New Tasks (%d)\n", len(diff.New))
		for _, r := range diff.New {
			status := "PASSED"
			if !r.HeadPassed {
				status = "FAILED"
			}
			fmt.Printf("- `%s`: %s\n", r.TaskName, status)
		}
	}

	// Removed tasks
	if len(diff.Removed) > 0 {
		fmt.Println()
		fmt.Printf("#### 🗑️ Removed Tasks (%d)\n", len(diff.Removed))
		for _, r := range diff.Removed {
			fmt.Printf("- `%s`\n", r.TaskName)
		}
	}
}

func formatChangeMarkdown(change float64) string {
	if change > 0 {
		return fmt.Sprintf("🟢 +%.1f%%", change*100)
	} else if change < 0 {
		return fmt.Sprintf("🔴 %.1f%%", change*100)
	}
	return "➖ 0.0%"
}
