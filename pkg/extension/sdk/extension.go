package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/mcpchecker/mcpchecker/pkg/extension/protocol"
	"golang.org/x/exp/jsonrpc2"
)

// Extension represents an extension that can be run as a JSON-RPC server.
type Extension struct {
	mu           sync.RWMutex
	info         ExtensionInfo
	operations   map[string]*extensionOperation
	onInitialize InitializeHandler

	// conn is set when the extension is running
	conn *jsonrpc2.Connection
	// cancel is used to cancel the connection context on shutdown
	cancel context.CancelFunc
	// shutdown is set to true when shutdown has been requested
	shutdown bool
}

// ExtensionInfo contains metadata about the extension.
type ExtensionInfo struct {
	Name        string
	Version     string
	Description string
}

// InitializeHandler is called when the extension receives an initialize request.
type InitializeHandler func(config map[string]any) error

// ExtensionOption is a functional option for configuring an Extension.
type ExtensionOption func(*Extension)

// NewExtension creates a new Extension with the given info and options.
func NewExtension(info ExtensionInfo, opts ...ExtensionOption) *Extension {
	e := &Extension{
		info:       info,
		operations: make(map[string]*extensionOperation),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithInitializeHandler sets the handler called during initialization.
func WithInitializeHandler(handler InitializeHandler) ExtensionOption {
	return func(e *Extension) {
		e.onInitialize = handler
	}
}

// initialize calls the initialization handler with the given config.
// This is used internally for one-shot mode when --config is provided via CLI.
func (e *Extension) initialize(config map[string]any) error {
	if e.onInitialize != nil {
		return e.onInitialize(config)
	}
	return nil
}

// AddOperation registers an operation with its handler.
func (e *Extension) AddOperation(o *Operation, handler OperationHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.operations[o.name] = &extensionOperation{
		operation: o,
		handler:   handler,
	}
}

// Run starts the extension, listening on stdin/stdout for JSON-RPC messages.
// This blocks until the connection is closed or an error occurs.
// EOF is treated as a clean shutdown (returns nil).
//
// For one-shot mode, pass --config with a JSON object to pre-initialize:
//
//	echo '{"jsonrpc":"2.0",...}' | ./extension --config '{"kubeconfig":"/path/to/config"}'
func (e *Extension) Run(ctx context.Context) error {
	// Check for --config flag for one-shot mode initialization
	if err := e.parseAndInitializeFromArgs(); err != nil {
		return fmt.Errorf("failed to initialize from args: %w", err)
	}

	// Create a cancellable context so we can interrupt reads on shutdown
	connCtx, cancel := context.WithCancel(ctx)

	conn, err := jsonrpc2.Dial(connCtx, &stdioDialer{}, &jsonrpc2.ConnectionOptions{
		Handler: e,
		Framer:  protocol.NewlineFramer(),
	})
	if err != nil {
		cancel()
		return fmt.Errorf("failed to start extension: %w", err)
	}

	e.mu.Lock()
	e.conn = conn
	e.cancel = cancel
	e.mu.Unlock()

	err = conn.Wait()

	// Treat EOF or context cancellation as clean shutdown
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

// parseAndInitializeFromArgs checks for --config flag and initializes if present.
func (e *Extension) parseAndInitializeFromArgs() error {
	for i, arg := range os.Args[1:] {
		if arg == "--config" && i+1 < len(os.Args[1:]) {
			configJSON := os.Args[i+2]
			var config map[string]any
			if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
				return fmt.Errorf("invalid --config JSON: %w", err)
			}
			return e.initialize(config)
		}
		// Also handle --config=value format
		if len(arg) > 9 && arg[:9] == "--config=" {
			configJSON := arg[9:]
			var config map[string]any
			if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
				return fmt.Errorf("invalid --config JSON: %w", err)
			}
			return e.initialize(config)
		}
	}
	return nil
}

// Handle processes incoming JSON-RPC requests.
func (e *Extension) Handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	switch req.Method {
	case protocol.MethodInitialize:
		return e.handleInitialize(ctx, req)
	case protocol.MethodExecute:
		return e.handleExecute(ctx, req)
	case protocol.MethodShutdown:
		return e.handleShutdown(ctx, req)
	default:
		return nil, jsonrpc2.NewError(protocol.CodeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (e *Extension) handleInitialize(_ context.Context, req *jsonrpc2.Request) (*protocol.InitializeResult, error) {
	var params protocol.InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, jsonrpc2.NewError(protocol.CodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
	}

	if params.ProtocolVersion != protocol.ProtocolVersion {
		return nil, jsonrpc2.NewError(
			protocol.CodeInvalidParams,
			fmt.Sprintf("unsupported protocol version: %s (expected %s)", params.ProtocolVersion, protocol.ProtocolVersion),
		)
	}

	if e.onInitialize != nil {
		if err := e.onInitialize(params.Config); err != nil {
			return nil, jsonrpc2.NewError(protocol.CodeInternalError, fmt.Sprintf("initialization failed: %v", err))
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	operations := make(map[string]*protocol.Operation, len(e.operations))
	for name, op := range e.operations {
		operations[name] = &protocol.Operation{
			Description: op.operation.description,
			Params:      op.operation.params,
		}
	}

	return &protocol.InitializeResult{
		Name:            e.info.Name,
		Version:         e.info.Version,
		ProtocolVersion: protocol.ProtocolVersion,
		Description:     e.info.Description,
		Operations:      operations,
	}, nil
}

func (e *Extension) handleExecute(ctx context.Context, req *jsonrpc2.Request) (*protocol.ExecuteResult, error) {
	var params protocol.ExecuteParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, jsonrpc2.NewError(protocol.CodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
	}

	e.mu.RLock()
	op, ok := e.operations[params.Operation]
	e.mu.RUnlock()

	if !ok {
		return &protocol.ExecuteResult{
			Success: false,
			Error:   fmt.Sprintf("unknown operation: %s", params.Operation),
		}, nil
	}

	opReq := &OperationRequest{
		Args:    params.Args,
		Context: params.Context,
	}

	result, err := op.handler(ctx, opReq)
	if err != nil {
		return &protocol.ExecuteResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return result, nil
}

func (e *Extension) handleShutdown(_ context.Context, _ *jsonrpc2.Request) (any, error) {
	e.mu.Lock()
	e.shutdown = true
	cancel := e.cancel
	e.mu.Unlock()

	// Cancel the connection context to interrupt any blocked reads.
	// This is done in a goroutine to allow the response to be sent first.
	if cancel != nil {
		go cancel()
	}

	return struct{}{}, nil
}

// Log sends a log message to the client.
func (e *Extension) Log(ctx context.Context, level, message string, data map[string]any) error {
	e.mu.RLock()
	conn := e.conn
	shutdown := e.shutdown
	e.mu.RUnlock()

	if conn == nil || shutdown {
		return fmt.Errorf("extension not running")
	}

	params := protocol.LogParams{
		Level:   level,
		Message: message,
		Data:    data,
	}

	return conn.Notify(ctx, protocol.MethodLog, params)
}

// LogDebug sends a debug log message.
func (e *Extension) LogDebug(ctx context.Context, message string, data map[string]any) error {
	return e.Log(ctx, "debug", message, data)
}

// LogInfo sends an info log message.
func (e *Extension) LogInfo(ctx context.Context, message string, data map[string]any) error {
	return e.Log(ctx, "info", message, data)
}

// LogWarn sends a warning log message.
func (e *Extension) LogWarn(ctx context.Context, message string, data map[string]any) error {
	return e.Log(ctx, "warn", message, data)
}

// LogError sends an error log message.
func (e *Extension) LogError(ctx context.Context, message string, data map[string]any) error {
	return e.Log(ctx, "error", message, data)
}

// stdioDialer implements jsonrpc2.Dialer for stdin/stdout communication.
type stdioDialer struct{}

var _ jsonrpc2.Dialer = &stdioDialer{}

func (d *stdioDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &stdioConn{}, nil
}

type stdioConn struct{}

func (c *stdioConn) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (c *stdioConn) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (c *stdioConn) Close() error {
	// Close both stdin and stdout to fully signal connection closure.
	stdinErr := os.Stdin.Close()
	stdoutErr := os.Stdout.Close()
	if stdinErr != nil {
		return stdinErr
	}
	return stdoutErr
}
