package eval

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/genmcp/gevals/pkg/mcpproxy"
)

const (
	assertionTypeToolsUsed        = "toolsUsed"
	assertionTypeRequireAny       = "requireAny"
	assertionTypeToolsNotUsed     = "toolsNotUsed"
	assertionTypeMinToolCalls     = "minToolCalls"
	assertionTypeMaxToolCalls     = "maxToolCalls"
	assertionTypeResourcesRead    = "resourcesRead"
	assertionTypeResourcesNotRead = "resourcesNotRead"
	assertionTypePromptsUsed      = "promptsUsed"
	assertionTypePromptsNotUsed   = "promptsNotUsed"
	assertionTypeCallOrder        = "callOrder"
	assertionTypeNoDuplicateCalls = "noDuplicateCalls"
)

type SingleAssertionResult struct {
	Passed  bool     `json:"passed"`
	Reason  string   `json:"reason,omitempty"`
	Details []string `json:"details,omitempty"`
}

type CompositeAssertionResult struct {
	ToolsUsed        *SingleAssertionResult `json:"toolsUsed,omitempty"`
	RequireAny       *SingleAssertionResult `json:"requireAny,omitempty"`
	ToolsNotUsed     *SingleAssertionResult `json:"toolsNotUsed,omitempty"`
	MinToolCalls     *SingleAssertionResult `json:"minToolCalls,omitempty"`
	MaxToolCalls     *SingleAssertionResult `json:"maxToolCalls,omitempty"`
	ResourcesRead    *SingleAssertionResult `json:"resourcesRead,omitempty"`
	ResourcesNotRead *SingleAssertionResult `json:"resourcesNotRead,omitempty"`
	PromptsUsed      *SingleAssertionResult `json:"promptsUsed,omitempty"`
	PromptsNotUsed   *SingleAssertionResult `json:"promptsNotUsed,omitempty"`
	CallOrder        *SingleAssertionResult `json:"callOrder,omitempty"`
	NoDuplicateCalls *SingleAssertionResult `json:"noDuplicateCalls,omitempty"`
}

type CompositeAssertionEvaluator interface {
	Evaluate(history *mcpproxy.CallHistory) *CompositeAssertionResult
}

type SingleAssertionEvaluator interface {
	Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult
	Type() string
}

type assertionEvaluator struct {
	evaluators []SingleAssertionEvaluator
}

func NewCompositeAssertionEvaluator(assertions *TaskAssertions) CompositeAssertionEvaluator {
	evaluators := make([]SingleAssertionEvaluator, 0)
	if len(assertions.ToolsUsed) > 0 {
		evaluators = append(evaluators, NewToolsUsedEvaluator(assertions.ToolsUsed))
	}

	if len(assertions.RequireAny) > 0 {
		evaluators = append(evaluators, NewRequireAnyEvaluator(assertions.RequireAny))
	}

	if len(assertions.ToolsNotUsed) > 0 {
		evaluators = append(evaluators, NewToolsNotUsedEvaluator(assertions.ToolsNotUsed))
	}

	if assertions.MinToolCalls != nil {
		evaluators = append(evaluators, NewMinToolCallsEvaluator(*assertions.MinToolCalls))
	}

	if assertions.MaxToolCalls != nil {
		evaluators = append(evaluators, NewMaxToolCallsEvaluator(*assertions.MaxToolCalls))
	}

	if len(assertions.ResourcesRead) > 0 {
		evaluators = append(evaluators, NewResourcesReadEvaluator(assertions.ResourcesRead))
	}

	if len(assertions.ResourcesNotRead) > 0 {
		evaluators = append(evaluators, NewResourcesNotReadEvaluator(assertions.ResourcesNotRead))
	}

	if len(assertions.PromptsUsed) > 0 {
		evaluators = append(evaluators, NewPromptsUsedEvaluator(assertions.PromptsUsed))
	}

	if len(assertions.PromptsNotUsed) > 0 {
		evaluators = append(evaluators, NewPromptsNotUsedEvaluator(assertions.PromptsNotUsed))
	}

	if len(assertions.CallOrder) > 0 {
		evaluators = append(evaluators, NewCallOrderEvaluator(assertions.CallOrder))
	}

	if assertions.NoDuplicateCalls {
		evaluators = append(evaluators, NewNoDuplicateCallsEvaluator())
	}

	return &assertionEvaluator{
		evaluators: evaluators,
	}
}

func (a *assertionEvaluator) Evaluate(history *mcpproxy.CallHistory) *CompositeAssertionResult {
	res := &CompositeAssertionResult{}

	for _, eval := range a.evaluators {
		got := eval.Evaluate(history)
		switch eval.Type() {
		case assertionTypeToolsUsed:
			res.ToolsUsed = got
		case assertionTypeRequireAny:
			res.RequireAny = got
		case assertionTypeToolsNotUsed:
			res.ToolsNotUsed = got
		case assertionTypeMinToolCalls:
			res.MinToolCalls = got
		case assertionTypeMaxToolCalls:
			res.MaxToolCalls = got
		case assertionTypeResourcesRead:
			res.ResourcesRead = got
		case assertionTypeResourcesNotRead:
			res.ResourcesNotRead = got
		case assertionTypePromptsUsed:
			res.PromptsUsed = got
		case assertionTypePromptsNotUsed:
			res.PromptsNotUsed = got
		case assertionTypeCallOrder:
			res.CallOrder = got
		case assertionTypeNoDuplicateCalls:
			res.NoDuplicateCalls = got
		default:
		}
	}

	return res
}

type toolsUsedEvaluator struct {
	assertions []ToolAssertion
}

func NewToolsUsedEvaluator(assertions []ToolAssertion) SingleAssertionEvaluator {
	return &toolsUsedEvaluator{
		assertions: assertions,
	}
}

func (e *toolsUsedEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		found := false
		for _, call := range history.ToolCalls {
			if matchesToolAssertion(call, assertion) {
				found = true
				break
			}
		}

		if !found {
			return &SingleAssertionResult{
				Passed: false,
				Reason: fmt.Sprintf("Required tool not called: server=%s, tool=%s, pattern=%s",
					assertion.Server, assertion.Tool, assertion.ToolPattern,
				),
			}
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *toolsUsedEvaluator) Type() string {
	return assertionTypeToolsUsed
}

type requireAnyEvaluator struct {
	assertions []ToolAssertion
}

func NewRequireAnyEvaluator(assertions []ToolAssertion) SingleAssertionEvaluator {
	return &requireAnyEvaluator{
		assertions: assertions,
	}
}

func (e *requireAnyEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		for _, call := range history.ToolCalls {
			if matchesToolAssertion(call, assertion) {
				return &SingleAssertionResult{
					Passed:  true,
					Details: []string{fmt.Sprintf("Found server=%s, tool=%s", call.ServerName, call.ToolName)},
				}
			}
		}

	}
	return &SingleAssertionResult{
		Passed: false,
		Reason: "None of the required tools were called",
	}

}

func (e *requireAnyEvaluator) Type() string {
	return assertionTypeRequireAny
}

type toolsNotUsedEvaluator struct {
	assertions []ToolAssertion
}

func NewToolsNotUsedEvaluator(assertions []ToolAssertion) SingleAssertionEvaluator {
	return &toolsNotUsedEvaluator{
		assertions: assertions,
	}
}

func (e *toolsNotUsedEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		for _, call := range history.ToolCalls {
			if matchesToolAssertion(call, assertion) {
				return &SingleAssertionResult{
					Passed: false,
					Details: []string{fmt.Sprintf("Forbidden tool was called: server=%s, tool=%s",
						call.ServerName, call.ToolName),
					},
				}
			}
		}

	}

	return &SingleAssertionResult{Passed: true}
}

func (e *toolsNotUsedEvaluator) Type() string {
	return assertionTypeToolsNotUsed
}

type minToolCallsEvaluator struct {
	min int
}

func NewMinToolCallsEvaluator(min int) SingleAssertionEvaluator {
	return &minToolCallsEvaluator{
		min: min,
	}
}

func (e *minToolCallsEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	actual := len(history.ToolCalls)
	if actual < e.min {
		return &SingleAssertionResult{
			Passed: false,
			Reason: fmt.Sprintf("Too few tool calls: expected >= %d, got %d",
				e.min, actual),
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *minToolCallsEvaluator) Type() string {
	return assertionTypeMinToolCalls
}

type maxToolCallsEvaluator struct {
	max int
}

func NewMaxToolCallsEvaluator(max int) SingleAssertionEvaluator {
	return &maxToolCallsEvaluator{
		max: max,
	}
}

func (e *maxToolCallsEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	actual := len(history.ToolCalls)
	if actual > e.max {
		return &SingleAssertionResult{
			Passed: false,
			Reason: fmt.Sprintf("Too many tool calls: expected <= %d, got %d",
				e.max, actual),
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *maxToolCallsEvaluator) Type() string {
	return assertionTypeMaxToolCalls
}

type resourcesReadEvaluator struct {
	assertions []ResourceAssertion
}

func NewResourcesReadEvaluator(assertions []ResourceAssertion) SingleAssertionEvaluator {
	return &resourcesReadEvaluator{
		assertions: assertions,
	}
}

func (e *resourcesReadEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		found := false
		for _, call := range history.ResourceReads {
			if matchesResourceAssertion(call, assertion) {
				found = true
				break
			}
		}

		if !found {
			return &SingleAssertionResult{
				Passed: false,
				Reason: fmt.Sprintf("Required resource not read: server=%s, uri=%s, pattern=%s",
					assertion.Server, assertion.URI, assertion.URIPattern,
				),
			}
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *resourcesReadEvaluator) Type() string {
	return assertionTypeResourcesRead
}

type resourcesNotReadEvaluator struct {
	assertions []ResourceAssertion
}

func NewResourcesNotReadEvaluator(assertions []ResourceAssertion) SingleAssertionEvaluator {
	return &resourcesNotReadEvaluator{
		assertions: assertions,
	}
}

func (e *resourcesNotReadEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		for _, call := range history.ResourceReads {
			if matchesResourceAssertion(call, assertion) {
				return &SingleAssertionResult{
					Passed: false,
					Reason: fmt.Sprintf("Forbidden resource read: server=%s, uri=%s",
						assertion.Server, call.URI,
					),
				}
			}
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *resourcesNotReadEvaluator) Type() string {
	return assertionTypeResourcesNotRead
}

type promptsUsedEvaluator struct {
	assertions []PromptAssertion
}

func NewPromptsUsedEvaluator(assertions []PromptAssertion) SingleAssertionEvaluator {
	return &promptsUsedEvaluator{
		assertions: assertions,
	}
}

func (e *promptsUsedEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		found := false
		for _, call := range history.PromptGets {
			if matchesPromptAssertion(call, assertion) {
				found = true
				break
			}
		}

		if !found {
			return &SingleAssertionResult{
				Passed: false,
				Reason: fmt.Sprintf("Required prompt not used: server=%s, prompt=%s, pattern=%s",
					assertion.Server, assertion.Prompt, assertion.PromptPattern,
				),
			}
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *promptsUsedEvaluator) Type() string {
	return assertionTypePromptsUsed
}

type promptsNotUsedEvaluator struct {
	assertions []PromptAssertion
}

func NewPromptsNotUsedEvaluator(assertions []PromptAssertion) SingleAssertionEvaluator {
	return &promptsNotUsedEvaluator{
		assertions: assertions,
	}
}

func (e *promptsNotUsedEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	for _, assertion := range e.assertions {
		for _, call := range history.PromptGets {
			if matchesPromptAssertion(call, assertion) {
				return &SingleAssertionResult{
					Passed: false,
					Reason: fmt.Sprintf("Forbidden prompt used: server=%s, prompt=%s",
						assertion.Server, call.Name,
					),
				}
			}
		}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *promptsNotUsedEvaluator) Type() string {
	return assertionTypePromptsNotUsed
}

type callOrderEvaluator struct {
	callOrder []CallOrderAssertion
}

func NewCallOrderEvaluator(callOrder []CallOrderAssertion) SingleAssertionEvaluator {
	return &callOrderEvaluator{
		callOrder: callOrder,
	}
}

func (e *callOrderEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	type indexedCall struct {
		timestamp time.Time
		callType  string
		server    string
		name      string
	}

	allCalls := make([]indexedCall, 0, len(history.PromptGets)+len(history.ResourceReads)+len(history.ToolCalls))

	for _, tc := range history.ToolCalls {
		allCalls = append(allCalls, indexedCall{
			timestamp: tc.Timestamp,
			callType:  "tool",
			server:    tc.ServerName,
			name:      tc.ToolName,
		})
	}

	for _, rr := range history.ResourceReads {
		allCalls = append(allCalls, indexedCall{
			timestamp: rr.Timestamp,
			callType:  "resource",
			server:    rr.ServerName,
			name:      rr.URI,
		})
	}

	for _, pg := range history.PromptGets {
		allCalls = append(allCalls, indexedCall{
			timestamp: pg.Timestamp,
			callType:  "prompt",
			server:    pg.ServerName,
			name:      pg.Name,
		})
	}

	// sort chronologically
	sort.Slice(allCalls, func(i, j int) bool {
		return allCalls[i].timestamp.Before(allCalls[j].timestamp)
	})

	assertionIdx := 0
	for _, call := range allCalls {
		expectedCall := e.callOrder[assertionIdx]

		if call.callType == expectedCall.Type &&
			call.server == expectedCall.Server &&
			call.name == expectedCall.Name {
			assertionIdx++
			if assertionIdx >= len(e.callOrder) {
				// Found all calls in order
				return &SingleAssertionResult{Passed: true}
			}
		}
	}

	return &SingleAssertionResult{
		Passed: false,
		Reason: fmt.Sprintf("Expected call order not satisfied. Got to %d/%d",
			assertionIdx, len(e.callOrder)),
	}
}

func (e *callOrderEvaluator) Type() string {
	return assertionTypeCallOrder
}

type noDuplicateCallsEvaluator struct{}

func NewNoDuplicateCallsEvaluator() SingleAssertionEvaluator {
	return &noDuplicateCallsEvaluator{}
}

func (e *noDuplicateCallsEvaluator) Evaluate(history *mcpproxy.CallHistory) *SingleAssertionResult {
	seen := make(map[string]struct{})

	for _, call := range history.ToolCalls {
		key := fmt.Sprintf("%s:%s:%v", call.ServerName, call.ToolName, call.Request.Params.Arguments)

		if _, ok := seen[key]; ok {
			return &SingleAssertionResult{
				Passed: false,
				Reason: fmt.Sprintf("Duplicate call detected: %s.%s", call.ServerName, call.ToolName),
			}
		}

		seen[key] = struct{}{}
	}

	return &SingleAssertionResult{Passed: true}
}

func (e *noDuplicateCallsEvaluator) Type() string {
	return assertionTypeNoDuplicateCalls
}

func matchesToolAssertion(call *mcpproxy.ToolCall, assertion ToolAssertion) bool {
	if call == nil {
		return false
	}

	if call.ServerName != assertion.Server {
		return false
	}

	// if no tool or pattern specified, match any tool from this server
	if assertion.Tool == "" && assertion.ToolPattern == "" {
		return true
	}

	if assertion.Tool != "" && call.ToolName == assertion.Tool {
		return true
	}

	if assertion.ToolPattern != "" {
		matched, _ := regexp.MatchString(assertion.ToolPattern, call.ToolName)
		return matched
	}

	return false
}

func matchesResourceAssertion(call *mcpproxy.ResourceRead, assertion ResourceAssertion) bool {
	if call == nil {
		return false
	}

	if call.ServerName != assertion.Server {
		return false
	}

	// if no URI or pattern specified, match any resource from this server
	if assertion.URI == "" && assertion.URIPattern == "" {
		return true
	}

	if assertion.URI != "" && call.URI == assertion.URI {
		return true
	}

	if assertion.URIPattern != "" {
		matched, _ := regexp.MatchString(assertion.URIPattern, call.URI)
		return matched
	}

	return false
}

func matchesPromptAssertion(call *mcpproxy.PromptGet, assertion PromptAssertion) bool {
	if call == nil {
		return false
	}

	if call.ServerName != assertion.Server {
		return false
	}

	// if no prompt or pattern specified, match any prompt from this server
	if assertion.Prompt == "" && assertion.PromptPattern == "" {
		return true
	}

	if assertion.Prompt != "" && call.Name == assertion.Prompt {
		return true
	}

	if assertion.PromptPattern != "" {
		matched, _ := regexp.MatchString(assertion.PromptPattern, call.Name)
		return matched
	}

	return false
}
