# MCP Config Reference

MCP configuration defines which MCP servers to use during evaluation, including connection details and tool permissions.

## Configuration Methods

There are two ways to configure MCP servers:

1. **Config file** - Specify `mcpConfigFile` in the eval definition (recommended for complex setups)
2. **Environment variables** - Set `MCP_*` environment variables (recommended for CI/CD and single-server setups)

If both are provided, the config file takes priority.

## Environment Variable Configuration

For simple setups or CI/CD pipelines, configure MCP using environment variables:

### HTTP Server Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `MCP_URL` | Full HTTP URL for MCP server | - | `http://localhost:8080/mcp` |
| `MCP_HOST` | HTTP server host | `localhost` | `api.example.com` |
| `MCP_PORT` | HTTP server port | - | `8080` |
| `MCP_PATH` | HTTP path | `/mcp` | `/api/v1/mcp` |
| `MCP_HEADERS` | JSON object of HTTP headers | - | `{"Authorization":"Bearer token"}` |

Use either `MCP_URL` for a full URL, or `MCP_HOST`/`MCP_PORT`/`MCP_PATH` to build one.

### Stdio Server Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MCP_COMMAND` | Stdio server command | `npx` |
| `MCP_ARGS` | Comma-separated or JSON array of args | `-y,@modelcontextprotocol/server-filesystem` |
| `MCP_ENV` | JSON object of environment variables | `{"KUBECONFIG":"/path/to/config"}` |

### Common Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `MCP_SERVER_NAME` | Server name in config | `default` | `my-server` |
| `MCP_ENABLE_ALL_TOOLS` | Enable all tools | `true` | `false` |

### Examples

**HTTP server:**
```bash
export MCP_URL=http://localhost:8080/mcp
export MCP_HEADERS='{"Authorization":"Bearer my-token"}'
```

**HTTP server from components:**
```bash
export MCP_HOST=api.example.com
export MCP_PORT=8080
export MCP_PATH=/api/mcp
```

**Stdio server:**
```bash
export MCP_COMMAND=npx
export MCP_ARGS='-y,@modelcontextprotocol/server-filesystem,/tmp'
```

**Stdio server with JSON args:**
```bash
export MCP_COMMAND=npx
export MCP_ARGS='["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]'
```

## Config File Configuration

For more complex setups with multiple servers, use a config file.

### MCP Config Structure

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

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mcpServers` | object | Yes | Map of server names to server configurations |

### Server Configuration Fields

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
