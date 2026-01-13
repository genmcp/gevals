# gevals Extension Protocol Specification

**Version**: 0.0.1
**Status**: Draft

## Overview

This document specifies the communication protocol between gevals and extension binaries. Extensions provide domain-specific operations (e.g., Kubernetes resource management, database queries) that can be used in task setup, verification, and cleanup phases.

The protocol is based on JSON-RPC 2.0 over newline-delimited stdio.

## Transport

| Property | Value |
|----------|-------|
| Framing | Newline-delimited JSON (each message is one line terminated by `\n`) |
| Encoding | UTF-8 |
| Input | gevals writes to extension's stdin |
| Output | Extension writes to stdout |
| Stderr | Reserved for debug logs (not parsed by gevals) |

## Lifecycle

```
┌────────┐                              ┌─────────────┐
│ gevals │                              │  extension  │
└───┬────┘                              └──────┬──────┘
    │                                          │
    │──── spawn process ──────────────────────▶│
    │                                          │
    │──── initialize ─────────────────────────▶│
    │◀─── manifest ───────────────────────────│
    │                                          │
    │──── execute ────────────────────────────▶│
    │◀─── log (0..n) ─────────────────────────│
    │◀─── result ─────────────────────────────│
    │                                          │
    │     ... more execute requests ...        │
    │                                          │
    │──── shutdown ───────────────────────────▶│
    │◀─── ack ────────────────────────────────│
    │                                          │
    │◀─── process exits ──────────────────────│
```

1. gevals spawns the extension binary
2. gevals sends `initialize` request
3. Extension responds with its manifest (name, version, available operations)
4. gevals sends `execute` requests for operations
5. Extension may send `log` notifications during execution
6. Extension sends result for each execute request
7. gevals sends `shutdown` when done
8. Extension exits

## Messages

### Initialize

Sent once after spawn. Returns extension manifest.

#### Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "0.0.1",
    "config": {
      "namespace": "my-app"
    }
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `protocolVersion` | string | Yes | Protocol version gevals supports |
| `config` | object | No | Extension-specific configuration from eval spec |

#### Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "name": "kubernetes",
    "version": "1.2.0",
    "protocolVersion": "0.0.1",
    "description": "Kubernetes resource operations",
    "requires": [
      {"command": "kubectl"}
    ],
    "operations": {
      "apply": {
        "description": "Apply a Kubernetes manifest",
        "params": {
          "type": "object",
          "properties": {
            "file": {
              "type": "string",
              "description": "Path to manifest file"
            },
            "namespace": {
              "type": "string",
              "default": "default",
              "description": "Target namespace"
            }
          },
          "required": ["file"]
        }
      },
      "condition": {
        "description": "Verify a resource condition",
        "params": {
          "type": "object",
          "properties": {
            "resource": {
              "type": "string",
              "description": "Resource (e.g., pod/nginx)"
            },
            "conditionType": {
              "type": "string",
              "description": "Condition type (e.g., Ready)"
            },
            "status": {
              "type": "string",
              "default": "True",
              "description": "Expected status value"
            },
            "timeout": {
              "type": "string",
              "default": "60s",
              "description": "How long to wait (duration format)"
            }
          },
          "required": ["resource", "conditionType"]
        }
      }
    }
  }
}
```

##### Manifest Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Extension identifier |
| `version` | string | Yes | Semantic version |
| `protocolVersion` | string | Yes | Protocol version supported |
| `description` | string | No | Human-readable description |
| `requires` | array | No | System requirements |
| `operations` | object | Yes | Map of operation name to operation definition |

##### Requirement Object

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Binary that must be in PATH |

##### Operation Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | No | Human-readable description |
| `params` | object | Yes | JSON Schema defining the operation's parameters |

The `params` field must be a valid JSON Schema object. Use standard JSON Schema keywords like `type`, `properties`, `required`, `default`, etc. to define the expected arguments.

---

### Execute

Run an operation.

#### Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "execute",
  "params": {
    "operation": "apply",
    "args": {
      "file": "deployment.yaml",
      "namespace": "my-app"
    },
    "context": {
      "workdir": "/path/to/task",
      "phase": "setup",
      "env": {
        "KUBECONFIG": "/home/user/.kube/config"
      },
      "timeout": "5m"
    }
  }
}
```

##### Params Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `operation` | string | Yes | Operation name from manifest |
| `args` | object | Yes | Operation arguments |
| `context` | object | Yes | Execution context |

##### Context Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workdir` | string | Yes | Task directory (for resolving relative paths) |
| `phase` | string | Yes | One of: `"setup"`, `"verify"`, `"cleanup"` |
| `env` | object | No | Environment variables from task spec |
| `timeout` | string | No | Maximum execution time (duration format) |
| `agent` | object | No | Agent context (only present in verify phase) |

##### Agent Context Object

Present only when `phase` is `"verify"`:

| Field | Type | Description |
|-------|------|-------------|
| `prompt` | string | The prompt given to the agent |
| `output` | string | The agent's response |

```json
{
  "context": {
    "phase": "verify",
    "agent": {
      "prompt": "Create a pod named nginx in the default namespace",
      "output": "I've created the nginx pod using kubectl run..."
    }
  }
}
```

#### Success Response

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "success": true,
    "message": "Applied deployment.yaml to namespace my-app",
    "outputs": {
      "resourceVersion": "12345",
      "uid": "abc-123-def"
    }
  }
}
```

#### Failure Response

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "success": false,
    "message": "Condition not met",
    "error": "Timed out waiting for pod/nginx Ready=True (current: False)",
    "outputs": {
      "lastStatus": "False",
      "lastCheckTime": "2024-01-15T10:30:00Z"
    }
  }
}
```

##### Result Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `success` | boolean | Yes | Whether the operation succeeded |
| `message` | string | No | Human-readable summary |
| `error` | string | No | Error details (when `success` is false) |
| `outputs` | map[string]string | No | String key-value outputs for use in subsequent steps |

**Note**: Operation failures (e.g., pod not ready) use `result` with `success: false`. Protocol errors (e.g., unknown operation) use JSON-RPC `error` response.

---

### Log (Notification)

Progress updates during execution. This is a JSON-RPC notification (no `id`, no response expected).

```json
{
  "jsonrpc": "2.0",
  "method": "log",
  "params": {
    "level": "info",
    "message": "Waiting for deployment rollout..."
  }
}
```

```json
{
  "jsonrpc": "2.0",
  "method": "log",
  "params": {
    "level": "debug",
    "message": "Pod status check",
    "data": {
      "pod": "nginx-abc123",
      "phase": "ContainerCreating"
    }
  }
}
```

##### Params Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `level` | string | Yes | One of: `"debug"`, `"info"`, `"warn"`, `"error"` |
| `message` | string | Yes | Log message |
| `data` | object | No | Structured data for debugging |

gevals displays logs based on verbosity settings.

---

### Shutdown

Graceful termination request.

#### Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "shutdown",
  "params": {}
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {}
}
```

Extension should exit after sending response.

---

## Error Handling

### Protocol Errors

Use JSON-RPC error format for protocol-level failures:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "error": {
    "code": -32601,
    "message": "Unknown operation: appply"
  }
}
```

#### Standard Error Codes

| Code | Name | Description |
|------|------|-------------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid request | Missing required fields |
| -32601 | Method not found | Unknown method or operation |
| -32602 | Invalid params | Argument validation failed |
| -32603 | Internal error | Unexpected extension error |

#### Extension Error Codes

Reserved range: -32000 to -32099

| Code | Name | Description |
|------|------|-------------|
| -32000 | Operation failed | Generic operation failure |
| -32001 | Timeout | Operation exceeded timeout |
| -32002 | Requirement not met | Required command not found |

### Error Categories

| Scenario | Response Type | Example |
|----------|---------------|---------|
| Operation logic failure | `result` with `success: false` | Pod not ready after timeout |
| Unknown operation | `error` with code -32601 | `"appply"` instead of `"apply"` |
| Missing required arg | `error` with code -32602 | `file` arg not provided |
| Extension crash | Process exits non-zero | Panic, segfault |
| Invalid JSON input | `error` with code -32700 | Malformed JSON |

---

## One-Shot Mode

For testing and simple usage, extensions support one-shot mode where a single execute request is sent without initialize/shutdown:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"execute","params":{"operation":"apply","args":{"file":"x.yaml"},"context":{"workdir":"/tmp","phase":"setup"}}}' | ext-kubernetes
```

Extension behavior:
1. Detect stdin closes after one message
2. Process the execute request
3. Write result to stdout
4. Exit

This allows easy testing with shell pipes.

---

## Example Session

```
→ {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.0.1"}}
← {"jsonrpc":"2.0","id":1,"result":{"name":"kubernetes","version":"1.0.0","protocolVersion":"0.0.1","operations":{...}}}
→ {"jsonrpc":"2.0","id":2,"method":"execute","params":{"operation":"create-namespace","args":{"name":"test-ns"},"context":{"workdir":"/tasks/create-pod","phase":"setup"}}}
← {"jsonrpc":"2.0","method":"log","params":{"level":"info","message":"Creating namespace test-ns"}}
← {"jsonrpc":"2.0","id":2,"result":{"success":true,"message":"Created namespace test-ns"}}
→ {"jsonrpc":"2.0","id":3,"method":"execute","params":{"operation":"condition","args":{"resource":"pod/nginx","conditionType":"Ready"},"context":{"workdir":"/tasks/create-pod","phase":"verify","agent":{"prompt":"Create nginx pod","output":"Done"}}}}
← {"jsonrpc":"2.0","method":"log","params":{"level":"info","message":"Checking pod/nginx..."}}
← {"jsonrpc":"2.0","method":"log","params":{"level":"info","message":"Waiting, status=ContainerCreating"}}
← {"jsonrpc":"2.0","id":3,"result":{"success":true,"message":"Pod nginx Ready=True"}}
→ {"jsonrpc":"2.0","id":4,"method":"shutdown","params":{}}
← {"jsonrpc":"2.0","id":4,"result":{}}
```

(Extension process exits)

---

## Duration Format

Duration values use Go duration syntax:

| Example | Meaning |
|---------|---------|
| `30s` | 30 seconds |
| `5m` | 5 minutes |
| `1h30m` | 1 hour 30 minutes |
| `100ms` | 100 milliseconds |

---

## Future Considerations

The following features are explicitly out of scope for v0.0.1 but may be added in future versions:

- **Cancellation**: Abort in-flight operations
- **Capability negotiation**: Extensions declare optional features
- **Binary data**: Passing file contents directly (currently use file paths)
- **Streaming results**: Progressive output for long operations
