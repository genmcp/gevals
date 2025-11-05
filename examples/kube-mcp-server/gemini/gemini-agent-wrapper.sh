#!/usr/bin/env bash
# Wrapper script to configure gemini-cli with MCP server from config file

set -e

CONFIG_FILE="$1"
shift
ALLOWED_TOOLS="$1"
shift
PROMPT="$*"

# Extract URL from MCP config
URL=$(grep -o '"url"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*"url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

if [ -z "$URL" ]; then
	echo "Error: Could not extract URL from config file $CONFIG_FILE" >&2
	echo "Config contents:" >&2
	cat "$CONFIG_FILE" >&2
	exit 1
fi

# Generate unique server name for this eval run to avoid conflicts
SERVER_NAME="mcp-eval-$$"

echo "Configuring gemini with MCP server: $URL (as $SERVER_NAME)" >&2

# Add MCP server for this run
gemini mcp add "$SERVER_NAME" "$URL" --scope project --transport http --trust >/dev/null 2>&1

# Ensure cleanup on exit (success or failure)
trap "gemini mcp remove '$SERVER_NAME' >/dev/null 2>&1 || true" EXIT

# Run gemini with configured server and allowed tools
# --approval-mode yolo: Auto-approve all tool calls (required for automated evals)
# --output-format text: Ensure text output for parsing
if [ -n "$ALLOWED_TOOLS" ]; then
	gemini --allowed-mcp-server-names "$SERVER_NAME" \
		--allowed-tools "$ALLOWED_TOOLS" \
		--approval-mode yolo \
		--output-format text \
		--prompt "$PROMPT"
else
	gemini --allowed-mcp-server-names "$SERVER_NAME" \
		--approval-mode yolo \
		--output-format text \
		--prompt "$PROMPT"
fi
