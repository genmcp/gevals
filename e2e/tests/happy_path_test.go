package tests

import (
	"testing"

	"github.com/genmcp/gevals/e2e/testcase"
)

// TestTaskPassesWithToolCallAndJudge verifies the happy path where:
// - Agent receives a prompt
// - Agent calls an MCP tool
// - Tool returns a successful result
// - LLM judge evaluates the output and passes
func TestTaskPassesWithToolCallAndJudge(t *testing.T) {
	testcase.New(t, "task-passes-with-tool-and-judge").
		WithMCPServer("kubernetes", func(s *testcase.MCPServerBuilder) {
			s.Tool("kubectl_apply", func(tool *testcase.ToolDef) {
				tool.WithDescription("Apply a Kubernetes manifest").
					WithStringParam("manifest", "YAML manifest content", true).
					ReturnsText("pod/nginx-web created")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnPromptContaining("nginx").
				CallTool("kubectl_apply", map[string]any{
					"manifest": "apiVersion: v1\nkind: Pod\nmetadata:\n  name: nginx-web",
				}).
				ThenRespond("I created the nginx pod named nginx-web successfully.")
		}).
		WithTask(func(task *testcase.TaskConfig) {
			task.Name("create-nginx-pod").
				Easy().
				Prompt("Create an nginx pod named nginx-web").
				VerifyContains("nginx-web")
		}).
		WithEval(func(eval *testcase.EvalConfig) {
			eval.Name("test-eval")
		}).
		WithJudge(func(j *testcase.JudgeBuilder) {
			j.Always().Pass("The output correctly describes creation of the nginx-web pod")
		}).
		ExpectTaskPassed().
		ExpectToolCalled("kubernetes", "kubectl_apply").
		ExpectJudgeCalled().
		Run()
}

// TestTaskPassesWithScriptVerification verifies the happy path where:
// - Agent receives a prompt
// - Agent calls an MCP tool
// - Verification is done via script (no LLM judge)
func TestTaskPassesWithScriptVerification(t *testing.T) {
	testcase.New(t, "task-passes-with-script-verify").
		WithMCPServer("filesystem", func(s *testcase.MCPServerBuilder) {
			s.Tool("write_file", func(tool *testcase.ToolDef) {
				tool.WithDescription("Write content to a file").
					WithStringParam("path", "File path", true).
					WithStringParam("content", "File content", true).
					ReturnsText("File written successfully")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnPromptContaining("hello.txt").
				CallTool("write_file", map[string]any{
					"path":    "/tmp/hello.txt",
					"content": "Hello, World!",
				}).
				ThenRespond("I wrote 'Hello, World!' to /tmp/hello.txt")
		}).
		WithTask(func(task *testcase.TaskConfig) {
			task.Name("write-hello-file").
				Easy().
				Prompt("Write 'Hello, World!' to a file called hello.txt").
				// Script verification - exit 0 means pass
				VerifyScript("echo 'pass'")
		}).
		WithEval(func(eval *testcase.EvalConfig) {
			eval.Name("test-eval-script")
		}).
		ExpectTaskPassed().
		ExpectToolCalled("filesystem", "write_file").
		ExpectJudgeNotCalled().
		Run()
}

// TestTaskPassesWithMultipleToolCalls verifies that:
// - Agent can make multiple tool calls in sequence
// - All tool calls are captured and verifiable
func TestTaskPassesWithMultipleToolCalls(t *testing.T) {
	testcase.New(t, "task-passes-with-multiple-tools").
		WithMCPServer("kubernetes", func(s *testcase.MCPServerBuilder) {
			s.Tool("kubectl_get", func(tool *testcase.ToolDef) {
				tool.WithDescription("Get Kubernetes resources").
					WithStringParam("resource", "Resource type", true).
					ReturnsText("NAME      READY   STATUS    RESTARTS   AGE\nnginx     1/1     Running   0          5m")
			}).
			Tool("kubectl_apply", func(tool *testcase.ToolDef) {
				tool.WithDescription("Apply a Kubernetes manifest").
					WithStringParam("manifest", "YAML manifest content", true).
					ReturnsText("service/nginx-svc created")
			})
		}).
		WithAgent(func(a *testcase.AgentBuilder) {
			a.OnPromptContaining("expose").
				CallTool("kubectl_get", map[string]any{
					"resource": "pods",
				}).
				CallTool("kubectl_apply", map[string]any{
					"manifest": "apiVersion: v1\nkind: Service\nmetadata:\n  name: nginx-svc",
				}).
				ThenRespond("I found the nginx pod and created a service nginx-svc to expose it.")
		}).
		WithTask(func(task *testcase.TaskConfig) {
			task.Name("expose-nginx").
				Medium().
				Prompt("Find the nginx pod and expose it with a service").
				VerifyContains("nginx-svc")
		}).
		WithEval(func(eval *testcase.EvalConfig) {
			eval.Name("test-eval-multi-tool")
		}).
		WithJudge(func(j *testcase.JudgeBuilder) {
			j.Always().Pass("Agent correctly found the pod and created a service")
		}).
		ExpectTaskPassed().
		ExpectToolCalled("kubernetes", "kubectl_get").
		ExpectToolCalled("kubernetes", "kubectl_apply").
		ExpectToolCalledTimes("kubernetes", "kubectl_get", 1).
		ExpectToolCalledTimes("kubernetes", "kubectl_apply", 1).
		Run()
}
