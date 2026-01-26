package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mcpchecker/mcpchecker/pkg/extension/client"
	extprotocol "github.com/mcpchecker/mcpchecker/pkg/extension/protocol"
)

type extensionStep struct {
	alias     string
	operation string
	args      map[string]any
}

func NewExtensionParser(ctx context.Context, alias string) PrefixParser {
	return func(operation string, raw json.RawMessage) (StepRunner, error) {
		manager, ok := client.ManagerFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("failed to get extension manager from context")
		}

		ext, err := manager.Get(ctx, alias)
		if err != nil {
			return nil, fmt.Errorf("failed to get extension %q: %w", alias, err)
		}

		manifest := ext.Manifest()
		op, ok := manifest.Operations[operation]
		if !ok {
			return nil, fmt.Errorf("operation %q not declared in extension %q", operation, alias)
		}

		params, err := op.GetParams()
		if err != nil {
			return nil, fmt.Errorf("failed to get params for operation %s.%s: %w", alias, operation, err)
		}

		var args map[string]any
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, fmt.Errorf("failed to parse args: %w", err)
			}
		}

		err = params.Validate(args)
		if err != nil {
			return nil, fmt.Errorf("provided args did not match params for operation %s.%s: %w", alias, operation, err)
		}

		return &extensionStep{
			alias:     alias,
			operation: operation,
			args:      args,
		}, nil
	}
}

var _ StepRunner = &extensionStep{}

func (r *extensionStep) Execute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	manager, ok := client.ManagerFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get extension manager from context")
	}

	ext, err := manager.Get(ctx, r.alias)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension %q: %w", r.alias, err)
	}

	params := &extprotocol.ExecuteParams{
		Operation: r.operation,
		Args:      r.args,
		Context: extprotocol.ExecuteContext{
			Workdir: input.Workdir,
			Env:     input.Env,
		},
	}

	if input.Agent != nil {
		params.Context.Agent = &extprotocol.AgentContext{
			Prompt: input.Agent.Prompt,
			Output: input.Agent.Output,
		}
	}

	res, err := ext.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s.%s: %w", r.alias, r.operation, err)
	}

	return &StepOutput{
		Success: res.Success,
		Type:    r.alias + "." + r.operation,
		Message: res.Message,
		Error:   res.Error,
		Outputs: res.Outputs,
	}, nil
}
