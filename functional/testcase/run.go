package testcase

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/genmcp/gevals/functional/servers/agent"
	"github.com/genmcp/gevals/functional/servers/mcp"
	"github.com/genmcp/gevals/functional/servers/openai"
)

// Environment variables for binary paths
const (
	EnvGevalsBinary    = "GEVALS_BINARY"
	EnvMockAgentBinary = "MOCK_AGENT_BINARY"
)

// Runner orchestrates the execution of a test case
type Runner struct {
	tc *TestCase
	t  *testing.T

	// Runtime state
	generator   *Generator
	mcpServers  map[string]*mcp.MockMCPServer
	judgeServer *openai.MockOpenAIServer
	mcpURLs     map[string]string

	// Generated file paths
	taskFile      string
	evalFile      string
	mcpConfigFile string
	agentConfig   string
	outputFile    string
}

// Run executes the test case
func (r *Runner) Run() {
	r.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Setup
	if err := r.setup(ctx); err != nil {
		r.t.Fatalf("test setup failed: %v", err)
	}
	defer r.cleanup()

	// Generate configuration files
	if err := r.generateConfigs(); err != nil {
		r.t.Fatalf("config generation failed: %v", err)
	}

	// Run gevals
	runCtx := r.runGevals(ctx)

	// Run assertions
	r.runAssertions(runCtx)
}

func (r *Runner) setup(ctx context.Context) error {
	var err error

	// Create generator for temp files
	r.generator, err = NewGenerator(r.t)
	if err != nil {
		return err
	}

	// Start MCP mock servers
	r.mcpServers = make(map[string]*mcp.MockMCPServer)
	r.mcpURLs = make(map[string]string)

	for name, builder := range r.tc.mcpServers {
		server := builder.Build()
		url, err := server.Start()
		if err != nil {
			// Stop any already-started servers before returning
			r.stopStartedServers()
			return err
		}
		r.mcpServers[name] = server
		r.mcpURLs[name] = url
	}

	// Start judge mock server
	if r.tc.judgeMock != nil {
		r.judgeServer = r.tc.judgeMock.Build()
		_, err := r.judgeServer.Start()
		if err != nil {
			// Stop any already-started MCP servers before returning
			r.stopStartedServers()
			return err
		}
	}

	return nil
}

// stopStartedServers stops all servers that have been started during setup.
// This is called when setup fails partway through to clean up resources.
func (r *Runner) stopStartedServers() {
	// Stop MCP servers
	for _, server := range r.mcpServers {
		if server != nil {
			server.Stop()
		}
	}

	// Stop judge server if it was started
	if r.judgeServer != nil {
		r.judgeServer.Stop()
	}
}

func (r *Runner) cleanup() {
	// Stop MCP servers
	for _, server := range r.mcpServers {
		server.Stop()
	}

	// Stop judge server
	if r.judgeServer != nil {
		r.judgeServer.Stop()
	}
}

func (r *Runner) generateConfigs() error {
	var err error

	// Generate task YAML if task config is provided
	if r.tc.task != nil {
		r.taskFile, err = r.generator.GenerateTaskYAML(r.tc.task)
		if err != nil {
			return err
		}
	}

	// Generate MCP config JSON
	r.mcpConfigFile, err = r.generator.GenerateMCPConfigJSON(r.mcpURLs)
	if err != nil {
		return err
	}

	// Generate agent config JSON if mock agent is configured
	if r.tc.agentMock != nil {
		r.agentConfig, err = r.generator.GenerateAgentConfigJSON(r.tc.agentMock.Build())
		if err != nil {
			return err
		}
	}

	// Generate eval YAML
	if r.tc.eval != nil {
		evalSpec := r.tc.eval.Build()

		// Update eval config with generated file paths
		evalSpec.Config.McpConfigFile = r.mcpConfigFile

		// Set up agent reference if mock agent is configured
		if r.tc.agentMock != nil {
			agentSpecFile, err := r.generateAgentSpecFile()
			if err != nil {
				return err
			}
			evalSpec.Config.Agent = &AgentRef{
				Type: "file",
				Path: agentSpecFile,
			}
		}

		// Set up LLM judge to point to mock server via environment variables
		if r.judgeServer != nil {
			if evalSpec.Config.LLMJudge == nil {
				evalSpec.Config.LLMJudge = &LLMJudgeEvalConfig{}
			}
			// Use custom environment variable keys for the mock server
			if evalSpec.Config.LLMJudge.Env == nil {
				evalSpec.Config.LLMJudge.Env = &LLMJudgeEnvConfig{}
			}
			evalSpec.Config.LLMJudge.Env.BaseUrlKey = "E2E_OPENAI_BASE_URL"
			evalSpec.Config.LLMJudge.Env.ApiKeyKey = "E2E_OPENAI_API_KEY"
			evalSpec.Config.LLMJudge.Env.ModelNameKey = "E2E_OPENAI_MODEL"
		}

		// Add task to task sets if not already configured
		if len(evalSpec.Config.TaskSets) == 0 && r.taskFile != "" {
			evalSpec.Config.TaskSets = []TaskSet{{Path: r.taskFile}}
		}

		r.evalFile, err = r.generator.GenerateEvalYAML(evalSpec)
		if err != nil {
			return err
		}

		// Output file is gevals-{eval-name}-out.json in the temp directory
		// gevals writes to current working directory, so we run from temp dir
		evalName := evalSpec.Metadata.Name
		if evalName == "" {
			evalName = "eval"
		}
		r.outputFile = filepath.Join(r.generator.TempDir(), fmt.Sprintf("gevals-%s-out.json", evalName))
	}

	return nil
}

// generateAgentSpecFile creates an agent spec YAML that uses the mock agent binary.
// The agent spec follows gevals's expected format with commands templates.
func (r *Runner) generateAgentSpecFile() (string, error) {
	mockAgentBinary, err := GetMockAgentBinary()
	if err != nil {
		return "", err
	}

	// Create agent spec using gevals's commands template format.
	// The mock agent receives:
	// - Its behavior config via MOCK_AGENT_CONFIG env var (set by runner)
	// - MCP server config via --mcp-config flag (using gevals's per-server file template)
	// - The task prompt via --prompt flag
	//
	// Note: The mock agent expects a single MCP config file, but gevals provides
	// per-server files. The mock agent handles this by using the first server file.
	agentSpec := map[string]any{
		"kind": "Agent",
		"metadata": map[string]any{
			"name": "mock-agent",
		},
		"commands": map[string]any{
			// Template for MCP server file args - just the file path
			"argTemplateMcpServer": "{{ .File }}",
			// Template for allowed tools - format as server__tool
			"argTemplateAllowedTools": "{{ .ServerName }}__{{ .ToolName }}",
			// Full command template to run the mock agent
			"runPrompt": fmt.Sprintf("%s --mcp-config {{ .McpServerFileArgs }} --prompt '{{ .Prompt }}'", mockAgentBinary),
		},
	}

	return r.generator.writeYAML("agent-spec.yaml", agentSpec)
}

func (r *Runner) runGevals(ctx context.Context) *RunContext {
	if r.tc.IsInProcess() {
		return r.runGevalInProcess()
	}
	return r.runGevalSubprocess(ctx)
}

func (r *Runner) runGevalInProcess() *RunContext {
	runCtx := &RunContext{
		MCPServers:  r.mcpServers,
		JudgeServer: r.judgeServer,
	}

	// Build environment variables
	env := make(map[string]string)
	if r.agentConfig != "" {
		env[agent.EnvConfigPath] = r.agentConfig
	}
	// Set OpenAI API key to dummy value (we're using mock)
	env["OPENAI_API_KEY"] = "sk-mock-key"

	// Set mock OpenAI server environment variables if judge is configured
	if r.judgeServer != nil {
		env["E2E_OPENAI_BASE_URL"] = r.judgeServer.URL()
		env["E2E_OPENAI_API_KEY"] = "sk-mock-key"
		env["E2E_OPENAI_MODEL"] = "gpt-4"
	}

	// Build command line arguments
	args := []string{"eval", r.evalFile}

	// Execute in-process
	executor := &InProcessExecutor{
		Args: args,
		Dir:  r.generator.TempDir(),
		Env:  env,
	}
	result := executor.Execute()

	runCtx.CommandOutput = result.Stdout + result.Stderr
	runCtx.CommandError = result.Err
	runCtx.ExitCode = result.ExitCode

	// Log command output for debugging
	if result.Err != nil {
		r.t.Logf("gevals command failed (in-process): %v", result.Err)
		r.t.Logf("command output:\n%s", runCtx.CommandOutput)
	}

	// Parse eval results from output file
	if _, statErr := os.Stat(r.outputFile); statErr == nil {
		results, parseErr := ReadEvalResults(r.outputFile)
		if parseErr != nil {
			r.t.Logf("warning: failed to parse eval results: %v", parseErr)
		} else {
			runCtx.EvalResults = results
		}
	} else {
		r.t.Logf("output file not found: %s", r.outputFile)
		r.t.Logf("command output:\n%s", runCtx.CommandOutput)
	}

	return runCtx
}

func (r *Runner) runGevalSubprocess(ctx context.Context) *RunContext {
	runCtx := &RunContext{
		MCPServers:  r.mcpServers,
		JudgeServer: r.judgeServer,
	}

	// Find gevals binary
	gevalsBinary, err := GetGevalsBinary()
	if err != nil {
		r.t.Fatalf("failed to find gevals binary: %v", err)
	}

	// Build command - eval takes config file as positional argument
	args := []string{"eval", r.evalFile}
	cmd := exec.CommandContext(ctx, gevalsBinary, args...)

	// Run from temp directory so output file is written there
	cmd.Dir = r.generator.TempDir()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment
	cmd.Env = os.Environ()
	if r.agentConfig != "" {
		cmd.Env = append(cmd.Env, agent.EnvConfigPath+"="+r.agentConfig)
	}
	// Set OpenAI API key to dummy value (we're using mock)
	cmd.Env = append(cmd.Env, "OPENAI_API_KEY=sk-mock-key")

	// Set mock OpenAI server environment variables if judge is configured
	if r.judgeServer != nil {
		cmd.Env = append(cmd.Env, "E2E_OPENAI_BASE_URL="+r.judgeServer.URL())
		cmd.Env = append(cmd.Env, "E2E_OPENAI_API_KEY=sk-mock-key")
		cmd.Env = append(cmd.Env, "E2E_OPENAI_MODEL=gpt-4")
	}

	// Run command
	err = cmd.Run()
	runCtx.CommandOutput = stdout.String() + stderr.String()
	runCtx.CommandError = err

	if cmd.ProcessState != nil {
		runCtx.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Log command output for debugging
	if err != nil {
		r.t.Logf("gevals command failed: %v", err)
		r.t.Logf("command output:\n%s", runCtx.CommandOutput)
	}

	// Parse eval results from output file
	if _, statErr := os.Stat(r.outputFile); statErr == nil {
		results, parseErr := ReadEvalResults(r.outputFile)
		if parseErr != nil {
			r.t.Logf("warning: failed to parse eval results: %v", parseErr)
		} else {
			runCtx.EvalResults = results
		}
	} else {
		r.t.Logf("output file not found: %s", r.outputFile)
		r.t.Logf("command output:\n%s", runCtx.CommandOutput)
	}

	return runCtx
}

func (r *Runner) runAssertions(ctx *RunContext) {
	for _, assertion := range r.tc.assertions {
		assertion.Assert(r.t, ctx)
	}
}

// GetGevalsBinary returns the path to the gevals binary.
// It first checks the GEVALS_BINARY environment variable,
// then looks for the binary in common locations.
func GetGevalsBinary() (string, error) {
	// Check environment variable first
	if path := os.Getenv(EnvGevalsBinary); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("GEVALS_BINARY set to %q but file not found", path)
	}

	// Try common locations relative to working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(wd, "..", "..", "bin", "gevals"),    // from functional/testcase or functional/tests
		filepath.Join(wd, "..", "bin", "gevals"),          // from functional
		filepath.Join(wd, "bin", "gevals"),                // current dir
		filepath.Join(wd, "..", "..", "gevals"),           // repo root
		filepath.Join(wd, "..", "gevals"),                 // parent
		filepath.Join(wd, "gevals"),                       // current dir
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("gevals binary not found; set %s environment variable", EnvGevalsBinary)
}

// GetMockAgentBinary returns the path to the mock agent binary.
// It first checks the MOCK_AGENT_BINARY environment variable,
// then looks for the binary in common locations.
func GetMockAgentBinary() (string, error) {
	// Check environment variable first
	if path := os.Getenv(EnvMockAgentBinary); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("MOCK_AGENT_BINARY set to %q but file not found", path)
	}

	// Try common locations relative to working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(wd, "..", "..", "bin", "mock-agent"),    // from functional/testcase or functional/tests
		filepath.Join(wd, "..", "bin", "mock-agent"),          // from functional
		filepath.Join(wd, "bin", "mock-agent"),                // current dir
		filepath.Join(wd, "..", "..", "mock-agent"),           // repo root
		filepath.Join(wd, "..", "mock-agent"),                 // parent
		filepath.Join(wd, "mock-agent"),                       // current dir
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("mock-agent binary not found; set %s environment variable", EnvMockAgentBinary)
}
