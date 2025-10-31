package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/genmcp/gevals/pkg/eval"
	"github.com/genmcp/gevals/pkg/mcpproxy"
	"github.com/genmcp/gevals/pkg/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

const (
	defaultMaxEvents      = 40
	defaultMaxOutputLines = 6
	defaultMaxLineLength  = 100
)

// NewViewCmd creates the view command for rendering eval results.
func NewViewCmd() *cobra.Command {
	var (
		taskFilter     string
		showTimeline   = true
		maxEvents      = defaultMaxEvents
		maxOutputLines = defaultMaxOutputLines
		maxLineLength  = defaultMaxLineLength
	)

	cmd := &cobra.Command{
		Use:   "view <results-file>",
		Short: "Pretty-print evaluation results from a JSON file",
		Long: `Render the JSON output produced by "geval run" in a human-friendly format.

Examples:
  geval view gevals-netedge-selector-mismatch-out.json
  geval view --task netedge-selector-mismatch --max-events 15 results.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := loadEvalResults(args[0])
			if err != nil {
				return err
			}

			filtered := filterResults(results, taskFilter)
			if len(filtered) == 0 {
				if taskFilter == "" {
					return errors.New("no tasks found in results")
				}
				return fmt.Errorf("no tasks matched filter %q", taskFilter)
			}

			for idx, result := range filtered {
				if idx > 0 {
					fmt.Println()
				}
				printEvalResult(result, viewOptions{
					showTimeline:   showTimeline,
					maxEvents:      maxEvents,
					maxOutputLines: maxOutputLines,
					maxLineLength:  maxLineLength,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&taskFilter, "task", "", "Only show results for tasks whose name contains this value")
	cmd.Flags().BoolVar(&showTimeline, "timeline", showTimeline, "Include a condensed agent timeline derived from taskOutput")
	cmd.Flags().IntVar(&maxEvents, "max-events", maxEvents, "Maximum number of timeline events to display (0 = unlimited)")
	cmd.Flags().IntVar(&maxOutputLines, "max-output-lines", maxOutputLines, "Maximum lines to display for command output in the timeline")
	cmd.Flags().IntVar(&maxLineLength, "max-line-length", maxLineLength, "Maximum characters per line when formatting timeline output")

	return cmd
}

func loadEvalResults(path string) ([]*eval.EvalResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	var results []*eval.EvalResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to parse results JSON: %w", err)
	}

	return results, nil
}

type viewOptions struct {
	showTimeline   bool
	maxEvents      int
	maxOutputLines int
	maxLineLength  int
}

func filterResults(results []*eval.EvalResult, filter string) []*eval.EvalResult {
	if filter == "" {
		return results
	}

	filter = strings.ToLower(filter)
	filtered := make([]*eval.EvalResult, 0, len(results))
	for _, r := range results {
		if strings.Contains(strings.ToLower(r.TaskName), filter) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func printEvalResult(result *eval.EvalResult, opts viewOptions) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	bold.Printf("Task: %s\n", result.TaskName)
	fmt.Printf("  Path: %s\n", result.TaskPath)
	if result.Difficulty != "" {
		fmt.Printf("  Difficulty: %s\n", result.Difficulty)
	}

	status := "PASSED"
	statusColor := green

	switch {
	case result.AgentExecutionError:
		status = "FAILED (agent error)"
		statusColor = red
	case !result.TaskPassed:
		status = "FAILED"
		statusColor = red
	case result.TaskPassed && !result.AllAssertionsPassed:
		status = "PASSED (assertions failed)"
		statusColor = yellow
	}

	statusColor.Printf("  Status: %s\n", status)
	if trimmed := strings.TrimSpace(result.TaskError); trimmed != "" {
		printMultilineField("Error", trimmed)
	}

	if prompt := loadTaskPrompt(result.TaskPath); prompt != "" {
		printMultilineField("Prompt", prompt)
	}

	printAssertions(result.AssertionResults, yellow)
	printCallHistory(result.CallHistory)

	if opts.showTimeline {
		timeline := summarizeTaskOutput(result.TaskOutput, opts.maxEvents, opts.maxOutputLines, opts.maxLineLength)
		if len(timeline) > 0 {
			fmt.Println("  Timeline:")
			for _, line := range timeline {
				printTimelineLine(line)
			}
		}
	}
}

func printAssertions(results *eval.CompositeAssertionResult, warn *color.Color) {
	if results == nil {
		return
	}

	failed := results.FailedAssertions()
	total := results.TotalAssertions()
	if total == 0 {
		return
	}

	if failed == 0 {
		fmt.Printf("  Assertions: %d/%d passed\n", total, total)
		return
	}

	warn.Printf("  Assertions: %d/%d passed\n", total-failed, total)

	type entry struct {
		name   string
		result *eval.SingleAssertionResult
	}

	all := []entry{
		{"ToolsUsed", results.ToolsUsed},
		{"RequireAny", results.RequireAny},
		{"ToolsNotUsed", results.ToolsNotUsed},
		{"MinToolCalls", results.MinToolCalls},
		{"MaxToolCalls", results.MaxToolCalls},
		{"ResourcesRead", results.ResourcesRead},
		{"ResourcesNotRead", results.ResourcesNotRead},
		{"PromptsUsed", results.PromptsUsed},
		{"PromptsNotUsed", results.PromptsNotUsed},
		{"CallOrder", results.CallOrder},
		{"NoDuplicateCalls", results.NoDuplicateCalls},
	}

	for _, entry := range all {
		if entry.result == nil || entry.result.Passed {
			continue
		}
		fmt.Printf("    • %s: %s\n", entry.name, entry.result.Reason)
		for _, detail := range entry.result.Details {
			fmt.Printf("      %s\n", detail)
		}
	}
}

func printCallHistory(history *mcpproxy.CallHistory) {
	if history == nil {
		return
	}

	toolCalls := len(history.ToolCalls)
	resourceReads := len(history.ResourceReads)
	promptGets := len(history.PromptGets)

	if toolCalls == 0 && resourceReads == 0 && promptGets == 0 {
		return
	}

	fmt.Printf("  Call history:")
	if toolCalls > 0 {
		fmt.Printf(" tools=%d", toolCalls)
		if summaries := summarizeToolCalls(history.ToolCalls); summaries != "" {
			fmt.Printf(" (%s)", summaries)
		}
	}
	if resourceReads > 0 {
		fmt.Printf(" resources=%d", resourceReads)
	}
	if promptGets > 0 {
		fmt.Printf(" prompts=%d", promptGets)
	}
	fmt.Println()

	if toolCalls > 0 {
		printToolCallDetails(history.ToolCalls)
	}
}

func printToolCallDetails(calls []*mcpproxy.ToolCall) {
	fmt.Println("    Tool output:")
	for _, call := range calls {
		status := "ok"
		if !call.Success {
			status = "fail"
		}
		header := fmt.Sprintf("      • %s::%s (%s)", call.ServerName, call.ToolName, status)
		fmt.Println(header)

		snippet := strings.TrimSpace(extractToolText(call))
		if snippet == "" {
			continue
		}

		block := limitMultiline(snippet, 12, 110)
		for _, line := range strings.Split(block, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fmt.Printf("        %s\n", line)
		}
	}
}

func extractToolText(call *mcpproxy.ToolCall) string {
	if call == nil || call.Result == nil {
		return ""
	}

	var builder strings.Builder
	for _, content := range call.Result.Content {
		switch v := content.(type) {
		case *mcp.TextContent:
			builder.WriteString(v.Text)
			if !strings.HasSuffix(v.Text, "\n") {
				builder.WriteString("\n")
			}
		case *mcp.ResourceLink:
			data, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				continue
			}
			builder.Write(data)
			builder.WriteString("\n")
		case *mcp.EmbeddedResource:
			data, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				continue
			}
			builder.Write(data)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func summarizeToolCalls(calls []*mcpproxy.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}

	type key struct {
		server  string
		success bool
	}

	counts := make(map[key]int)
	for _, call := range calls {
		callKey := key{server: call.ServerName, success: call.Success}
		counts[callKey]++
	}

	type serverSummary struct {
		server  string
		success bool
		count   int
	}

	summaries := make([]serverSummary, 0, len(counts))
	for k, v := range counts {
		summaries = append(summaries, serverSummary{
			server:  k.server,
			success: k.success,
			count:   v,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].server == summaries[j].server {
			return summaries[i].success && !summaries[j].success
		}
		return summaries[i].server < summaries[j].server
	})

	parts := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		state := "ok"
		if !summary.success {
			state = "fail"
		}
		parts = append(parts, fmt.Sprintf("%s:%d %s", summary.server, summary.count, state))
	}

	return strings.Join(parts, ", ")
}

type agentEvent struct {
	Type    string          `json:"type"`
	Item    json.RawMessage `json:"item,omitempty"`
	Message string          `json:"message,omitempty"`
}

type agentItem struct {
	ID               string      `json:"id"`
	Type             string      `json:"type"`
	Text             string      `json:"text,omitempty"`
	Command          string      `json:"command,omitempty"`
	AggregatedOutput string      `json:"aggregated_output,omitempty"`
	Status           string      `json:"status,omitempty"`
	Server           string      `json:"server,omitempty"`
	Tool             string      `json:"tool,omitempty"`
	ExitCode         *int        `json:"exit_code,omitempty"`
	Items            []todoEntry `json:"items,omitempty"`
}

type todoEntry struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

func summarizeTaskOutput(raw string, maxEvents, maxOutputLines, maxLineLength int) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	lines := strings.Split(raw, "\n")
	summaries := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var evt agentEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			summaries = append(summaries, fmt.Sprintf("unparsed event: %s", truncateString(line, maxLineLength)))
			continue
		}

		if summary := formatEvent(evt, maxOutputLines, maxLineLength); summary != "" {
			summaries = append(summaries, summary)
		}
	}

	if maxEvents > 0 && len(summaries) > maxEvents {
		extra := len(summaries) - maxEvents
		summaries = append(summaries[:maxEvents], fmt.Sprintf("… %d additional events omitted", extra))
	}

	return summaries
}

func formatEvent(evt agentEvent, maxOutputLines, maxLineLength int) string {
	switch evt.Type {
	case "thread.started", "turn.started", "turn.completed":
		return ""
	}

	if evt.Type != "item.completed" && evt.Type != "item.failed" && evt.Type != "item.updated" && evt.Type != "item.started" {
		if evt.Message != "" {
			msg := evt.Message
			if wrapped := wrapText(msg, maxLineLength); wrapped != "" {
				return wrapped
			}
			return msg
		}
		return ""
	}

	if len(evt.Item) == 0 {
		return ""
	}

	var item agentItem
	if err := json.Unmarshal(evt.Item, &item); err != nil {
		return ""
	}

	if evt.Type == "item.started" {
		switch item.Type {
		case "command_execution", "mcp_tool_call":
			return ""
		}
	}

	switch item.Type {
	case "reasoning":
		text := normalizeWhitespace(item.Text)
		text = wrapText(text, maxLineLength)
		return fmt.Sprintf("thought: %s", text)
	case "command_execution":
		summary := fmt.Sprintf("command: %s", item.Command)
		if item.Status != "" {
			summary = fmt.Sprintf("%s (%s)", summary, item.Status)
		}
		if item.ExitCode != nil {
			summary = fmt.Sprintf("%s exit=%d", summary, *item.ExitCode)
		}
		summary = wrapText(summary, maxLineLength)
		if item.AggregatedOutput != "" {
			block := limitMultiline(item.AggregatedOutput, maxOutputLines, maxLineLength)
			if block != "" {
				summary = fmt.Sprintf("%s\n%s", summary, indentBlock(block, "      "))
			}
		}
		return summary
	case "mcp_tool_call":
		if item.Server == "" && item.Tool == "" {
			return "tool call"
		}
		detail := fmt.Sprintf("tool: %s::%s", item.Server, item.Tool)
		if item.Status != "" {
			detail = fmt.Sprintf("%s (%s)", detail, item.Status)
		}
		return detail
	case "todo_list":
		count := len(item.Items)
		if count == 0 {
			return "plan: todo list started"
		}
		headline := normalizeWhitespace(item.Items[0].Text)
		headline = wrapText(headline, maxLineLength)
		if count == 1 {
			return fmt.Sprintf("plan: %s", headline)
		}
		return fmt.Sprintf("plan: %d tasks (%s)", count, headline)
	default:
		return fmt.Sprintf("%s event", item.Type)
	}
}

func limitMultiline(raw string, maxLines, maxLineLength int) string {
	raw = strings.TrimRight(raw, "\n")
	if raw == "" {
		return ""
	}

	lines := strings.Split(raw, "\n")
	limited := make([]string, 0, len(lines))
	for idx, line := range lines {
		if maxLines > 0 && idx >= maxLines {
			limited = append(limited, fmt.Sprintf("… (+%d lines)", len(lines)-idx))
			break
		}
		if maxLineLength > 0 {
			for _, wrapped := range strings.Split(wrapText(line, maxLineLength), "\n") {
				limited = append(limited, wrapped)
			}
		} else {
			limited = append(limited, line)
		}
	}
	return strings.Join(limited, "\n")
}

func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return fmt.Sprintf("%s…", strings.TrimSpace(s[:max-1]))
}

func indentBlock(block, indent string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func normalizeWhitespace(in string) string {
	in = strings.ReplaceAll(in, "\n", " ")
	in = strings.ReplaceAll(in, "\t", " ")
	in = strings.ReplaceAll(in, "**", "")
	fields := strings.Fields(in)
	return strings.Join(fields, " ")
}

func wrapText(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	lines := make([]string, 0)
	current := words[0]

	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
		} else {
			current += " " + word
		}
	}
	lines = append(lines, current)

	return strings.Join(lines, "\n")
}

func loadTaskPrompt(taskPath string) string {
	if taskPath == "" {
		return ""
	}

	spec, err := task.FromFile(taskPath)
	if err != nil || spec == nil || spec.Steps.Prompt == nil || spec.Steps.Prompt.IsEmpty() {
		return ""
	}

	text, err := spec.Steps.Prompt.GetValue()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(text)
}

func printMultilineField(label, value string) {
	value = strings.TrimRight(value, "\n")
	value = strings.ReplaceAll(value, "\n': exit status", " exit status")
	if !strings.Contains(value, "\n") {
		fmt.Printf("  %s: %s\n", label, value)
		return
	}

	fmt.Printf("  %s:\n", label)
	lines := mergeContinuationLines(strings.Split(value, "\n"))
	for _, line := range lines {
		fmt.Printf("    %s\n", line)
	}
}

func mergeContinuationLines(lines []string) []string {
	merged := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if len(merged) > 0 {
			switch trimmed[0] {
			case '\'', '"', ')', '.', ':':
				merged[len(merged)-1] = merged[len(merged)-1] + " " + trimmed
				continue
			}
		}

		merged = append(merged, trimmed)
	}

	for i, line := range merged {
		line = strings.ReplaceAll(line, "' : exit", "' exit")
		line = strings.ReplaceAll(line, "\" : exit", "\" exit")
		merged[i] = line
	}

	return merged
}

func printTimelineLine(entry string) {
	parts := strings.Split(entry, "\n")
	if len(parts) == 0 {
		return
	}

	fmt.Printf("    - %s\n", parts[0])
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "" {
			continue
		}
		clean := part
		if strings.HasPrefix(clean, "      ") {
			clean = strings.TrimPrefix(clean, "      ")
		}
		fmt.Printf("      %s\n", clean)
	}
}
