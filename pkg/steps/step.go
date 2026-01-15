package steps

import (
	"context"
	"encoding/json"
	"time"
)

const (
	DefaultTimeout = 5 * time.Minute
)

var (
	DefaultRegistry = &Registry{
		parsers:       make(map[string]Parser),
		prefixParsers: make(map[string]PrefixParser),
	}
)

type StepRunner interface {
	Execute(ctx context.Context, input *StepInput) (*StepOutput, error)
}

type StepInput struct {
	Env     map[string]string
	Workdir string
	Agent   *AgentContext
}

type StepOutput struct {
	Type    string            `json:"type,omitempty"`
	Success bool              `json:"success"`
	Message string            `json:"message,omitempty"`
	Outputs map[string]string `json:"outputs,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type AgentContext struct {
	Prompt string
	Output string
}

type StepConfig map[string]json.RawMessage

func init() {
	DefaultRegistry.Register("http", ParseHttpStep)
	DefaultRegistry.Register("script", ParseScriptStep)
	DefaultRegistry.Register("llmJudge", ParseLLMJudgeStep)
}
