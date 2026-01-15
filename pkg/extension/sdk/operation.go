package sdk

import (
	"context"

	"github.com/genmcp/gevals/pkg/extension/protocol"
	"github.com/google/jsonschema-go/jsonschema"
)

// Operation defines an operation that an extension can perform.
type Operation struct {
	name        string
	description string
	params      jsonschema.Schema
}

// OperationOption is a functional option for configuring an Operation.
type OperationOption func(*Operation)

// NewOperation creates a new Operation with the given name and options.
func NewOperation(name string, opts ...OperationOption) *Operation {
	o := &Operation{name: name}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithDescription sets the description for the operation.
func WithDescription(desc string) OperationOption {
	return func(o *Operation) {
		o.description = desc
	}
}

// WithParams sets the JSON schema for the operation parameters.
func WithParams(schema jsonschema.Schema) OperationOption {
	return func(o *Operation) {
		o.params = schema
	}
}

// OperationRequest contains all the context and arguments for an operation execution.
type OperationRequest struct {
	// Args contains the arguments passed to the operation.
	// These should be unmarshaled into the appropriate type.
	Args any

	// Context contains execution context from the protocol.
	Context protocol.ExecuteContext
}

// OperationResult is an alias for the protocol ExecuteResult.
type OperationResult = protocol.ExecuteResult

// OperationHandler is a function that handles an operation execution.
type OperationHandler func(ctx context.Context, req *OperationRequest) (*OperationResult, error)

// extensionOperation pairs an operation definition with its handler.
type extensionOperation struct {
	operation *Operation
	handler   OperationHandler
}
