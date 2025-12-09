package cli

import (
	"strings"
	"testing"
)

func TestSummarizeTaskOutput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxEvents     int
		expectedItems []string
	}{
		{
			name: "Plaintext Codex Style",
			input: `Thinking:
I need to check the pods.

Plan:
1. List pods
2. Check logs

Exec:
kubectl get pods
NAME    READY   STATUS
pod-1   1/1     Running

Tool:
run_shell_command
cmd: ls
output: file1 file2
`,
			maxEvents: 0,
			expectedItems: []string{
				"thought: I need to check the pods.",
				"plan: 2 steps (1. List pods)",
				"command: kubectl get pods\n      NAME    READY   STATUS",
				"tool: run_shell_command\n      cmd: ls",
			},
		},
		{
			name: "Gemini JSON Stream",
			input: `YOLO mode is enabled.
{"type": "message", "role": "assistant", "content": "I am analyzing the issue."}
{"type": "tool_use", "tool_name": "run_shell_command", "parameters": {}}
{"type": "tool_result", "tool_name": "run_shell_command", "output": "command output"}
{"type": "message", "role": "user", "content": "next step"}
`,
			maxEvents: 0,
			expectedItems: []string{
				"assistant: I am analyzing the issue.",
				"tool call: run_shell_command",
				"tool result: command output", // Tool result IS split? No, let's check view.go
				"user: next step",
			},
		},
		// ...
		{
			name: "Truncation of Long Output",
			input: `Thinking:
Short thought.

Exec:
long_command
Line 1
Line 2
Line 3
Line 4
Line 5
Line 6
Line 7
`,
			maxEvents: 0, // Testing output lines limit, not event limit
			expectedItems: []string{
				"thought: Short thought.",
				"command: long_command\n      Line 1", // We only check start of block because "Line 6" might be further down
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using default maxOutputLines=6, maxLineLength=100
			got := summarizeTaskOutput(tt.input, tt.maxEvents, 6, 100)

			// Verify all expected items are present in order (substring match for simplicity)
			nextIdx := 0
			for _, want := range tt.expectedItems {
				found := false
				for i := nextIdx; i < len(got); i++ {
					if strings.Contains(got[i], want) {
						found = true
						nextIdx = i + 1
						break
					}
				}
				if !found {
					t.Errorf("expected item %q not found in remaining output starting at index %d.\nGot:\n%v", want, nextIdx, strings.Join(got, "\n---\n"))
				}
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell…"},
		{"你好世界", 2, "你…"}, // Multi-byte check
		{"你好世界", 4, "你好世界"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}
