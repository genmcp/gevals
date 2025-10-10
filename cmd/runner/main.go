package main

import (
	"fmt"
	"os"

	"github.com/genmcp/gevals/pkg/eval"
	"github.com/spf13/cobra"
)

var (
	agentCmd string
	taskFile string
)

var rootCmd = &cobra.Command{
	Use:   "runner",
	Short: "Evaluate agent CLI performance on tasks",
	Long: `runner is a framework for evaluating how well agent CLI binaries perform on defined tasks.
Tasks include prompts, optional setup/verifier/cleanup scripts, and expected output patterns.`,
	Example: `  runner --agent "my-agent '{{.Prompt}}'" --task task.yaml
  runner --agent "agent-cli --prompt '{{.Prompt}}'" --task tasks/create-pod.yaml`,
	RunE: runEvaluation,
}

func init() {
	// Required flags
	rootCmd.Flags().StringVar(&agentCmd, "agent", "", "Agent command template (use {{.Prompt}} for prompt placeholder)")
	rootCmd.Flags().StringVar(&taskFile, "task", "", "Path to task YAML file")

	// Mark required flags
	rootCmd.MarkFlagRequired("agent")
	//	rootCmd.MarkFlagRequired("task")
}

func runEvaluation(cmd *cobra.Command, args []string) error {
	// Load task
	task, err := eval.LoadTask(taskFile)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Create evaluator
	evaluator := eval.NewEvaluator(agentCmd)

	// Run evaluation
	fmt.Printf("Running evaluation...\n")
	fmt.Printf("Task: %s\n", task.Prompt)
	fmt.Printf("Difficulty: %s\n\n", task.Difficulty)

	result := evaluator.Evaluate(task)

	// Print results
	printResult(result)

	if !result.Success {
		return fmt.Errorf("evaluation failed")
	}

	return nil
}

func printResult(r *eval.Result) {
	fmt.Println("=== Evaluation Results ===")
	fmt.Printf("Setup:        %s\n", statusStr(r.SetupSuccess))
	fmt.Printf("Agent:        %s\n", statusStr(r.AgentSuccess))
	fmt.Printf("Expectations: %s\n", statusStr(r.ExpectationsMet))
	fmt.Printf("Verifier:     %s\n", statusStr(r.VerifierSuccess))
	fmt.Printf("Cleanup:      %s\n", statusStr(r.CleanupSuccess))
	fmt.Printf("Overall:      %s\n", statusStr(r.Success))

	if r.Error != nil {
		fmt.Printf("\nError: %v\n", r.Error)
	}

	if r.AgentOutput != "" {
		fmt.Printf("\n=== Agent Output ===\n%s\n", r.AgentOutput)
	}
}

func statusStr(success bool) string {
	if success {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
