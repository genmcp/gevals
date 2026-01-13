package openai

import (
	"encoding/json"
	"fmt"
	"time"
)

// JudgeResult represents the result returned by the judge via submit_judgement tool
// This matches the LLMJudgeResult struct in pkg/llmjudge/llmjudge.go
type JudgeResult struct {
	Passed          bool   `json:"passed"`
	Reason          string `json:"reason"`
	FailureCategory string `json:"failureCategory"`
}

// Failure categories as defined in the judge
const (
	FailureCategorySemanticMismatch   = "semantic_mismatch"
	FailureCategoryMissingInformation = "missing_information"
	FailureCategoryContainsExtraInfo  = "contains_extra_info"
	FailureCategoryNA                 = "n/a"
)

// BuildJudgeResponse creates a proper chat completion response for judge calls
// The response includes a tool call to submit_judgement with the given result
func BuildJudgeResponse(result JudgeResult) *Response {
	args, _ := json.Marshal(result)
	return &Response{
		Body: &ChatCompletionResponse{
			ID:      "chatcmpl-mock-judge",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{{
				Index: 0,
				Message: Message{
					Role: "assistant",
					ToolCalls: []ToolCall{{
						ID:   "call_mock_judge_001",
						Type: "function",
						Function: FunctionCall{
							Name:      "submit_judgement",
							Arguments: string(args),
						},
					}},
				},
				FinishReason: "tool_calls",
			}},
			Usage: &Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		},
	}
}

// JudgePass creates a response where the judge passes with the given reason
func JudgePass(reason string) *Response {
	return BuildJudgeResponse(JudgeResult{
		Passed:          true,
		Reason:          reason,
		FailureCategory: FailureCategoryNA,
	})
}

// JudgeFail creates a response where the judge fails with a category and reason
// Valid categories: "semantic_mismatch", "missing_information", "contains_extra_info"
func JudgeFail(category, reason string) *Response {
	return BuildJudgeResponse(JudgeResult{
		Passed:          false,
		Reason:          reason,
		FailureCategory: category,
	})
}

// JudgeFailSemanticMismatch creates a response for semantic mismatch failure
func JudgeFailSemanticMismatch(reason string) *Response {
	return JudgeFail(FailureCategorySemanticMismatch, reason)
}

// JudgeFailMissingInformation creates a response for missing information failure
func JudgeFailMissingInformation(reason string) *Response {
	return JudgeFail(FailureCategoryMissingInformation, reason)
}

// JudgeFailContainsExtraInfo creates a response for contains extra info failure
func JudgeFailContainsExtraInfo(reason string) *Response {
	return JudgeFail(FailureCategoryContainsExtraInfo, reason)
}

// JudgeError creates an API error response
func JudgeError(statusCode int, message string) *Response {
	return &Response{
		StatusCode: statusCode,
		Error: &APIError{
			Error: APIErrorDetail{
				Message: message,
				Type:    "server_error",
				Code:    "internal_error",
			},
		},
	}
}

// JudgeTimeout creates a response with a delay that simulates a timeout
// The delay should exceed the client's timeout to trigger a timeout error
func JudgeTimeout(delay time.Duration) *Response {
	// Return a valid response but with a long delay
	// The client will timeout before receiving it
	resp := JudgePass("This response was delayed")
	resp.Delay = delay
	return resp
}

// JudgeRateLimited creates a rate limit error response (429)
func JudgeRateLimited(message string) *Response {
	if message == "" {
		message = "Rate limit exceeded. Please retry after some time."
	}
	return &Response{
		StatusCode: 429,
		Error: &APIError{
			Error: APIErrorDetail{
				Message: message,
				Type:    "rate_limit_error",
				Code:    "rate_limit_exceeded",
			},
		},
	}
}

// JudgeServiceUnavailable creates a 503 service unavailable response
func JudgeServiceUnavailable(message string) *Response {
	if message == "" {
		message = "The server is currently unavailable. Please try again later."
	}
	return &Response{
		StatusCode: 503,
		Error: &APIError{
			Error: APIErrorDetail{
				Message: message,
				Type:    "server_error",
				Code:    "service_unavailable",
			},
		},
	}
}

// JudgeInvalidResponse creates a response with malformed JSON in the tool call
// This can be used to test error handling when the judge returns invalid data
func JudgeInvalidResponse() *Response {
	return &Response{
		Body: &ChatCompletionResponse{
			ID:      "chatcmpl-mock-judge-invalid",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{{
				Index: 0,
				Message: Message{
					Role: "assistant",
					ToolCalls: []ToolCall{{
						ID:   "call_mock_judge_invalid",
						Type: "function",
						Function: FunctionCall{
							Name:      "submit_judgement",
							Arguments: "{invalid json",
						},
					}},
				},
				FinishReason: "tool_calls",
			}},
		},
	}
}

// JudgeWrongTool creates a response where the judge calls the wrong tool
// This can be used to test error handling when the judge doesn't call submit_judgement
func JudgeWrongTool(toolName string, args map[string]any) *Response {
	argsJSON, _ := json.Marshal(args)
	return &Response{
		Body: &ChatCompletionResponse{
			ID:      "chatcmpl-mock-judge-wrong-tool",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{{
				Index: 0,
				Message: Message{
					Role: "assistant",
					ToolCalls: []ToolCall{{
						ID:   "call_mock_judge_wrong",
						Type: "function",
						Function: FunctionCall{
							Name:      toolName,
							Arguments: string(argsJSON),
						},
					}},
				},
				FinishReason: "tool_calls",
			}},
		},
	}
}

// JudgeNoToolCall creates a response with no tool calls (just a text message)
// This can be used to test error handling when the judge fails to call any tool
func JudgeNoToolCall(message string) *Response {
	return &Response{
		Body: &ChatCompletionResponse{
			ID:      "chatcmpl-mock-judge-no-tool",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: message,
				},
				FinishReason: "stop",
			}},
		},
	}
}

// JudgeMultipleToolCalls creates a response with multiple tool calls
// This can be used to test error handling when the judge calls multiple tools
func JudgeMultipleToolCalls(results ...JudgeResult) *Response {
	toolCalls := make([]ToolCall, len(results))
	for i, result := range results {
		args, _ := json.Marshal(result)
		toolCalls[i] = ToolCall{
			ID:   fmt.Sprintf("call_mock_judge_%03d", i),
			Type: "function",
			Function: FunctionCall{
				Name:      "submit_judgement",
				Arguments: string(args),
			},
		}
	}
	return &Response{
		Body: &ChatCompletionResponse{
			ID:      "chatcmpl-mock-judge-multi",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{{
				Index: 0,
				Message: Message{
					Role:      "assistant",
					ToolCalls: toolCalls,
				},
				FinishReason: "tool_calls",
			}},
		},
	}
}

// JudgeEmptyChoices creates a response with no choices
// This can be used to test error handling when the API returns no choices
func JudgeEmptyChoices() *Response {
	return &Response{
		Body: &ChatCompletionResponse{
			ID:      "chatcmpl-mock-judge-empty",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{},
		},
	}
}
