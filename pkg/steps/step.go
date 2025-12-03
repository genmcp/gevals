package steps

import (
	"context"
	"encoding/json"
	"time"
)

const (
	DefaultTimout = 5 * time.Minute
)

var (
	DefaultRegistry = &Registry{
		parsers: make(map[string]Parser),
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
	Success bool
	Message string
	Outputs map[string]string
	Error   string
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
