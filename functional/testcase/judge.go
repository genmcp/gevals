package testcase

import (
	"time"

	"github.com/genmcp/gevals/functional/servers/openai"
)

// JudgeBuilder provides a fluent API for configuring mock judge behavior.
// The judge is an LLM that evaluates agent output and returns pass/fail decisions.
type JudgeBuilder struct {
	server *openai.MockOpenAIServer
}

// NewJudgeBuilder creates a new judge builder
func NewJudgeBuilder() *JudgeBuilder {
	return &JudgeBuilder{
		server: openai.NewMockOpenAIServer(),
	}
}

// WhenOutputContains configures the judge to respond when the agent output contains a substring.
// Returns a JudgeResponseBuilder to configure what the judge should return.
func (b *JudgeBuilder) WhenOutputContains(substring string) *JudgeResponseBuilder {
	return &JudgeResponseBuilder{
		judge:   b,
		matcher: openai.UserMessageContains(substring),
	}
}

// WhenOutputMatches configures the judge to respond when the agent output matches a regex.
// Returns a JudgeResponseBuilder to configure what the judge should return.
func (b *JudgeBuilder) WhenOutputMatches(pattern string) *JudgeResponseBuilder {
	return &JudgeResponseBuilder{
		judge:   b,
		matcher: openai.MessageMatches(pattern),
	}
}

// WhenSystemPromptContains configures the judge to respond when the system prompt contains a substring.
func (b *JudgeBuilder) WhenSystemPromptContains(substring string) *JudgeResponseBuilder {
	return &JudgeResponseBuilder{
		judge:   b,
		matcher: openai.SystemMessageContains(substring),
	}
}

// Always configures the judge to always respond with the configured response.
// This is typically used as a catch-all or when only one response is needed.
func (b *JudgeBuilder) Always() *JudgeResponseBuilder {
	return &JudgeResponseBuilder{
		judge:   b,
		matcher: openai.AnyRequest(),
	}
}

// Build returns the configured mock OpenAI server
func (b *JudgeBuilder) Build() *openai.MockOpenAIServer {
	return b.server
}

// JudgeResponseBuilder configures what the judge should return for a matched request
type JudgeResponseBuilder struct {
	judge   *JudgeBuilder
	matcher openai.RequestMatcher
	name    string
	times   int
}

// Named sets an optional name for this expectation (useful for debugging)
func (rb *JudgeResponseBuilder) Named(name string) *JudgeResponseBuilder {
	rb.name = name
	return rb
}

// Times limits how many times this expectation can match
func (rb *JudgeResponseBuilder) Times(n int) *JudgeResponseBuilder {
	rb.times = n
	return rb
}

// Pass configures the judge to return a passing result with the given reason.
// Returns the JudgeBuilder to continue configuration.
func (rb *JudgeResponseBuilder) Pass(reason string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgePass(reason),
		Times:    rb.times,
	})
	return rb.judge
}

// Fail configures the judge to return a failing result with the given category and reason.
// Valid categories: "semantic_mismatch", "missing_information", "contains_extra_info"
// Returns the JudgeBuilder to continue configuration.
func (rb *JudgeResponseBuilder) Fail(category, reason string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeFail(category, reason),
		Times:    rb.times,
	})
	return rb.judge
}

// FailSemanticMismatch configures the judge to fail with semantic_mismatch category
func (rb *JudgeResponseBuilder) FailSemanticMismatch(reason string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeFailSemanticMismatch(reason),
		Times:    rb.times,
	})
	return rb.judge
}

// FailMissingInformation configures the judge to fail with missing_information category
func (rb *JudgeResponseBuilder) FailMissingInformation(reason string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeFailMissingInformation(reason),
		Times:    rb.times,
	})
	return rb.judge
}

// FailContainsExtraInfo configures the judge to fail with contains_extra_info category
func (rb *JudgeResponseBuilder) FailContainsExtraInfo(reason string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeFailContainsExtraInfo(reason),
		Times:    rb.times,
	})
	return rb.judge
}

// Error configures the judge to return an API error
func (rb *JudgeResponseBuilder) Error(statusCode int, message string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeError(statusCode, message),
		Times:    rb.times,
	})
	return rb.judge
}

// Timeout configures the judge to delay response (simulating a timeout)
func (rb *JudgeResponseBuilder) Timeout(delay time.Duration) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeTimeout(delay),
		Times:    rb.times,
	})
	return rb.judge
}

// RateLimited configures the judge to return a 429 rate limit error
func (rb *JudgeResponseBuilder) RateLimited(message string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeRateLimited(message),
		Times:    rb.times,
	})
	return rb.judge
}

// ServiceUnavailable configures the judge to return a 503 error
func (rb *JudgeResponseBuilder) ServiceUnavailable(message string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeServiceUnavailable(message),
		Times:    rb.times,
	})
	return rb.judge
}

// InvalidResponse configures the judge to return malformed JSON (for testing error handling)
func (rb *JudgeResponseBuilder) InvalidResponse() *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeInvalidResponse(),
		Times:    rb.times,
	})
	return rb.judge
}

// NoToolCall configures the judge to return a text response without calling submit_judgement
func (rb *JudgeResponseBuilder) NoToolCall(message string) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeNoToolCall(message),
		Times:    rb.times,
	})
	return rb.judge
}

// WrongTool configures the judge to call a different tool instead of submit_judgement
func (rb *JudgeResponseBuilder) WrongTool(toolName string, args map[string]any) *JudgeBuilder {
	rb.judge.server.Expect(&openai.Expectation{
		Name:     rb.name,
		Matcher:  rb.matcher,
		Response: openai.JudgeWrongTool(toolName, args),
		Times:    rb.times,
	})
	return rb.judge
}

// OpenAIBuilder provides lower-level access to the mock OpenAI server.
// Use JudgeBuilder for the typical judge use case.
type OpenAIBuilder struct {
	server *openai.MockOpenAIServer
}

// NewOpenAIBuilder creates a new OpenAI builder for advanced use cases
func NewOpenAIBuilder() *OpenAIBuilder {
	return &OpenAIBuilder{
		server: openai.NewMockOpenAIServer(),
	}
}

// Expect adds an expectation directly to the server
func (b *OpenAIBuilder) Expect(e *openai.Expectation) *OpenAIBuilder {
	b.server.Expect(e)
	return b
}

// SetFallback sets the response when no expectation matches
func (b *OpenAIBuilder) SetFallback(r *openai.Response) *OpenAIBuilder {
	b.server.SetFallback(r)
	return b
}

// Build returns the configured mock OpenAI server
func (b *OpenAIBuilder) Build() *openai.MockOpenAIServer {
	return b.server
}

// Re-export types from openai package for convenience
type (
	MockOpenAIServer      = openai.MockOpenAIServer
	CapturedRequest       = openai.CapturedRequest
	ChatCompletionRequest = openai.ChatCompletionRequest
	Expectation           = openai.Expectation
	Response              = openai.Response
	RequestMatcher        = openai.RequestMatcher
	JudgeResult           = openai.JudgeResult
)

// Re-export failure category constants
const (
	FailureCategorySemanticMismatch   = openai.FailureCategorySemanticMismatch
	FailureCategoryMissingInformation = openai.FailureCategoryMissingInformation
	FailureCategoryContainsExtraInfo  = openai.FailureCategoryContainsExtraInfo
	FailureCategoryNA                 = openai.FailureCategoryNA
)

// Re-export matcher functions for advanced use cases
var (
	AnyRequest            = openai.AnyRequest
	MessageContains       = openai.MessageContains
	MessageContainsWithRole = openai.MessageContainsWithRole
	SystemMessageContains = openai.SystemMessageContains
	UserMessageContains   = openai.UserMessageContains
	MessageMatches        = openai.MessageMatches
	MessageMatchesWithRole = openai.MessageMatchesWithRole
	HasTool               = openai.HasTool
	ToolChoiceForces      = openai.ToolChoiceForces
	ToolChoiceIs          = openai.ToolChoiceIs
	ToolChoiceAllowsTools = openai.ToolChoiceAllowsTools
	ModelIs               = openai.ModelIs
	And                   = openai.And
	Or                    = openai.Or
	Not                   = openai.Not
	MatchFunc             = openai.MatchFunc
)

// Re-export judge response helpers for advanced use cases
var (
	JudgePass                    = openai.JudgePass
	JudgeFail                    = openai.JudgeFail
	JudgeFailSemanticMismatch    = openai.JudgeFailSemanticMismatch
	JudgeFailMissingInformation  = openai.JudgeFailMissingInformation
	JudgeFailContainsExtraInfo   = openai.JudgeFailContainsExtraInfo
	JudgeError                   = openai.JudgeError
	JudgeTimeout                 = openai.JudgeTimeout
	JudgeRateLimited             = openai.JudgeRateLimited
	JudgeServiceUnavailable      = openai.JudgeServiceUnavailable
	JudgeInvalidResponse         = openai.JudgeInvalidResponse
	JudgeWrongTool               = openai.JudgeWrongTool
	JudgeNoToolCall              = openai.JudgeNoToolCall
	JudgeMultipleToolCalls       = openai.JudgeMultipleToolCalls
	JudgeEmptyChoices            = openai.JudgeEmptyChoices
	BuildJudgeResponse           = openai.BuildJudgeResponse
)
