//go:build functional

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcpchecker/mcpchecker/functional/testcase"
)

// TestLabelFiltering_NoSelector runs 3 labeled tasks and ensures all are executed when no selector is provided.
func TestLabelFiltering_NoSelector(t *testing.T) {
	tasksDir := writeLabelFilteringTasks(t)
	glob := filepath.Join(tasksDir, "*.yaml")

	testcase.New(t, "label-filtering-no-selector").
		WithMCPServer("kubernetes", func(s *testcase.MCPServerBuilder) {
			s.Tool("noop", func(tool *testcase.ToolDef) {
				tool.WithDescription("No-op tool").ReturnsText("ok")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnAnyPrompt().ThenRespond("ok")
		}).
		WithEval(func(ec *testcase.EvalConfig) {
			ec.Name("label-filtering-no-selector").
				TaskSet(func(ts *testcase.TaskSetBuilder) {
					ts.Glob(glob)
				})
		}).
		Expect(&testcase.TaskCountAssertion{Expected: 3}).
		Expect(testcase.AssertFunc("output contains Total Tasks: 3", func(t *testing.T, ctx *testcase.RunContext) {
			if ctx == nil {
				t.Fatalf("nil run context")
			}
			if ctx.CommandOutput == "" {
				t.Fatalf("empty command output")
			}
			if !contains(ctx.CommandOutput, "Total Tasks: 3") {
				t.Fatalf("expected command output to contain %q, got:\n%s", "Total Tasks: 3", ctx.CommandOutput)
			}
		})).
		Expect(testcase.AssertFunc("results include all tasks", func(t *testing.T, ctx *testcase.RunContext) {
			for _, name := range []string{"k8s-basic-task", "k8s-advanced-task", "istio-task"} {
				if ctx.ResultForTask(name) == nil {
					t.Fatalf("expected result for task %q", name)
				}
			}
		})).
		Run()
}

// TestLabelFiltering_KubernetesBasicSelector ensures only the kubernetes/basic task is selected via TaskSet labelSelector.
func TestLabelFiltering_KubernetesBasicSelector(t *testing.T) {
	tasksDir := writeLabelFilteringTasks(t)
	glob := filepath.Join(tasksDir, "*.yaml")

	testcase.New(t, "label-filtering-kubernetes-basic-selector").
		WithMCPServer("kubernetes", func(s *testcase.MCPServerBuilder) {
			s.Tool("noop", func(tool *testcase.ToolDef) {
				tool.WithDescription("No-op tool").ReturnsText("ok")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnAnyPrompt().ThenRespond("ok")
		}).
		WithEval(func(ec *testcase.EvalConfig) {
			ec.Name("label-filtering-kubernetes-basic-selector").
				TaskSet(func(ts *testcase.TaskSetBuilder) {
					ts.Glob(glob).LabelSelector(map[string]string{
						"suite":    "kubernetes",
						"category": "basic",
					})
				})
		}).
		Expect(&testcase.TaskCountAssertion{Expected: 1}).
		Expect(testcase.AssertFunc("only k8s-basic-task executed", func(t *testing.T, ctx *testcase.RunContext) {
			if ctx.ResultForTask("k8s-basic-task") == nil {
				t.Fatalf("expected k8s-basic-task to be executed")
			}
			if ctx.ResultForTask("k8s-advanced-task") != nil {
				t.Fatalf("did not expect k8s-advanced-task to be executed")
			}
			if ctx.ResultForTask("istio-task") != nil {
				t.Fatalf("did not expect istio-task to be executed")
			}
		})).
		Run()
}

// TestLabelFiltering_KubernetesSelector ensures both kubernetes tasks are selected when using only suite=kubernetes selector.
func TestLabelFiltering_KubernetesSelector(t *testing.T) {
	tasksDir := writeLabelFilteringTasks(t)
	glob := filepath.Join(tasksDir, "*.yaml")

	testcase.New(t, "label-filtering-kubernetes-selector").
		WithMCPServer("kubernetes", func(s *testcase.MCPServerBuilder) {
			s.Tool("noop", func(tool *testcase.ToolDef) {
				tool.WithDescription("No-op tool").ReturnsText("ok")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnAnyPrompt().ThenRespond("ok")
		}).
		WithEval(func(ec *testcase.EvalConfig) {
			ec.Name("label-filtering-kubernetes-selector").
				TaskSet(func(ts *testcase.TaskSetBuilder) {
					ts.Glob(glob).LabelSelector(map[string]string{
						"suite": "kubernetes",
					})
				})
		}).
		Expect(&testcase.TaskCountAssertion{Expected: 2}).
		Expect(testcase.AssertFunc("output contains Total Tasks: 2", func(t *testing.T, ctx *testcase.RunContext) {
			if ctx == nil {
				t.Fatalf("nil run context")
			}
			if ctx.CommandOutput == "" {
				t.Fatalf("empty command output")
			}
			if !contains(ctx.CommandOutput, "Total Tasks: 2") {
				t.Fatalf("expected command output to contain %q, got:\n%s", "Total Tasks: 2", ctx.CommandOutput)
			}
		})).
		Expect(testcase.AssertFunc("both kubernetes tasks executed", func(t *testing.T, ctx *testcase.RunContext) {
			if ctx.ResultForTask("k8s-basic-task") == nil {
				t.Fatalf("expected k8s-basic-task to be executed")
			}
			if ctx.ResultForTask("k8s-advanced-task") == nil {
				t.Fatalf("expected k8s-advanced-task to be executed")
			}
			if ctx.ResultForTask("istio-task") != nil {
				t.Fatalf("did not expect istio-task to be executed")
			}
		})).
		Run()
}

// TestLabelFiltering_IstioSelector ensures only the istio task is selected via TaskSet labelSelector.
func TestLabelFiltering_IstioSelector(t *testing.T) {
	tasksDir := writeLabelFilteringTasks(t)
	glob := filepath.Join(tasksDir, "*.yaml")

	testcase.New(t, "label-filtering-istio-selector").
		WithMCPServer("kubernetes", func(s *testcase.MCPServerBuilder) {
			s.Tool("noop", func(tool *testcase.ToolDef) {
				tool.WithDescription("No-op tool").ReturnsText("ok")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnAnyPrompt().ThenRespond("ok")
		}).
		WithEval(func(ec *testcase.EvalConfig) {
			ec.Name("label-filtering-istio-selector").
				TaskSet(func(ts *testcase.TaskSetBuilder) {
					ts.Glob(glob).LabelSelector(map[string]string{
						"suite": "istio",
					})
				})
		}).
		Expect(&testcase.TaskCountAssertion{Expected: 1}).
		Expect(testcase.AssertFunc("only istio-task executed", func(t *testing.T, ctx *testcase.RunContext) {
			if ctx.ResultForTask("istio-task") == nil {
				t.Fatalf("expected istio-task to be executed")
			}
			if ctx.ResultForTask("k8s-basic-task") != nil {
				t.Fatalf("did not expect k8s-basic-task to be executed")
			}
			if ctx.ResultForTask("k8s-advanced-task") != nil {
				t.Fatalf("did not expect k8s-advanced-task to be executed")
			}
		})).
		Run()
}

func writeLabelFilteringTasks(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("failed to create tasks dir: %v", err)
	}

	writeFile(t, filepath.Join(tasksDir, "task1.yaml"), `
apiVersion: mcpchecker/v1alpha2
kind: Task
metadata:
  name: k8s-basic-task
  difficulty: easy
  labels:
    suite: kubernetes
    category: basic
spec:
  prompt:
    inline: "kubernetes basic task"
  verify:
    - script:
        inline: "exit 0"
`)

	writeFile(t, filepath.Join(tasksDir, "task2.yaml"), `
apiVersion: mcpchecker/v1alpha2
kind: Task
metadata:
  name: k8s-advanced-task
  difficulty: hard
  labels:
    suite: kubernetes
    category: advanced
spec:
  prompt:
    inline: "kubernetes advanced task"
  verify:
    - script:
        inline: "exit 0"
`)

	writeFile(t, filepath.Join(tasksDir, "task3.yaml"), `
apiVersion: mcpchecker/v1alpha2
kind: Task
metadata:
  name: istio-task
  difficulty: medium
  labels:
    suite: istio
spec:
  prompt:
    inline: "istio task"
  verify:
    - script:
        inline: "exit 0"
`)

	return tasksDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
