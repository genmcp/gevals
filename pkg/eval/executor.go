package eval

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// Evaluator runs task evaluations against an agent binary
type Evaluator struct {
	// AgentCommand is a template for the agent CLI command
	// Use {{.Prompt}} as a placeholder for the prompt text
	// Example: "agent-binary --prompt '{{.Prompt}}'"
	AgentCommand string
}

// NewEvaluator creates a new evaluator
func NewEvaluator(agentCommand string) *Evaluator {
	return &Evaluator{
		AgentCommand: agentCommand,
	}
}

// Evaluate runs a single task evaluation
func (e *Evaluator) Evaluate(task *Task) *Result {
	result := &Result{
		Task: task,
	}

	// Run setup script if provided
	if task.Setup != "" {
		if err := e.runScript(task.Setup); err != nil {
			result.SetupSuccess = false
			result.Error = fmt.Errorf("setup failed: %w", err)
			return result
		}
		result.SetupSuccess = true
	} else {
		result.SetupSuccess = true // No setup means success
	}

	// Execute agent with prompt
	output, err := e.runAgent(task.Prompt)
	result.AgentOutput = output
	if err != nil {
		result.AgentSuccess = false
		result.Error = fmt.Errorf("agent execution failed: %w", err)
		e.cleanup(task)
		return result
	}
	result.AgentSuccess = true

	// Check expectations
	result.ExpectationsMet = e.checkExpectations(output, task.Expect)

	// Run verifier script if provided
	if task.Verifier != "" {
		if err := e.runScript(task.Verifier); err != nil {
			result.VerifierSuccess = false
			result.Error = fmt.Errorf("verifier failed: %w", err)
			e.cleanup(task)
			return result
		}
		result.VerifierSuccess = true
	} else {
		result.VerifierSuccess = true // No verifier means success
	}

	// Cleanup
	result.CleanupSuccess = e.cleanup(task) == nil

	// Overall success
	result.Success = result.SetupSuccess &&
		result.AgentSuccess &&
		result.ExpectationsMet &&
		result.VerifierSuccess

	return result
}

// runScript executes a shell script
func (e *Evaluator) runScript(scriptPath string) error {
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runAgent executes the agent binary with a prompt using the command template
func (e *Evaluator) runAgent(prompt string) (string, error) {
	// Parse and execute template
	tmpl, err := template.New("agent").Parse(e.AgentCommand)
	if err != nil {
		return "", fmt.Errorf("failed to parse agent command template: %w", err)
	}

	var cmdStr bytes.Buffer
	if err := tmpl.Execute(&cmdStr, map[string]string{"Prompt": prompt}); err != nil {
		return "", fmt.Errorf("failed to execute agent command template: %w", err)
	}

	// Execute command via shell to handle complex arguments
	cmd := exec.Command("bash", "-c", cmdStr.String())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String() + stderr.String(), err
	}

	return stdout.String(), nil
}

// checkExpectations validates that output matches expected patterns
func (e *Evaluator) checkExpectations(output string, expectations []Expectation) bool {
	if len(expectations) == 0 {
		return true // No expectations means success
	}

	for _, expect := range expectations {
		if expect.Contains != "" {
			matched, err := regexp.MatchString(expect.Contains, output)
			if err != nil || !matched {
				return false
			}
		}
	}

	return true
}

// cleanup runs the cleanup script
func (e *Evaluator) cleanup(task *Task) error {
	if task.Cleanup == "" {
		return nil
	}
	return e.runScript(task.Cleanup)
}

// escapeShellArg escapes a string for safe use in shell commands
func escapeShellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
