package mcpproxy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	basePath = "testdata"
)

func TestParseConfigFile(t *testing.T) {
	type serverTypes struct {
		isStdio bool
		isHttp  bool
	}

	tt := map[string]struct {
		file          string
		expected      *MCPConfig
		expectedTypes map[string]serverTypes
		expectErr     bool
	}{
		"basic": {
			file: "basic.json",
			expected: &MCPConfig{
				MCPServers: map[string]*ServerConfig{
					"filesystem": {
						Command: "npx",
						Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
					},
				},
			},
			expectedTypes: map[string]serverTypes{
				"filesystem": {isStdio: true},
			},
		},
		"with-env": {
			file: "with-env.json",
			expected: &MCPConfig{
				MCPServers: map[string]*ServerConfig{
					"postgres": {
						Command: "uvx",
						Args:    []string{"mcp-server-postgres"},
						Env: map[string]string{
							"POSTGRES_PASSWORD": "secret",
						},
					},
				},
			},
			expectedTypes: map[string]serverTypes{
				"postgres": {isStdio: true},
			},
		},
		"http-server": {
			file: "http-server.json",
			expected: &MCPConfig{
				MCPServers: map[string]*ServerConfig{
					"api-server": {
						Type: "http",
						URL:  "${API_BASE_URL:-https://api.example.com}/mcp",
						Headers: map[string]string{
							"Authorization": "Bearer ${API_KEY}",
						},
					},
				},
			},
			expectedTypes: map[string]serverTypes{
				"api-server": {isHttp: true},
			},
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			got, err := ParseConfigFile(fmt.Sprintf("%s/%s", basePath, tc.file))
			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
