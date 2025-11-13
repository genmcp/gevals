package mcpproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerEnvForwarding(t *testing.T) {
	// Build the mock MCP server binary
	mockServerPath := filepath.Join(t.TempDir(), "mcpEnvDumpTest")
	
	// Get the absolute path to the testdata directory
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	testdataPath := filepath.Join(testDir, "testdata", "mcpEnvDumpTest")
	
	buildCmd := exec.Command("go", "build", "-o", mockServerPath, testdataPath)
	var stderr bytes.Buffer
	buildCmd.Stderr = &stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build mock MCP server: %v\nStderr: %s", err, stderr.String())
	}

	tests := []struct {
		name        string
		env         map[string]string
		setupEnv    map[string]string // Environment to set before expansion
		expectedEnv map[string]string // Expected env vars in dump file
		expectError bool
	}{
		{
			name: "simple env forwarding",
			env: map[string]string{
				"TEST_SIMPLE": "simple-value",
			},
			expectedEnv: map[string]string{
				"TEST_SIMPLE": "simple-value",
			},
		},
		{
			name: "env with expansion default used",
			env: map[string]string{
				"TEST_EXPANDED": "${TEST_BASE_URL:-http://localhost:8080}",
			},
			setupEnv: map[string]string{},
			expectedEnv: map[string]string{
				"TEST_EXPANDED": "http://localhost:8080",
			},
		},
		{
			name: "env with expansion var set",
			env: map[string]string{
				"TEST_EXPANDED": "${TEST_BASE_URL:-http://localhost:8080}",
			},
			setupEnv: map[string]string{
				"TEST_BASE_URL": "http://custom:9090",
			},
			expectedEnv: map[string]string{
				"TEST_EXPANDED": "http://custom:9090",
			},
		},
		{
			name: "multiple env vars",
			env: map[string]string{
				"TEST_VAR1": "value1",
				"TEST_VAR2": "${TEST_VAR2_DEFAULT:-default2}",
			},
			setupEnv: map[string]string{},
			expectedEnv: map[string]string{
				"TEST_VAR1": "value1",
				"TEST_VAR2": "default2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env
			originalEnv := make(map[string]string)
			for k, v := range tt.setupEnv {
				if orig, ok := os.LookupEnv(k); ok {
					originalEnv[k] = orig
				}
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.setupEnv {
					if orig, ok := originalEnv[k]; ok {
						os.Setenv(k, orig)
					} else {
						os.Unsetenv(k)
					}
				}
			}()

			// Create temp directory for output file
			tempDir := t.TempDir()
			envDumpFile := filepath.Join(tempDir, "env-dump.json")

			// Create server config
			config := &ServerConfig{
				Command: mockServerPath,
				Args:    []string{},
				Env:     tt.env,
			}

			// Add ENV_DUMP_FILE to config env
			if config.Env == nil {
				config.Env = make(map[string]string)
			}
			config.Env["ENV_DUMP_FILE"] = envDumpFile

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Create and start server
			server, err := NewProxyServerForConfig(ctx, "mcpEnvDumpTest", config)
			if err != nil {
				// Note: stderr from the MCP server process is automatically included
				// in the error message from NewProxyServerForConfig
				t.Fatalf("Failed to create server (failed checking environment variables?): %v", err)
			}

			// Start server in background
			serverErr := make(chan error, 1)
			go func() {
				serverErr <- server.Run(ctx)
			}()

			// Wait for server to be ready
			err = server.WaitReady(ctx)
			require.NoError(t, err, "Server failed to become ready")

			// Wait for the initialize request to be processed and file to be written
			// The MCP client automatically sends initialize when connecting, so we just need to wait
			maxWait := 5 * time.Second
			checkInterval := 100 * time.Millisecond
			waited := time.Duration(0)
			for waited < maxWait {
				if _, err := os.Stat(envDumpFile); err == nil {
					// File exists, give it a moment to be fully written
					time.Sleep(100 * time.Millisecond)
					break
				}
				time.Sleep(checkInterval)
				waited += checkInterval
			}

			// Read and verify the env dump file
			data, err := os.ReadFile(envDumpFile)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "Failed to read env dump file")
			require.NotEmpty(t, data, "Env dump file is empty")

			var dumpedEnv map[string]string
			err = json.Unmarshal(data, &dumpedEnv)
			require.NoError(t, err, "Failed to parse env dump file")

			// Verify expected env vars are present
			for key, expectedValue := range tt.expectedEnv {
				actualValue, ok := dumpedEnv[key]
				assert.True(t, ok, "Expected env var %s not found in dump", key)
				assert.Equal(t, expectedValue, actualValue, "Env var %s has wrong value", key)
			}

			// Verify ENV_DUMP_FILE is present
			assert.Contains(t, dumpedEnv, "ENV_DUMP_FILE", "ENV_DUMP_FILE should be in dump")

			// Clean up
			cancel()
			server.Close()
		})
	}
}

