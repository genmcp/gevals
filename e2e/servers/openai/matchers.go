package openai

import (
	"regexp"
	"strings"
)

// RequestMatcher determines if a request matches an expectation
type RequestMatcher interface {
	Matches(req *ChatCompletionRequest) bool
}

// AnyRequestMatcher matches all requests
type AnyRequestMatcher struct{}

func (m AnyRequestMatcher) Matches(req *ChatCompletionRequest) bool {
	return true
}

// AnyRequest returns a matcher that matches all requests
func AnyRequest() RequestMatcher {
	return AnyRequestMatcher{}
}

// MessageContentMatcher matches if any message content contains a substring
type MessageContentMatcher struct {
	Substring string
	Role      string // Optional: only check messages with this role (empty = all roles)
}

func (m MessageContentMatcher) Matches(req *ChatCompletionRequest) bool {
	for _, msg := range req.Messages {
		if m.Role != "" && msg.Role != m.Role {
			continue
		}
		if strings.Contains(msg.Content, m.Substring) {
			return true
		}
	}
	return false
}

// MessageContains returns a matcher for messages containing a substring
func MessageContains(substring string) RequestMatcher {
	return MessageContentMatcher{Substring: substring}
}

// MessageContainsWithRole returns a matcher for messages with a specific role containing a substring
func MessageContainsWithRole(role, substring string) RequestMatcher {
	return MessageContentMatcher{Substring: substring, Role: role}
}

// SystemMessageContains returns a matcher for system messages containing a substring
func SystemMessageContains(substring string) RequestMatcher {
	return MessageContentMatcher{Substring: substring, Role: "system"}
}

// UserMessageContains returns a matcher for user messages containing a substring
func UserMessageContains(substring string) RequestMatcher {
	return MessageContentMatcher{Substring: substring, Role: "user"}
}

// MessageContentRegexMatcher matches if any message content matches a regex
type MessageContentRegexMatcher struct {
	Pattern *regexp.Regexp
	Role    string // Optional: only check messages with this role
}

func (m MessageContentRegexMatcher) Matches(req *ChatCompletionRequest) bool {
	for _, msg := range req.Messages {
		if m.Role != "" && msg.Role != m.Role {
			continue
		}
		if m.Pattern.MatchString(msg.Content) {
			return true
		}
	}
	return false
}

// MessageMatches returns a matcher for messages matching a regex pattern
func MessageMatches(pattern string) RequestMatcher {
	return MessageContentRegexMatcher{Pattern: regexp.MustCompile(pattern)}
}

// MessageMatchesWithRole returns a matcher for messages with a specific role matching a regex
func MessageMatchesWithRole(role, pattern string) RequestMatcher {
	return MessageContentRegexMatcher{Pattern: regexp.MustCompile(pattern), Role: role}
}

// HasToolMatcher matches if the request includes a specific tool
type HasToolMatcher struct {
	ToolName string
}

func (m HasToolMatcher) Matches(req *ChatCompletionRequest) bool {
	for _, tool := range req.Tools {
		if tool.Function.Name == m.ToolName {
			return true
		}
	}
	return false
}

// HasTool returns a matcher that checks if a specific tool is in the request
func HasTool(toolName string) RequestMatcher {
	return HasToolMatcher{ToolName: toolName}
}

// ToolChoiceFunctionMatcher matches when tool_choice forces a specific function
type ToolChoiceFunctionMatcher struct {
	FunctionName string
}

func (m ToolChoiceFunctionMatcher) Matches(req *ChatCompletionRequest) bool {
	if req.ToolChoice == nil {
		return false
	}
	return req.ToolChoice.ForcedFunctionName() == m.FunctionName
}

// ToolChoiceForces returns a matcher that checks if tool_choice forces a specific function
func ToolChoiceForces(functionName string) RequestMatcher {
	return ToolChoiceFunctionMatcher{FunctionName: functionName}
}

// ToolChoiceStringMatcher matches string tool_choice values
type ToolChoiceStringMatcher struct {
	Value string // "none", "auto", or "required"
}

func (m ToolChoiceStringMatcher) Matches(req *ChatCompletionRequest) bool {
	if req.ToolChoice == nil {
		return false
	}
	return req.ToolChoice.IsString && req.ToolChoice.StringValue == m.Value
}

// ToolChoiceIs returns a matcher for a specific string tool_choice value
func ToolChoiceIs(value string) RequestMatcher {
	return ToolChoiceStringMatcher{Value: value}
}

// ToolChoiceAllowedMatcher matches when tool_choice uses allowed_tools with specific tools
type ToolChoiceAllowedMatcher struct {
	Mode          string   // "auto" or "required", empty means any
	RequiredTools []string // All these tools must be in the allowed list
}

func (m ToolChoiceAllowedMatcher) Matches(req *ChatCompletionRequest) bool {
	if req.ToolChoice == nil || !req.ToolChoice.IsAllowedTools() {
		return false
	}

	if m.Mode != "" && req.ToolChoice.AllowedTools.Mode != m.Mode {
		return false
	}

	allowedNames := req.ToolChoice.AllowedToolNames()
	for _, required := range m.RequiredTools {
		found := false
		for _, allowed := range allowedNames {
			if allowed == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// ToolChoiceAllowsTools returns a matcher for allowed_tools with specific required tools
func ToolChoiceAllowsTools(requiredTools ...string) RequestMatcher {
	return ToolChoiceAllowedMatcher{RequiredTools: requiredTools}
}

// ToolChoiceAllowsToolsWithMode returns a matcher for allowed_tools with mode and specific tools
func ToolChoiceAllowsToolsWithMode(mode string, requiredTools ...string) RequestMatcher {
	return ToolChoiceAllowedMatcher{Mode: mode, RequiredTools: requiredTools}
}

// ModelMatcher matches requests to a specific model
type ModelMatcher struct {
	Model string
}

func (m ModelMatcher) Matches(req *ChatCompletionRequest) bool {
	return req.Model == m.Model
}

// ModelIs returns a matcher for a specific model
func ModelIs(model string) RequestMatcher {
	return ModelMatcher{Model: model}
}

// SeedMatcher matches requests with a specific seed value
type SeedMatcher struct {
	Seed int64
}

func (m SeedMatcher) Matches(req *ChatCompletionRequest) bool {
	return req.Seed != nil && *req.Seed == m.Seed
}

// SeedIs returns a matcher for a specific seed value
func SeedIs(seed int64) RequestMatcher {
	return SeedMatcher{Seed: seed}
}

// AndMatcher combines multiple matchers with AND logic
type AndMatcher struct {
	Matchers []RequestMatcher
}

func (m AndMatcher) Matches(req *ChatCompletionRequest) bool {
	for _, matcher := range m.Matchers {
		if !matcher.Matches(req) {
			return false
		}
	}
	return true
}

// And combines multiple matchers with AND logic
func And(matchers ...RequestMatcher) RequestMatcher {
	return AndMatcher{Matchers: matchers}
}

// OrMatcher combines multiple matchers with OR logic
type OrMatcher struct {
	Matchers []RequestMatcher
}

func (m OrMatcher) Matches(req *ChatCompletionRequest) bool {
	for _, matcher := range m.Matchers {
		if matcher.Matches(req) {
			return true
		}
	}
	return false
}

// Or combines multiple matchers with OR logic
func Or(matchers ...RequestMatcher) RequestMatcher {
	return OrMatcher{Matchers: matchers}
}

// NotMatcher negates a matcher
type NotMatcher struct {
	Matcher RequestMatcher
}

func (m NotMatcher) Matches(req *ChatCompletionRequest) bool {
	return !m.Matcher.Matches(req)
}

// Not negates a matcher
func Not(matcher RequestMatcher) RequestMatcher {
	return NotMatcher{Matcher: matcher}
}

// FuncMatcher allows custom matching logic via a function
type FuncMatcher struct {
	Fn func(*ChatCompletionRequest) bool
}

func (m FuncMatcher) Matches(req *ChatCompletionRequest) bool {
	return m.Fn(req)
}

// MatchFunc creates a matcher from a function
func MatchFunc(fn func(*ChatCompletionRequest) bool) RequestMatcher {
	return FuncMatcher{Fn: fn}
}
