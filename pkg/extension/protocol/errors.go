package protocol

// NOTE: This package uses golang.org/x/exp/jsonrpc2, which is an experimental
// module. This dependency is intentionally used for its JSON-RPC 2.0 implementation.
// The module version is pinned in go.mod. If migrating to a different implementation,
// update all imports in protocol/, client/, and framer.go accordingly.
import "golang.org/x/exp/jsonrpc2"

// Standard JSON-RPC 2.0 error codes
const (
	CodeMethodNotFound int64 = -32601
	CodeInvalidParams  int64 = -32602
	CodeInternalError  int64 = -32603
)

// Extension-specific error codes (reserved range -32000 to -32099)
const (
	CodeOperationFailed  int64 = -32000
	CodeOperationTimeout int64 = -32001
)

func OperationFailedError(msg string) error {
	return jsonrpc2.NewError(CodeOperationFailed, msg)
}

func OperationTimeoutError(msg string) error {
	return jsonrpc2.NewError(CodeOperationTimeout, msg)
}
