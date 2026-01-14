// Package sdk provides a framework for building gevals extensions.
//
// Extensions are JSON-RPC 2.0 servers that communicate over stdio, allowing
// gevals to execute domain-specific operations during task setup, verification,
// and cleanup phases.
//
// # Creating an Extension
//
// Use [NewExtension] to create an extension, add operations with [Extension.AddOperation],
// then call [Extension.Run] to start serving:
//
//	ext := sdk.NewExtension(sdk.ExtensionInfo{
//	    Name:        "my-extension",
//	    Version:     "1.0.0",
//	    Description: "Example extension",
//	})
//
//	ext.AddOperation(
//	    sdk.NewOperation("greet",
//	        sdk.WithDescription("Say hello to someone"),
//	        sdk.WithParams(jsonschema.Schema{
//	            Type: "object",
//	            Properties: map[string]jsonschema.Schema{
//	                "name": {Type: "string"},
//	            },
//	        }),
//	    ),
//	    func(ctx context.Context, req *sdk.OperationRequest) (*sdk.OperationResult, error) {
//	        args, err := sdk.UnmarshalArgs[GreetArgs](req)
//	        if err != nil {
//	            return sdk.Failure(err), nil
//	        }
//	        return sdk.Success(fmt.Sprintf("Hello, %s!", args.Name)), nil
//	    },
//	)
//
//	if err := ext.Run(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// # Operations
//
// Operations are the actions your extension can perform. Each operation has:
//   - A name (used in execute requests)
//   - An optional description
//   - An optional JSON schema defining the expected parameters
//
// Create operations with [NewOperation] and functional options like [WithDescription]
// and [WithParams].
//
// # Handling Requests
//
// Operation handlers receive an [OperationRequest] containing:
//   - Args: The operation arguments (use [UnmarshalArgs] to convert to a typed struct)
//   - Context: Execution context including working directory, phase, and environment
//
// Return an [OperationResult] indicating success or failure. Use the helper functions
// [Success], [SuccessWithOutputs], [Failure], and [FailureWithMessage] for convenience.
//
// # Logging
//
// Extensions can send log messages to the client during operation execution:
//
//	ext.LogInfo(ctx, "Processing request", map[string]any{"file": filename})
//	ext.LogError(ctx, "Operation failed", map[string]any{"error": err.Error()})
package sdk
