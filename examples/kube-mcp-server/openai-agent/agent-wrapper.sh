#!/usr/bin/env bash
# Wrapper script to extract MCP server URL from config file and call agent

CONFIG_FILE="$1"
shift
PROMPT="$*"

# Extract the first server URL from the JSON config file
# Using grep and sed to parse JSON (simple approach)
URL=$(grep -o '"url"[[:space:]]*:[[:space:]]*"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*"url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

if [ -z "$URL" ]; then
    echo "Error: Could not extract URL from config file $CONFIG_FILE"
    echo "Config contents:"
    cat "$CONFIG_FILE"
    exit 1
fi

echo "Using MCP server URL: $URL"
./agent --mcp-url "$URL" --prompt "$PROMPT"
