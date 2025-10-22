# Agent Configuration Reference

Agent configuration files define how to run AI agents during evaluations, including how to pass MCP server configs and prompts.

## Agent YAML Structure

```yaml
kind: Agent
metadata:
  name: "agent-name"
  version: "1.0.0"  # optional
commands:
  useVirtualHome: false
  argTemplateMcpServer: "--mcp-config {{ .File }}"
  argTemplateAllowedTools: "mcp__{{ .ServerName }}__{{ .ToolName }}"
  allowedToolsJoinSeparator: " "  # optional
  runPrompt: |
    claude {{ .McpServerFileArgs }} --print "{{ .Prompt }}"
  getVersion: "claude --version"  # optional
```

## Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | Yes | Must be `"Agent"` |
| `metadata` | object | Yes | Agent metadata (see below) |
| `commands` | object | Yes | Command configuration (see below) |

## metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the agent |
| `version` | string | No | Agent version (overridden by `getVersion` if present) |

## commands Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `useVirtualHome` | boolean | No | Create isolated `$HOME` for agent (default: `false`) |
| `argTemplateMcpServer` | string | Yes | Template for MCP config file args (see below) |
| `argTemplateAllowedTools` | string | Yes | Template for allowed tool args (see below) |
| `allowedToolsJoinSeparator` | string | No | Separator for joining tools (default: `" "`) |
| `runPrompt` | string | Yes | Full command template to run agent (see below) |
| `getVersion` | string | No | Command to get agent version dynamically |

## Template Variables

### argTemplateMcpServer

Applied to each MCP server config file.

| Variable | Description |
|----------|-------------|
| `{{ .File }}` | Path to the MCP server config file |

**Example**:
```yaml
argTemplateMcpServer: "--mcp-config {{ .File }}"
```
→ Produces: `--mcp-config /tmp/mcp-config-123.json`

### argTemplateAllowedTools

Applied to each allowed tool.

| Variable | Description |
|----------|-------------|
| `{{ .ServerName }}` | Name of the MCP server |
| `{{ .ToolName }}` | Name of the tool |

**Example**:
```yaml
argTemplateAllowedTools: "mcp__{{ .ServerName }}__{{ .ToolName }}"
```
→ Produces: `mcp__kubernetes__pods_list mcp__kubernetes__pods_create`

### runPrompt

The complete command to execute the agent.

| Variable | Description |
|----------|-------------|
| `{{ .Prompt }}` | The task prompt text |
| `{{ .McpServerFileArgs }}` | All MCP server file arguments (space-separated) |
| `{{ .AllowedToolArgs }}` | All allowed tool arguments (joined by separator) |

**Example**:
```yaml
runPrompt: |
  claude {{ .McpServerFileArgs }} \
    --allowed-tools {{ .AllowedToolArgs }} \
    --print "{{ .Prompt }}"
```

## Complete Examples

### Claude Code Agent

This is the complete, production-ready agent configuration from `examples/kubernetes/agent.yaml`:

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

**Key features**:
- Uses comma separator for tools (required by Claude Code's `--allowedTools` flag)
- Includes `--strict-mcp-config` flag for strict MCP configuration validation
- Passes allowed tools via `--allowedTools` flag (must be quoted)
- Uses `--print` flag to output the prompt response

### Custom Agent with Allowed Tools

```yaml
kind: Agent
metadata:
  name: "custom-agent"
  version: "2.0.0"
commands:
  useVirtualHome: true
  argTemplateMcpServer: "--config {{ .File }}"
  argTemplateAllowedTools: "--allow {{ .ServerName }}::{{ .ToolName }}"
  allowedToolsJoinSeparator: " "
  runPrompt: |
    my-agent {{ .McpServerFileArgs }} {{ .AllowedToolArgs }} --task "{{ .Prompt }}"
```

### Agent with Comma-Separated Tools

```yaml
kind: Agent
metadata:
  name: "comma-agent"
commands:
  useVirtualHome: false
  argTemplateMcpServer: "-s {{ .File }}"
  argTemplateAllowedTools: "{{ .ServerName }}.{{ .ToolName }}"
  allowedToolsJoinSeparator: ","
  runPrompt: |
    agent run {{ .McpServerFileArgs }} --tools "{{ .AllowedToolArgs }}" "{{ .Prompt }}"
```

## How Templates Work

When running an agent:

1. **Format MCP server args**: Apply `argTemplateMcpServer` to each config file
2. **Format tool args**: Apply `argTemplateAllowedTools` to each allowed tool
3. **Join tools**: Combine tool args using `allowedToolsJoinSeparator`
4. **Build command**: Apply `runPrompt` with all variables
5. **Execute**: Run via `$SHELL -c "command"`

### Example Flow

Given:
- MCP config: `/tmp/mcp-123.json`
- Allowed tools: `kubernetes.pods_list`, `kubernetes.pods_create`
- Prompt: `"Create a pod named web"`

Templates:
```yaml
argTemplateMcpServer: "--mcp-config {{ .File }}"
argTemplateAllowedTools: "mcp__{{ .ServerName }}__{{ .ToolName }}"
runPrompt: claude {{ .McpServerFileArgs }} "{{ .Prompt }}"
```

Final command:
```bash
claude --mcp-config /tmp/mcp-123.json "Create a pod named web"
```
