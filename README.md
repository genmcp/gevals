# gevals

🧪 Test your MCP servers by having AI agents complete real tasks.

## What It Does

gevals validates MCP servers by:
1. 🔧 Running setup scripts (e.g., create test namespace)
2. 🤖 Giving an AI agent a task prompt (e.g., "create a nginx pod")
3. 📝 Recording which MCP tools the agent uses
4. ✅ Verifying the task succeeded (e.g., pod is running)
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
  # Option 1: Reference an agent file
  agentFile: agent.yaml           # How to run your AI agent

  # Option 2: Inline builtin agent (no separate file needed)
  # agent:
  #   type: "claude-code"
  # Or for OpenAI-compatible agents:
  # agent:
  #   type: "openai-agent"
  #   model: "gpt-4"

  mcpConfigFile: mcp-config.yaml  # Your MCP server config
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
builtin:
  type: "claude-code"  # Use built-in Claude Code configuration
```

Or with OpenAI-compatible agents:
```yaml
kind: Agent
metadata:
  name: "my-agent"
builtin:
  type: "openai-agent"
  model: "gpt-4"
# Set these environment variables:
# export MODEL_BASE_URL="https://api.openai.com/v1"
# export MODEL_KEY="sk-..."
```

For custom configurations, specify the `commands` section manually (see "Agent Configuration" below).

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
    file: verify.sh     # Checks pod is running
  cleanup:
    file: cleanup.sh    # Deletes pod
  prompt:
    inline: Create a nginx pod named web-server in the test-namespace
```

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

## Agent Configuration

### Inline vs File-based Configuration

You can configure agents in two ways:

1. **Inline in eval.yaml** (recommended for built-in agents):
```yaml
kind: Eval
config:
  agent:
    type: "claude-code"
```

2. **Separate agent.yaml file**:
```yaml
kind: Eval
config:
  agentFile: agent.yaml
```

Use inline configuration for simple setups with built-in agents. Use a separate file when you need custom commands or want to reuse the same agent across multiple evals.

### Built-in Agent Types

gevals provides built-in configurations for popular AI agents to eliminate boilerplate:

**Claude Code** (inline):
```yaml
kind: Eval
config:
  agent:
    type: "claude-code"
```

**Claude Code** (file-based):
```yaml
kind: Agent
metadata:
  name: "claude-code"
builtin:
  type: "claude-code"
```

**OpenAI-compatible agents** (inline):
```yaml
kind: Eval
config:
  agent:
    type: "openai-agent"
    model: "gpt-4"  # or any OpenAI-compatible model
```

**OpenAI-compatible agents** (file-based):
```yaml
kind: Agent
metadata:
  name: "openai-agent"
builtin:
  type: "openai-agent"
  model: "gpt-4"  # or any OpenAI-compatible model
```

Set environment variables for API access:
```bash
# Generic environment variables used by all OpenAI-compatible models
export MODEL_BASE_URL="https://api.openai.com/v1"
export MODEL_KEY="sk-..."

# For other providers (e.g., granite, custom endpoints):
# export MODEL_BASE_URL="https://your-endpoint/v1"
# export MODEL_KEY="your-key"
```

### Available Built-in Types

- `claude-code` - Anthropic's Claude Code CLI
- `openai-agent` - OpenAI-compatible agents using direct API calls (requires model)

### Custom Agent Configuration

For custom setups, specify the `commands` section:

```yaml
kind: Agent
metadata:
  name: "custom-agent"
commands:
  useVirtualHome: false
  argTemplateMcpServer: "--mcp {{ .File }}"
  argTemplateAllowedTools: "{{ .ToolName }}"
  runPrompt: |-
    my-agent --mcp-config {{ .McpServerFileArgs }} --prompt "{{ .Prompt }}"
```

### Overriding Built-in Defaults

You can use a built-in type and override specific settings:

```yaml
kind: Agent
metadata:
  name: "claude-custom"
builtin:
  type: "claude-code"
commands:
  useVirtualHome: true  # Override just this setting
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
