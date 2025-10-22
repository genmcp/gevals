package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/genmcp/gevals/pkg/eval"
	"github.com/spf13/cobra"
)

// NewRunCmd creates the run command
func NewRunCmd() *cobra.Command {
	var outputFormat string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "run [eval-config-file]",
		Short: "Run an evaluation",
		Long:  `Run an evaluation using the specified eval configuration file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]

			// Load eval spec
			spec, err := eval.FromFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to load eval config: %w", err)
			}

			// Create runner
			runner, err := eval.NewRunner(spec)
			if err != nil {
				return fmt.Errorf("failed to create eval runner: %w", err)
			}

			// Create progress display
			display := newProgressDisplay(verbose)

			// Run with progress
			ctx := context.Background()
			results, err := runner.RunWithProgress(ctx, display.handleProgress)
			if err != nil {
				return fmt.Errorf("eval failed: %w", err)
			}

			// Save results to JSON file
			outputFile := fmt.Sprintf("gevals-%s-out.json", spec.Metadata.Name)
			if err := saveResultsToFile(results, outputFile); err != nil {
				return fmt.Errorf("failed to save results to file: %w", err)
			}
			fmt.Printf("\n📄 Results saved to: %s\n", outputFile)

			// Display results
			if err := displayResults(results, outputFormat); err != nil {
				return fmt.Errorf("failed to display results: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	return cmd
}

// progressDisplay handles interactive progress display
type progressDisplay struct {
	verbose bool
	green   *color.Color
	red     *color.Color
	yellow  *color.Color
	cyan    *color.Color
	bold    *color.Color
}

func newProgressDisplay(verbose bool) *progressDisplay {
	return &progressDisplay{
		verbose: verbose,
		green:   color.New(color.FgGreen),
		red:     color.New(color.FgRed),
		yellow:  color.New(color.FgYellow),
		cyan:    color.New(color.FgCyan),
		bold:    color.New(color.Bold),
	}
}

func (d *progressDisplay) handleProgress(event eval.ProgressEvent) {
	switch event.Type {
	case eval.EventEvalStart:
		d.bold.Println("\n=== Starting Evaluation ===")

	case eval.EventTaskStart:
		fmt.Println()
		d.cyan.Printf("Task: %s\n", event.Task.TaskName)
		if event.Task.Difficulty != "" {
			fmt.Printf("  Difficulty: %s\n", event.Task.Difficulty)
		}

	case eval.EventTaskSetup:
		if d.verbose {
			fmt.Printf("  → Setting up task environment...\n")
		}

	case eval.EventTaskRunning:
		fmt.Printf("  → Running agent...\n")

	case eval.EventTaskVerifying:
		fmt.Printf("  → Verifying results...\n")

	case eval.EventTaskAssertions:
		if d.verbose {
			fmt.Printf("  → Evaluating assertions...\n")
		}

	case eval.EventTaskComplete:
		task := event.Task
		if task.TaskPassed && task.AllAssertionsPassed {
			d.green.Printf("  ✓ Task passed\n")
		} else if task.TaskPassed && !task.AllAssertionsPassed {
			d.yellow.Printf("  ~ Task passed but assertions failed\n")
		} else {
			if task.AgentExecutionError {
				d.red.Printf("  ✗ Agent failed to run\n")
				if task.TaskError != "" || task.TaskOutput != "" {
					errorFile, err := saveErrorToFile(task.TaskName, task.TaskError, task.TaskOutput)
					if err != nil {
						// If we can't save to file, fall back to printing inline
						fmt.Printf("    Error: %s\n", task.TaskError)
					} else {
						fmt.Printf("    Error details saved to: %s\n", errorFile)
					}
				}
			} else {
				d.red.Printf("  ✗ Task failed\n")
				if task.TaskError != "" {
					fmt.Printf("    Error: %s\n", task.TaskError)
				}
			}
		}

	case eval.EventEvalComplete:
		fmt.Println()
		d.bold.Println("=== Evaluation Complete ===")
	}
}

func displayResults(results []*eval.EvalResult, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)

	case "text":
		return displayTextResults(results)

	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

func displayTextResults(results []*eval.EvalResult) error {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	bold := color.New(color.Bold)

	fmt.Println()
	bold.Println("=== Results Summary ===")
	fmt.Println()

	totalTasks := len(results)
	tasksPassed := 0
	totalAssertions := 0
	passedAssertions := 0

	for _, result := range results {
		if result.TaskPassed {
			tasksPassed++
		}

		// Count individual assertions
		if result.AssertionResults != nil {
			totalAssertions += result.AssertionResults.TotalAssertions()
			passedAssertions += result.AssertionResults.PassedAssertions()
		}

		// Display individual result
		fmt.Printf("Task: %s\n", result.TaskName)
		fmt.Printf("  Path: %s\n", result.TaskPath)
		if result.Difficulty != "" {
			fmt.Printf("  Difficulty: %s\n", result.Difficulty)
		}

		if result.TaskPassed {
			green.Printf("  Task Status: PASSED\n")
		} else {
			if result.AgentExecutionError {
				red.Printf("  Task Status: FAILED (Agent execution error)\n")
				if result.TaskError != "" || result.TaskOutput != "" {
					errorFile, err := saveErrorToFile(result.TaskName, result.TaskError, result.TaskOutput)
					if err != nil {
						// If we can't save to file, fall back to printing inline
						fmt.Printf("  Error: %s\n", result.TaskError)
					} else {
						fmt.Printf("  Error details saved to: %s\n", errorFile)
					}
				}
			} else {
				red.Printf("  Task Status: FAILED\n")
				if result.TaskError != "" {
					fmt.Printf("  Error: %s\n", result.TaskError)
				}
			}
		}

		if result.AssertionResults != nil {
			passed := result.AssertionResults.PassedAssertions()
			total := result.AssertionResults.TotalAssertions()
			if result.AllAssertionsPassed {
				green.Printf("  Assertions: PASSED (%d/%d)\n", passed, total)
			} else {
				yellow.Printf("  Assertions: FAILED (%d/%d)\n", passed, total)
				printFailedAssertions(result.AssertionResults)
			}
		}

		fmt.Println()
	}

	bold.Println("=== Overall Statistics ===")
	fmt.Printf("Total Tasks: %d\n", totalTasks)

	if tasksPassed == totalTasks {
		green.Printf("Tasks Passed: %d/%d\n", tasksPassed, totalTasks)
	} else {
		fmt.Printf("Tasks Passed: %d/%d\n", tasksPassed, totalTasks)
	}

	if totalAssertions > 0 {
		if passedAssertions == totalAssertions {
			green.Printf("Assertions Passed: %d/%d\n", passedAssertions, totalAssertions)
		} else {
			fmt.Printf("Assertions Passed: %d/%d\n", passedAssertions, totalAssertions)
		}
	}

	// Group by difficulty
	fmt.Println()
	bold.Println("=== Statistics by Difficulty ===")
	displayStatsByDifficulty(results, green)

	return nil
}

func displayStatsByDifficulty(results []*eval.EvalResult, green *color.Color) {
	// Group results by difficulty
	type difficultyStats struct {
		totalTasks       int
		tasksPassed      int
		totalAssertions  int
		passedAssertions int
	}

	statsByDifficulty := make(map[string]*difficultyStats)

	for _, result := range results {
		difficulty := result.Difficulty
		if difficulty == "" {
			difficulty = "unspecified"
		}

		if statsByDifficulty[difficulty] == nil {
			statsByDifficulty[difficulty] = &difficultyStats{}
		}

		stats := statsByDifficulty[difficulty]
		stats.totalTasks++

		if result.TaskPassed {
			stats.tasksPassed++
		}

		if result.AssertionResults != nil {
			stats.totalAssertions += result.AssertionResults.TotalAssertions()
			stats.passedAssertions += result.AssertionResults.PassedAssertions()
		}
	}

	// Display stats in order: easy, medium, hard, then any others
	orderedDifficulties := []string{"easy", "medium", "hard"}

	for _, difficulty := range orderedDifficulties {
		stats, exists := statsByDifficulty[difficulty]
		if !exists {
			continue
		}

		fmt.Printf("\n%s:\n", difficulty)

		if stats.tasksPassed == stats.totalTasks {
			green.Printf("  Tasks: %d/%d\n", stats.tasksPassed, stats.totalTasks)
		} else {
			fmt.Printf("  Tasks: %d/%d\n", stats.tasksPassed, stats.totalTasks)
		}

		if stats.totalAssertions > 0 {
			if stats.passedAssertions == stats.totalAssertions {
				green.Printf("  Assertions: %d/%d\n", stats.passedAssertions, stats.totalAssertions)
			} else {
				fmt.Printf("  Assertions: %d/%d\n", stats.passedAssertions, stats.totalAssertions)
			}
		}
	}

	// Display any other difficulties (e.g., "unspecified") that weren't in the main list
	for difficulty, stats := range statsByDifficulty {
		isStandard := false
		for _, d := range orderedDifficulties {
			if d == difficulty {
				isStandard = true
				break
			}
		}
		if isStandard {
			continue
		}

		fmt.Printf("\n%s:\n", difficulty)

		if stats.tasksPassed == stats.totalTasks {
			green.Printf("  Tasks: %d/%d\n", stats.tasksPassed, stats.totalTasks)
		} else {
			fmt.Printf("  Tasks: %d/%d\n", stats.tasksPassed, stats.totalTasks)
		}

		if stats.totalAssertions > 0 {
			if stats.passedAssertions == stats.totalAssertions {
				green.Printf("  Assertions: %d/%d\n", stats.passedAssertions, stats.totalAssertions)
			} else {
				fmt.Printf("  Assertions: %d/%d\n", stats.passedAssertions, stats.totalAssertions)
			}
		}
	}
}

func printFailedAssertions(results *eval.CompositeAssertionResult) {
	printSingleAssertion("ToolsUsed", results.ToolsUsed)
	printSingleAssertion("RequireAny", results.RequireAny)
	printSingleAssertion("ToolsNotUsed", results.ToolsNotUsed)
	printSingleAssertion("MinToolCalls", results.MinToolCalls)
	printSingleAssertion("MaxToolCalls", results.MaxToolCalls)
	printSingleAssertion("ResourcesRead", results.ResourcesRead)
	printSingleAssertion("ResourcesNotRead", results.ResourcesNotRead)
	printSingleAssertion("PromptsUsed", results.PromptsUsed)
	printSingleAssertion("PromptsNotUsed", results.PromptsNotUsed)
	printSingleAssertion("CallOrder", results.CallOrder)
	printSingleAssertion("NoDuplicateCalls", results.NoDuplicateCalls)
}

func printSingleAssertion(name string, result *eval.SingleAssertionResult) {
	if result != nil && !result.Passed {
		fmt.Printf("    - %s: %s\n", name, result.Reason)
		for _, detail := range result.Details {
			fmt.Printf("      %s\n", detail)
		}
	}
}

func saveResultsToFile(results []*eval.EvalResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}

	return nil
}

// saveErrorToFile saves task error and output to a file and returns the filename
func saveErrorToFile(taskName, taskError, taskOutput string) (string, error) {
	// Create a safe filename from task name
	safeTaskName := strings.ReplaceAll(taskName, "/", "-")
	safeTaskName = strings.ReplaceAll(safeTaskName, " ", "-")
	filename := fmt.Sprintf("%s-error.txt", safeTaskName)

	content := ""
	if taskError != "" {
		content += fmt.Sprintf("=== Error ===\n%s\n", taskError)
	}
	if taskOutput != "" {
		content += fmt.Sprintf("\n=== Output ===\n%s\n", taskOutput)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write error file: %w", err)
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filename, nil // Return relative path if we can't get absolute
	}

	return absPath, nil
}

