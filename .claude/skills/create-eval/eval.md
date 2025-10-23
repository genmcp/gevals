# Eval Configuration Reference

Eval configuration files tie together agents, MCP servers, and tasks to create complete evaluation suites.

## Eval YAML Structure

```yaml
kind: Eval
metadata:
  name: "eval-name"
config:
  agentFile: agent.yaml
  mcpConfigFile: mcp-config.yaml
  taskSets:
    - path: tasks/task1.yaml
      assertions:
        toolsUsed: [...]
        minToolCalls: 1
    - glob: "tasks/**/*.yaml"
      assertions: {...}
```

## Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | Yes | Must be `"Eval"` |
| `metadata` | object | Yes | Eval metadata (see below) |
| `config` | object | Yes | Eval configuration (see below) |

## metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for the eval |

## config Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `agentFile` | string | Yes | Path to agent YAML file |
| `mcpConfigFile` | string | Yes | Path to MCP config file |
| `taskSets` | array | Yes | List of task sets with optional assertions |

## taskSets Array Items

Each item in `taskSets` has these fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | One required* | Path to single task file |
| `glob` | string | One required* | Glob pattern for multiple tasks |
| `assertions` | object | No | Assertions for this task set (see below) |

\* Exactly one of `path` or `glob` must be set

## assertions Object

All fields are optional. If not specified, that assertion is not checked.

### Tool Assertions

| Field | Type | Description |
|-------|------|-------------|
| `toolsUsed` | array | All listed tools MUST be called |
| `requireAny` | array | At least ONE listed tool must be called |
| `toolsNotUsed` | array | NONE of the listed tools can be called |
| `minToolCalls` | integer | Minimum number of total tool calls |
| `maxToolCalls` | integer | Maximum number of total tool calls |

### Resource Assertions

| Field | Type | Description |
|-------|------|-------------|
| `resourcesRead` | array | All listed resources MUST be read |
| `resourcesNotRead` | array | NONE of the listed resources can be read |

### Prompt Assertions

| Field | Type | Description |
|-------|------|-------------|
| `promptsUsed` | array | All listed prompts MUST be used |
| `promptsNotUsed` | array | NONE of the listed prompts can be used |

### Order Assertions

| Field | Type | Description |
|-------|------|-------------|
| `callOrder` | array | Calls must occur in specified order (not necessarily consecutive) |

### Efficiency Assertions

| Field | Type | Description |
|-------|------|-------------|
| `noDuplicateCalls` | boolean | Prevent duplicate tool calls with identical arguments |

## Tool Assertion Object

Each item in `toolsUsed`, `requireAny`, `toolsNotUsed`:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `server` | string | Yes | MCP server name |
| `tool` | string | No* | Exact tool name |
| `toolPattern` | string | No* | Regex pattern for tool name |

\* If neither `tool` nor `toolPattern` is set, matches ANY tool from the server

## Resource Assertion Object

Each item in `resourcesRead`, `resourcesNotRead`:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `server` | string | Yes | MCP server name |
| `uri` | string | No* | Exact resource URI |
| `uriPattern` | string | No* | Regex pattern for URI |

\* If neither `uri` nor `uriPattern` is set, matches ANY resource from the server

## Prompt Assertion Object

Each item in `promptsUsed`, `promptsNotUsed`:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `server` | string | Yes | MCP server name |
| `prompt` | string | No* | Exact prompt name |
| `promptPattern` | string | No* | Regex pattern for prompt name |

\* If neither `prompt` nor `promptPattern` is set, matches ANY prompt from the server

## Call Order Assertion Object

Each item in `callOrder`:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | One of: `"tool"`, `"resource"`, `"prompt"` |
| `server` | string | Yes | MCP server name |
| `name` | string | Yes | Tool/resource/prompt name |

## Complete Examples

### Simple Eval

```yaml
kind: Eval
metadata:
  name: "kubernetes-basic"
config:
  agentFile: agent.yaml
  mcpConfigFile: mcp-config.yaml
  taskSets:
    - path: tasks/create-pod.yaml
      assertions:
        toolsUsed:
          - server: kubernetes
            toolPattern: "pods_.*"
        minToolCalls: 1
        maxToolCalls: 5
```

### Multi-Task with Different Assertions

```yaml
kind: Eval
metadata:
  name: "comprehensive-eval"
config:
  agentFile: agents/claude-code.yaml
  mcpConfigFile: configs/prod-mcp.yaml
  taskSets:
    # Easy tasks: limited tool calls
    - glob: tasks/easy/**/*.yaml
      assertions:
        maxToolCalls: 3
        noDuplicateCalls: true

    # Medium tasks: specific tools required
    - glob: tasks/medium/**/*.yaml
      assertions:
        toolsUsed:
          - server: kubernetes
            toolPattern: "pods_.*"
        minToolCalls: 2
        maxToolCalls: 10

    # Hard tasks: complex requirements
    - glob: tasks/hard/**/*.yaml
      assertions:
        callOrder:
          - type: tool
            server: kubernetes
            name: pods_list
          - type: tool
            server: kubernetes
            name: pods_create
        noDuplicateCalls: true
```

### Strict Eval with Forbidden Actions

```yaml
kind: Eval
metadata:
  name: "safe-operations"
config:
  agentFile: agent.yaml
  mcpConfigFile: mcp-config.yaml
  taskSets:
    - glob: tasks/**/*.yaml
      assertions:
        # Must use list before create
        callOrder:
          - type: tool
            server: kubernetes
            name: pods_list
          - type: tool
            server: kubernetes
            name: pods_create

        # Cannot delete anything
        toolsNotUsed:
          - server: kubernetes
            toolPattern: ".*_delete"

        # Cannot read secrets
        resourcesNotRead:
          - server: kubernetes
            uriPattern: ".*secrets.*"

        # Efficiency requirements
        maxToolCalls: 15
        noDuplicateCalls: true
```

### Flexible Eval

```yaml
kind: Eval
metadata:
  name: "flexible-eval"
config:
  agentFile: agent.yaml
  mcpConfigFile: mcp-config.yaml
  taskSets:
    - path: tasks/creative-task.yaml
      assertions:
        # Must use at least one kubernetes tool (any tool)
        requireAny:
          - server: kubernetes
            # No tool/toolPattern = matches any tool from server
        maxToolCalls: 20
```

## Assertion Behavior Notes

### Pattern Matching
- `tool: "pods_create"` → Exact match only
- `toolPattern: "pods_.*"` → Regex match (pods_create, pods_list, etc.)
- Neither specified → Matches ANY tool from the server

### Call Order
`callOrder` requires calls in relative order, not consecutive. Other calls can happen between them.

Example: `[pods_list, pods_create]` passes for:
- ✅ `pods_list → deployments_list → pods_create`
- ❌ `pods_create → pods_list`
