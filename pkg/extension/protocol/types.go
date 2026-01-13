package protocol

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

const ProtocolVersion = "0.0.1"

const (
	MethodInitialize = "initialize"
	MethodExecute    = "execute"
	MethodShutdown   = "shutdown"
	MethodLog        = "log" // notification only
)

// InitializeParams is sent with the "initialize" method
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Config          map[string]any `json:"config,omitempty"`
}

// InitializeResult is returned form the "initialize" method
// This is the extension manifest
type InitializeResult struct {
	Name            string               `json:"name"`
	Version         string               `json:"version"`
	ProtocolVersion string               `json:"protocolVersion"`
	Description     string               `json:"description,omitempty"`
	Requires        []Requirement        `json:"requires,omitempty"`
	Operations      map[string]Operation `json:"operations"`
}

// Requirement describes prerequisistes for an extension to be able to run
type Requirement struct {
	Command string `json:"command"`
}

type Operation struct {
	Description string            `json:"description,omitempty"`
	Params      jsonschema.Schema `json:"params"`
	params      *jsonschema.Resolved
}

// GetParams returns the resolved params for the operation
// Where validation is needed, prefer GetParams over directly accessing Params
func (o *Operation) GetParams() (*jsonschema.Resolved, error) {
	if o.params != nil {
		return o.params, nil
	}

	resolved, err := o.Params.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve params schema: %w", err)
	}

	o.params = resolved

	return o.params, nil
}

// ExecuteParams is sent with the "execute" method
type ExecuteParams struct {
	Operation string         `json:"operation"`
	Args      any            `json:"args"` // Args MUST be json serializable
	Context   ExecuteContext `json:"context"`
}

type ExecuteContext struct {
	Workdir string            `json:"workdir"`
	Phase   string            `json:"phase"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout string            `json:"timeout,omitempty"`
	Agent   *AgentContext     `json:"agent,omitempty"`
}

type AgentContext struct {
	Prompt string `json:"prompt"`
	Output string `json:"output"`
}

// ExecuteResult is returned from the "execute" method
type ExecuteResult struct {
	Success bool              `json:"success"`
	Message string            `json:"message,omitempty"`
	Error   string            `json:"error,omitempty"`
	Outputs map[string]string `json:"outputs,omitempty"`
}

// LogParams is sent as a notification with the "log" method
type LogParams struct {
	Level   string         `json:"level"` // "debug", "info", "warn", "error"
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}
