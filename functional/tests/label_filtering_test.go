//go:build functional

package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/genmcp/gevals/functional/testcase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// TestLabelFiltering creates multiple tasks and verifies label-based filtering works end-to-end
func TestLabelFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "label-filtering-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tasksDir := filepath.Join(tmpDir, "tasks")
	require.NoError(t, os.MkdirAll(tasksDir, 0755))

	// Create task 1: kubernetes + basic (SHOULD MATCH)
	task1 := map[string]any{
		"apiVersion": "gevals/v1alpha2",
		"kind":       "Task",
		"metadata": map[string]any{
			"name":       "k8s-basic-task",
			"difficulty": "easy",
			"labels": map[string]string{
				"suite":    "kubernetes",
				"category": "basic",
			},
		},
		"spec": map[string]any{
			"prompt": map[string]any{
				"inline": "kubernetes basic task",
			},
			"verify": []map[string]any{
				{
					"script": map[string]any{
						"inline": "exit 0",
					},
				},
			},
		},
	}
	writeTaskFile(t, tasksDir, "task1.yaml", task1)

	// Create task 2: kubernetes + advanced (should NOT match)
	task2 := map[string]any{
		"apiVersion": "gevals/v1alpha2",
		"kind":       "Task",
		"metadata": map[string]any{
			"name":       "k8s-advanced-task",
			"difficulty": "hard",
			"labels": map[string]string{
				"suite":    "kubernetes",
				"category": "advanced",
			},
		},
		"spec": map[string]any{
			"prompt": map[string]any{
				"inline": "kubernetes advanced task",
			},
			"verify": []map[string]any{
				{
					"script": map[string]any{
						"inline": "exit 0",
					},
				},
			},
		},
	}
	writeTaskFile(t, tasksDir, "task2.yaml", task2)

	// Create task 3: istio (should NOT match)
	task3 := map[string]any{
		"apiVersion": "gevals/v1alpha2",
		"kind":       "Task",
		"metadata": map[string]any{
			"name":       "istio-task",
			"difficulty": "medium",
			"labels": map[string]string{
				"suite": "istio",
			},
		},
		"spec": map[string]any{
			"prompt": map[string]any{
				"inline": "istio task",
			},
			"verify": []map[string]any{
				{
					"script": map[string]any{
						"inline": "exit 0",
					},
				},
			},
		},
	}
	writeTaskFile(t, tasksDir, "task3.yaml", task3)

	gevalsBinary, err := testcase.GetGevalsBinary()
	require.NoError(t, err)

	// STEP 1: Run WITHOUT label selector - should execute ALL 3 tasks
	t.Log("=== Testing WITHOUT label selector (should execute 3 tasks) ===")

	evalConfigNoFilter := map[string]any{
		"kind": "Eval",
		"metadata": map[string]any{
			"name": "no-filter-test",
		},
		"config": map[string]any{
			"taskSets": []map[string]any{
				{
					"glob": filepath.Join(tasksDir, "*.yaml"),
					// NO labelSelector - should run all tasks
				},
			},
			"agent": map[string]any{
				"type": "builtin.claude-code",
			},
			"mcpConfigFile": createEmptyMCPConfig(t, tmpDir),
		},
	}

	evalNoFilterBytes, err := yaml.Marshal(evalConfigNoFilter)
	require.NoError(t, err)
	evalNoFilterFile := filepath.Join(tmpDir, "eval-no-filter.yaml")
	require.NoError(t, os.WriteFile(evalNoFilterFile, evalNoFilterBytes, 0644))

	cmdNoFilter := exec.Command(gevalsBinary, "eval", evalNoFilterFile)
	cmdNoFilter.Dir = tmpDir
	outputNoFilter, err := cmdNoFilter.CombinedOutput()
	require.NoError(t, err, "Output file should exist: %s", outputNoFilter)
	if err != nil {
		t.Fatalf("gevals eval command failed (no filter): %v\nOutput:\n%s", err, string(outputNoFilter))
	}

	t.Logf("gevals output (no filter):\n%s", string(outputNoFilter))

	// Verify all 3 tasks were executed
	outputNoFilterStr := string(outputNoFilter)
	assert.Contains(t, outputNoFilterStr, "Total Tasks: 3", "Should process all 3 tasks without filter")
	assert.Contains(t, outputNoFilterStr, "k8s-basic-task", "Should include k8s-basic-task")
	assert.Contains(t, outputNoFilterStr, "k8s-advanced-task", "Should include k8s-advanced-task")
	assert.Contains(t, outputNoFilterStr, "istio-task", "Should include istio-task")

	// Verify results file contains 3 tasks
	outputNoFilterFile := filepath.Join(tmpDir, "gevals-no-filter-test-out.json")
	_, err = os.Stat(outputNoFilterFile)
	require.NoError(t, err, "Output file should exist: %s", outputNoFilterFile)

	data, err := os.ReadFile(outputNoFilterFile)
	require.NoError(t, err)

	var results []map[string]any
	require.NoError(t, json.Unmarshal(data, &results))

	assert.Len(t, results, 3, "Results should contain all 3 tasks without filter")

	// STEP 2: Run WITH label selector - should execute ONLY 1 task
	t.Log("=== Testing WITH label selector (should execute 1 task) ===")

	evalConfigWithFilter := map[string]any{
		"kind": "Eval",
		"metadata": map[string]any{
			"name": "label-filtering-test",
		},
		"config": map[string]any{
			"taskSets": []map[string]any{
				{
					"glob": filepath.Join(tasksDir, "*.yaml"),
					"labelSelector": map[string]string{
						"suite":    "kubernetes",
						"category": "basic",
					},
				},
			},
			"agent": map[string]any{
				"type": "builtin.claude-code",
			},
			"mcpConfigFile": createEmptyMCPConfig(t, tmpDir),
		},
	}

	evalWithFilterBytes, err := yaml.Marshal(evalConfigWithFilter)
	require.NoError(t, err)
	evalWithFilterFile := filepath.Join(tmpDir, "eval-with-filter.yaml")
	require.NoError(t, os.WriteFile(evalWithFilterFile, evalWithFilterBytes, 0644))

	cmdWithFilter := exec.Command(gevalsBinary, "eval", evalWithFilterFile)
	cmdWithFilter.Dir = tmpDir
	outputWithFilter, err := cmdWithFilter.CombinedOutput()
	if err != nil {
		t.Fatalf("gevals eval command failed (with filter): %v\nOutput:\n%s", err, string(outputWithFilter))
	}

	t.Logf("gevals output (with filter):\n%s", string(outputWithFilter))

	// Verify filtering worked by checking output
	outputWithFilterStr := string(outputWithFilter)

	// Should see k8s-basic-task being executed
	assert.Contains(t, outputWithFilterStr, "k8s-basic-task", "Should execute k8s-basic-task")
	assert.Contains(t, outputWithFilterStr, "Total Tasks: 1", "Should only process 1 task with filter")

	// Should NOT see the other tasks
	assert.NotContains(t, outputWithFilterStr, "k8s-advanced-task", "Should NOT execute k8s-advanced-task")
	assert.NotContains(t, outputWithFilterStr, "istio-task", "Should NOT execute istio-task")

	// Verify results file exists and contains exactly 1 task
	outputWithFilterFile := filepath.Join(tmpDir, "gevals-label-filtering-test-out.json")
	_, err = os.Stat(outputWithFilterFile)
	require.NoError(t, err, "Output file should exist: %s", outputWithFilterFile)

	data, err = os.ReadFile(outputWithFilterFile)
	require.NoError(t, err)

	results = nil // Reset the slice
	require.NoError(t, json.Unmarshal(data, &results))

	// Should only have 1 result (the filtered task)
	assert.Len(t, results, 1, "Results should contain exactly 1 task (k8s-basic-task) with filter")
}

// writeTaskFile writes a task configuration to a YAML file in the specified directory
func writeTaskFile(t *testing.T, dir, filename string, task map[string]any) {
	t.Helper()
	taskBytes, err := yaml.Marshal(task)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), taskBytes, 0644))
}

// createEmptyMCPConfig creates an empty MCP server configuration file and returns its path
func createEmptyMCPConfig(t *testing.T, dir string) string {
	t.Helper()
	mcpConfig := map[string]any{
		"mcpServers": map[string]any{},
	}
	mcpBytes, err := json.Marshal(mcpConfig)
	require.NoError(t, err)
	mcpFile := filepath.Join(dir, "mcp-config.json")
	require.NoError(t, os.WriteFile(mcpFile, mcpBytes, 0644))
	return mcpFile
}
