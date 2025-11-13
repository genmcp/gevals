package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    struct{}   `json:"capabilities"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func main() {
	// Get the output file path from environment
	envDumpFile := os.Getenv("ENV_DUMP_FILE")
	if envDumpFile == "" {
		fmt.Fprintf(os.Stderr, "ENV_DUMP_FILE environment variable not set\n")
		os.Exit(1)
	}

	// Read JSON-RPC requests from stdin
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// Invalid JSON, skip
			continue
		}

		// Handle initialize request
		if req.Method == "initialize" {
			// Dump environment variables filtered by TEST_ prefix
			envVars := make(map[string]string)
			for _, env := range os.Environ() {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					key := parts[0]
					value := parts[1]
					// Filter by TEST_ prefix or specific vars we care about
					if strings.HasPrefix(key, "TEST_") || key == "ENV_DUMP_FILE" {
						envVars[key] = value
					}
				}
			}

			// Write to file
			data, err := json.MarshalIndent(envVars, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to marshal env vars: %v\n", err)
				continue
			}

			if err := os.WriteFile(envDumpFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write env dump file: %v\n", err)
				continue
			}

			// Send initialize response
			result := InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo: ServerInfo{
					Name:    "mcpEnvDumpTest",
					Version: "1.0.0",
				},
			}

			resultJSON, _ := json.Marshal(result)
			response := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultJSON,
			}

			if err := encoder.Encode(response); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to encode response: %v\n", err)
			}
		} else {
			// For other methods, send a simple response
			response := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage("{}"),
			}
			if err := encoder.Encode(response); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to encode response: %v\n", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}
}

