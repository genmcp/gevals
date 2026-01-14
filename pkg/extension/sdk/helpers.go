package sdk

import (
	"encoding/json"
	"fmt"

	"github.com/genmcp/gevals/pkg/extension/protocol"
)

// UnmarshalArgs unmarshals the operation request args into the provided type.
// This handles the case where args arrive as map[string]any from JSON unmarshaling
// and need to be converted to a typed struct.
func UnmarshalArgs[T any](req *OperationRequest) (T, error) {
	var result T

	if req.Args == nil {
		return result, nil
	}

	// Re-marshal to JSON then unmarshal to the target type
	data, err := json.Marshal(req.Args)
	if err != nil {
		return result, fmt.Errorf("failed to marshal args: %w", err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return result, nil
}

// Success creates a successful operation result with a message.
func Success(message string) *protocol.ExecuteResult {
	return &protocol.ExecuteResult{
		Success: true,
		Message: message,
	}
}

// SuccessWithOutputs creates a successful operation result with a message and outputs.
func SuccessWithOutputs(message string, outputs map[string]string) *protocol.ExecuteResult {
	return &protocol.ExecuteResult{
		Success: true,
		Message: message,
		Outputs: outputs,
	}
}

// Failure creates a failed operation result from an error.
func Failure(err error) *protocol.ExecuteResult {
	return &protocol.ExecuteResult{
		Success: false,
		Error:   err.Error(),
	}
}

// FailureWithMessage creates a failed operation result with a message and error.
func FailureWithMessage(message string, err error) *protocol.ExecuteResult {
	return &protocol.ExecuteResult{
		Success: false,
		Message: message,
		Error:   err.Error(),
	}
}
