package testcase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/genmcp/gevals/e2e/servers/agent"
	"github.com/genmcp/gevals/pkg/eval"
	"github.com/genmcp/gevals/pkg/task"
)

// GeneratedFiles holds paths to all generated configuration files
type GeneratedFiles struct {
	TempDir       string
	TaskFile      string
	EvalFile      string
	MCPConfigFile string
	AgentConfig   string
	OutputFile    string
}

// Generator handles generating configuration files for a test case
type Generator struct {
	t       *testing.T
	tempDir string
}

// NewGenerator creates a new generator with a temporary directory
func NewGenerator(t *testing.T) (*Generator, error) {
	tempDir, err := os.MkdirTemp("", "gevals-e2e-*")
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return &Generator{
		t:       t,
		tempDir: tempDir,
	}, nil
}

// TempDir returns the temporary directory path
func (g *Generator) TempDir() string {
	return g.tempDir
}

// GenerateTaskYAML writes a task spec to a YAML file
func (g *Generator) GenerateTaskYAML(taskSpec *task.TaskSpec) (string, error) {
	// Wrap in kind structure for proper deserialization
	wrapper := map[string]any{
		"kind":     task.KindTask,
		"metadata": taskSpec.Metadata,
		"steps":    g.buildTaskSteps(taskSpec),
	}

	return g.writeYAML("task.yaml", wrapper)
}

func (g *Generator) buildTaskSteps(taskSpec *task.TaskSpec) map[string]any {
	steps := make(map[string]any)

	if taskSpec == nil {
		return steps
	}

	if taskSpec.Steps.Prompt != nil {
		if taskSpec.Steps.Prompt.Inline != "" {
			steps["prompt"] = map[string]any{"inline": taskSpec.Steps.Prompt.Inline}
		} else if taskSpec.Steps.Prompt.File != "" {
			steps["prompt"] = map[string]any{"file": taskSpec.Steps.Prompt.File}
		}
	}

	if taskSpec.Steps.SetupScript != nil {
		if taskSpec.Steps.SetupScript.Inline != "" {
			steps["setup"] = map[string]any{"inline": taskSpec.Steps.SetupScript.Inline}
		} else if taskSpec.Steps.SetupScript.File != "" {
			steps["setup"] = map[string]any{"file": taskSpec.Steps.SetupScript.File}
		}
	}

	if taskSpec.Steps.CleanupScript != nil {
		if taskSpec.Steps.CleanupScript.Inline != "" {
			steps["cleanup"] = map[string]any{"inline": taskSpec.Steps.CleanupScript.Inline}
		} else if taskSpec.Steps.CleanupScript.File != "" {
			steps["cleanup"] = map[string]any{"file": taskSpec.Steps.CleanupScript.File}
		}
	}

	if taskSpec.Steps.VerifyScript != nil {
		verify := make(map[string]any)
		if taskSpec.Steps.VerifyScript.Step != nil {
			if taskSpec.Steps.VerifyScript.Step.Inline != "" {
				verify["inline"] = taskSpec.Steps.VerifyScript.Step.Inline
			} else if taskSpec.Steps.VerifyScript.Step.File != "" {
				verify["file"] = taskSpec.Steps.VerifyScript.Step.File
			}
		}
		if taskSpec.Steps.VerifyScript.LLMJudgeTaskConfig != nil {
			if taskSpec.Steps.VerifyScript.LLMJudgeTaskConfig.Contains != "" {
				verify["contains"] = taskSpec.Steps.VerifyScript.LLMJudgeTaskConfig.Contains
			}
			if taskSpec.Steps.VerifyScript.LLMJudgeTaskConfig.Exact != "" {
				verify["exact"] = taskSpec.Steps.VerifyScript.LLMJudgeTaskConfig.Exact
			}
		}
		if len(verify) > 0 {
			steps["verify"] = verify
		}
	}

	return steps
}

// GenerateEvalYAML writes an eval spec to a YAML file
func (g *Generator) GenerateEvalYAML(evalSpec *eval.EvalSpec) (string, error) {
	// Wrap in kind structure for proper deserialization
	wrapper := map[string]any{
		"kind":     eval.KindEval,
		"metadata": evalSpec.Metadata,
		"config":   evalSpec.Config,
	}

	return g.writeYAML("eval.yaml", wrapper)
}

// GenerateMCPConfigJSON writes an MCP server configuration to a JSON file.
// This generates the config format expected by the agent (mcpServers map).
func (g *Generator) GenerateMCPConfigJSON(servers map[string]string) (string, error) {
	config := map[string]any{
		"mcpServers": make(map[string]any),
	}

	for name, url := range servers {
		config["mcpServers"].(map[string]any)[name] = map[string]any{
			"url": url,
		}
	}

	return g.writeJSON("mcp-config.json", config)
}

// GenerateAgentConfigJSON writes a mock agent configuration to a JSON file
func (g *Generator) GenerateAgentConfigJSON(agentConfig *agent.Config) (string, error) {
	return g.writeJSON("agent-config.json", agentConfig)
}

// writeYAML writes data as YAML to a file in the temp directory
func (g *Generator) writeYAML(filename string, data any) (string, error) {
	path := filepath.Join(g.tempDir, filename)

	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, yamlBytes, 0644); err != nil {
		return "", err
	}

	return path, nil
}

// writeJSON writes data as JSON to a file in the temp directory
func (g *Generator) writeJSON(filename string, data any) (string, error) {
	path := filepath.Join(g.tempDir, filename)

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return "", err
	}

	return path, nil
}

// ReadEvalResults reads and parses the eval output JSON file
func ReadEvalResults(path string) ([]*eval.EvalResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []*eval.EvalResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// WriteFile writes content to a file in the temp directory
func (g *Generator) WriteFile(filename, content string) (string, error) {
	path := filepath.Join(g.tempDir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// Mkdir creates a subdirectory in the temp directory
func (g *Generator) Mkdir(name string) (string, error) {
	path := filepath.Join(g.tempDir, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}
