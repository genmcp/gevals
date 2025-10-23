# MCP Config Reference

MCP configuration files define which MCP servers to use during evaluation, including connection details and tool permissions.

## MCP Config Structure

```yaml
mcpServers:
  server-name:
    type: "http|stdio"
    url: "..."              # for HTTP servers
    headers: {}             # for HTTP servers
    command: "..."          # for stdio servers
    args: []                # for stdio servers
    env: {}                 # for stdio servers
    disabled: false
    alwaysAllow: []
    enableAllTools: true
```

## Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mcpServers` | object | Yes | Map of server names to server configurations |

## Server Configuration Fields

Each server under `mcpServers` has these available fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | `"http"` or `"stdio"` (auto-inferred if not set) |
| `url` | string | For HTTP | HTTP endpoint URL (supports `${VAR}` expansion) |
| `headers` | object | No | HTTP headers for requests (values support `${VAR}`) |
| `command` | string | For stdio | Executable to run (e.g., `"node"`, `"python"`, `"npx"`) |
| `args` | array | No | Command-line arguments for stdio servers |
| `env` | object | No | Environment variables for stdio server process |
| `disabled` | boolean | No | Skip this server if `true` (default: `false`) |
| `alwaysAllow` | array | No | List of tool names to enable |
| `enableAllTools` | boolean | No | Enable all tools from this server (default: `false`) |

### Type Inference

If `type` is not specified:
- Has `url` → automatically treated as HTTP server
- Has `command` → automatically treated as stdio server

## HTTP Server Example

```yaml
mcpServers:
  kubernetes:
    type: http
    url: http://localhost:8080/mcp
    headers:
      Authorization: "Bearer ${K8S_TOKEN}"
      X-Custom-Header: "value"
    enableAllTools: true
```

## Stdio Server Example

```yaml
mcpServers:
  filesystem:
    type: stdio
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "/tmp/allowed-dir"
    env:
      DEBUG: "true"
      LOG_LEVEL: "info"
    enableAllTools: true
```

## Environment Variable Expansion

Use in `url`, `headers` values, and `env` values:

| Syntax | Behavior |
|--------|----------|
| `${VAR}` | Required variable (error if not set) |
| `${VAR:-default}` | Optional with default value |

```yaml
mcpServers:
  api-server:
    type: http
    url: ${API_URL:-http://localhost:8080}
    headers:
      Authorization: "Bearer ${API_TOKEN}"
```

## Tool Permissions

### Enable All Tools

```yaml
mcpServers:
  my-server:
    url: http://localhost:8080
    enableAllTools: true
```

### Enable Specific Tools

```yaml
mcpServers:
  kubernetes:
    url: http://localhost:8080/mcp
    alwaysAllow:
      - "pods_list"
      - "pods_create"
      - "pods_delete"
```

## Complete Example

```yaml
mcpServers:
  # HTTP server with auth
  kubernetes:
    type: http
    url: http://localhost:8080/mcp
    headers:
      Authorization: "Bearer ${K8S_TOKEN}"
    enableAllTools: true

  # Stdio server with NPX
  filesystem:
    type: stdio
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "/workspace"
    env:
      LOG_LEVEL: "debug"
    alwaysAllow:
      - "read_file"
      - "write_file"

  # Disabled server
  legacy-api:
    type: http
    url: http://old-server:9090
    disabled: true
```
