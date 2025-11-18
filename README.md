# gevals

🧪 Test your MCP servers by having AI agents complete real tasks.

## What It Does

gevals validates MCP servers by:
1. 🔧 Running setup scripts (e.g., create test namespace)
2. 🤖 Giving an AI agent a task prompt (e.g., "create a nginx pod")
3. 📝 Recording which MCP tools the agent uses
4. ✅ Verifying the task succeeded via scripts OR LLM judge (e.g., pod is running, or response contains expected content)
5. 🔍 Checking assertions (e.g., did agent call `pods_create`?)
6. 🧹 Running cleanup scripts

If agents successfully complete tasks using your MCP server, your tools are well-designed.

## Quick Start

```bash
# Build
go build -o gevals ./cmd/gevals

# Run the example (requires Kubernetes cluster + MCP server)
./gevals eval examples/kubernetes/eval.yaml
```

The tool will:
- Display progress in real-time
- Save results to `gevals-<name>-out.json`
- Show pass/fail summary

## Example Setup

**eval.yaml** - Main config:
```yaml
kind: Eval
metadata:
  name: "kubernetes-test"
config:
  agentFile: agent.yaml           # How to run your AI agent
  mcpConfigFile: mcp-config.yaml  # Your MCP server config
  llmJudge:                        # Optional: LLM judge for semantic verification
    env:
      baseUrlKey: JUDGE_BASE_URL   # Env var name for LLM API base URL
      apiKeyKey: JUDGE_API_KEY     # Env var name for LLM API key
      modelNameKey: JUDGE_MODEL_NAME # Env var name for model name
  taskSets:
    - path: tasks/create-pod.yaml
      assertions:
        toolsUsed:
          - server: kubernetes
            toolPattern: "pods_.*"  # Agent must use pod-related tools
        minToolCalls: 1
        maxToolCalls: 10
```

**mcp-config.yaml** - MCP server to test:
```yaml
mcpServers:
  kubernetes:
    type: http
    url: http://localhost:8080/mcp
    enableAllTools: true
```

**agent.yaml** - AI agent configuration:
```yaml
kind: Agent
metadata:
  name: "claude-code"
commands:
  useVirtualHome: false
  argTemplateMcpServer: "--mcp-config {{ .File }}"
  argTemplateAllowedTools: "mcp__{{ .ServerName }}__{{ .ToolName }}"
  allowedToolsJoinSeparator: ","
  runPrompt: |-
    claude {{ .McpServerFileArgs }} --strict-mcp-config --allowedTools "{{ .AllowedToolArgs }}" --print "{{ .Prompt }}"
```

**tasks/create-pod.yaml** - Test task:
```yaml
kind: Task
metadata:
  name: "create-nginx-pod"
  difficulty: easy
steps:
  setup:
    file: setup.sh      # Creates test namespace
  verify:
    file: verify.sh     # Script-based: Checks pod is running
    # OR use LLM judge (requires llmJudge config in eval.yaml):
    # contains: "pod is running"  # Semantic check: response contains this text
    # exact: "The pod web-server is running"  # Semantic check: exact match
  cleanup:
    file: cleanup.sh    # Deletes pod
  prompt:
    inline: Create a nginx pod named web-server in the test-namespace
```

Note: You must choose either script-based verification (`file` or `inline`) OR LLM judge verification (`contains` or `exact`), not both.

## Assertions

Validate agent behavior:

```yaml
assertions:
  # Must call these tools
  toolsUsed:
    - server: kubernetes
      tool: pods_create              # Exact tool name
    - server: kubernetes
      toolPattern: "pods_.*"         # Regex pattern

  # Must call at least one of these
  requireAny:
    - server: kubernetes
      tool: pods_create

  # Must NOT call these
  toolsNotUsed:
    - server: kubernetes
      tool: namespaces_delete

  # Call limits
  minToolCalls: 1
  maxToolCalls: 10

  # Resource access
  resourcesRead:
    - server: filesystem
      uriPattern: "/data/.*\\.json$"
  resourcesNotRead:
    - server: filesystem
      uri: /etc/secrets/password

  # Prompt usage
  promptsUsed:
    - server: templates
      prompt: deployment-template

  # Call order (can have other calls between)
  callOrder:
    - type: tool
      server: kubernetes
      name: namespaces_create
    - type: tool
      server: kubernetes
      name: pods_create

  # No duplicate calls
  noDuplicateCalls: true
```

## Test Scripts

Scripts return exit 0 for success, non-zero for failure:

**setup.sh** - Prepare environment:
```bash
#!/usr/bin/env bash
kubectl create namespace test-ns
```

**verify.sh** - Check task succeeded:
```bash
#!/usr/bin/env bash
kubectl wait --for=condition=Ready pod/web-server -n test-ns --timeout=120s
```

**cleanup.sh** - Remove resources:
```bash
#!/usr/bin/env bash
kubectl delete pod web-server -n test-ns
```

Or use inline scripts in the task YAML:
```yaml
steps:
  setup:
    inline: |-
      #!/usr/bin/env bash
      kubectl create namespace test-ns
```

## LLM Judge Verification

Instead of script-based verification, you can use an LLM judge to semantically evaluate agent responses. This is useful when:
- You want to check if the agent's response contains specific information (semantic matching)
- The expected output format may vary but the meaning should be consistent
- You're testing tasks where the agent provides text responses rather than performing actions

### Configuration

First, configure the LLM judge in your `eval.yaml`:

```yaml
config:
  llmJudge:
    env:
      baseUrlKey: JUDGE_BASE_URL    # Environment variable for LLM API base URL
      apiKeyKey: JUDGE_API_KEY      # Environment variable for LLM API key
      modelNameKey: JUDGE_MODEL_NAME # Environment variable for model name
```

Set the required environment variables before running:
```bash
export JUDGE_BASE_URL="https://api.openai.com/v1"
export JUDGE_API_KEY="sk-..."
export JUDGE_MODEL_NAME="gpt-4o"
```

**Note**: The LLM judge currently only supports OpenAI-compatible APIs (APIs that follow the OpenAI API format). The implementation uses the OpenAI Go SDK with a configurable base URL, so you can use any OpenAI-compatible endpoint, but APIs with different formats are not supported.

### Evaluation Modes

The LLM judge supports two evaluation modes:

**CONTAINS mode** (`verify.contains`):
- Checks if the agent's response semantically contains all core information from the reference answer
- Extra, correct, and non-contradictory information is acceptable
- Format and phrasing differences are ignored (semantic matching)
- Use when you want to verify the response includes specific information

**EXACT mode** (`verify.exact`):
- Checks if the agent's response is semantically equivalent to the reference answer
- Simple rephrasing is acceptable (e.g., "Paris is the capital" vs "The capital is Paris")
- Adding or omitting information will fail
- Use when you need precise semantic equivalence

**Note**: Both modes use the same LLM-based semantic evaluation approach. The difference is only in the system prompt instructions given to the judge LLM. See [`pkg/llmjudge/prompts.go`](pkg/llmjudge/prompts.go) for the implementation details.

### Usage in Tasks

In your task YAML, use `verify.contains` or `verify.exact` instead of `verify.file` or `verify.inline`:

```yaml
steps:
  verify:
    contains: "mysql:8.0.36"  # Response must contain this information
```

```yaml
steps:
  verify:
    exact: "The pod web-server is running in namespace test-ns"  # Response must match exactly (semantically)
```

**Important**: You cannot use both script-based verification and LLM judge verification in the same task. Choose one method:
- Script-based: `verify.file` or `verify.inline` (runs a script that returns exit code 0 for success)
- LLM judge: `verify.contains` or `verify.exact` (semantically evaluates the agent's text response)

## Results

Pass/fail means:

**✅ Pass** → Your MCP server is well-designed
- Tools are discoverable
- Descriptions are clear
- Schemas work
- Implementation is correct

**❌ Fail** → Needs improvement
- Tool descriptions unclear
- Schema too complex
- Missing functionality
- Implementation bugs

## Output

Results saved to `gevals-<eval-name>-out.json`:

```json
{
  "taskName": "create-nginx-pod",
  "taskPassed": true,
  "allAssertionsPassed": true,
  "assertionResults": {
    "toolsUsed": { "passed": true },
    "minToolCalls": { "passed": true }
  },
  "callHistory": {
    "toolCalls": [
      {
        "serverName": "kubernetes",
        "toolName": "pods_create",
        "timestamp": "2025-01-15T10:30:00Z"
      }
    ]
  }
}
```

## How It Works

The tool creates an MCP proxy that sits between the AI agent and your MCP server:

```
AI Agent → MCP Proxy (recording) → Your MCP Server
```

Everything gets recorded:
- Which tools were called
- What arguments were passed
- When calls happened
- What responses came back

Then assertions validate the recorded behavior matches your expectations.
