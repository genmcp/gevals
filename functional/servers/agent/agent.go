package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EnvConfigPath is the environment variable for the config file path
const EnvConfigPath = "MOCK_AGENT_CONFIG"

// Run executes the mock agent with the given arguments.
// This is the main entry point called by the cmd/main.go binary.
func Run(ctx context.Context, args []string) error {
	// Parse arguments
	parsedArgs := parseArgs(args)

	// Load configuration
	configPath := parsedArgs.ConfigPath
	if configPath == "" {
		configPath = os.Getenv(EnvConfigPath)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load MCP config if provided
	var mcpConfig *MCPConfig
	if parsedArgs.MCPConfigPath != "" {
		mcpConfig, err = loadMCPConfig(parsedArgs.MCPConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load MCP config: %w", err)
		}
	}

	// Verify MCP connectivity if requested
	if config.VerifyMCPConnectivity && mcpConfig != nil {
		if err := verifyMCPConnectivity(ctx, mcpConfig); err != nil {
			return fmt.Errorf("MCP connectivity verification failed: %w", err)
		}
	}

	// Verify allowed tools if requested
	if len(config.VerifyAllowedTools) > 0 && mcpConfig != nil {
		if err := verifyAllowedTools(ctx, mcpConfig, config.VerifyAllowedTools); err != nil {
			return fmt.Errorf("allowed tools verification failed: %w", err)
		}
	}

	// Find matching behavior
	behavior := findMatchingBehavior(config, parsedArgs.Prompt)
	if behavior == nil {
		if config.DefaultError != "" {
			return fmt.Errorf("%s", config.DefaultError)
		}
		fmt.Print(config.DefaultResponse)
		return nil
	}

	// Check if this behavior should error
	if behavior.Error != "" {
		return fmt.Errorf("%s", behavior.Error)
	}

	// Execute tool calls if any
	if len(behavior.ToolCalls) > 0 && mcpConfig != nil {
		if err := executeToolCalls(ctx, mcpConfig, behavior.ToolCalls); err != nil {
			return fmt.Errorf("failed to execute tool calls: %w", err)
		}
	}

	// Output response
	fmt.Print(behavior.Response)
	return nil
}

// Args holds parsed command line arguments
type Args struct {
	ConfigPath    string
	MCPConfigPath string
	Prompt        string
	AllowedTools  []string
}

func parseArgs(args []string) Args {
	parsed := Args{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mcp-config":
			if i+1 < len(args) {
				parsed.MCPConfigPath = args[i+1]
				i++
			}
		case "--prompt", "--print", "-p":
			if i+1 < len(args) {
				parsed.Prompt = args[i+1]
				i++
			}
		case "--config":
			if i+1 < len(args) {
				parsed.ConfigPath = args[i+1]
				i++
			}
		case "--allowedTools":
			if i+1 < len(args) {
				// Parse space or comma separated tools
				toolsStr := args[i+1]
				parsed.AllowedTools = strings.FieldsFunc(toolsStr, func(r rune) bool {
					return r == ' ' || r == ','
				})
				i++
			}
		}
	}

	return parsed
}

func findMatchingBehavior(config *Config, prompt string) *Behavior {
	for i := range config.Behaviors {
		b := &config.Behaviors[i]

		// Check MatchAny
		if b.MatchAny {
			return b
		}

		// Check PromptContains
		if b.PromptContains != "" && strings.Contains(prompt, b.PromptContains) {
			return b
		}

		// Check PromptMatches (regex)
		if b.PromptMatches != "" {
			re, err := regexp.Compile(b.PromptMatches)
			if err == nil && re.MatchString(prompt) {
				return b
			}
		}
	}

	return nil
}

// MCPConfig represents the MCP server configuration file format
type MCPConfig struct {
	MCPServers map[string]*ServerConfig `json:"mcpServers"`
}

// ServerConfig represents a single MCP server configuration
type ServerConfig struct {
	URL string `json:"url"`
}

func loadMCPConfig(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config: %w", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	return &config, nil
}

// mcpServerInfo holds connection info and available tools for an MCP server
type mcpServerInfo struct {
	session *mcp.ClientSession
	tools   map[string]bool
}

// connectAndListTools connects to all MCP servers and lists their tools.
// Returns a map of server name to server info. Caller must close all sessions.
func connectAndListTools(ctx context.Context, mcpConfig *MCPConfig) (map[string]*mcpServerInfo, error) {
	servers := make(map[string]*mcpServerInfo)

	for name, serverCfg := range mcpConfig.MCPServers {
		if serverCfg.URL == "" {
			// Close any already-opened sessions before returning error
			for _, info := range servers {
				info.session.Close()
			}
			return nil, fmt.Errorf("server %q has no URL", name)
		}

		session, err := connectToMCP(ctx, serverCfg.URL)
		if err != nil {
			// Close any already-opened sessions before returning error
			for _, info := range servers {
				info.session.Close()
			}
			return nil, fmt.Errorf("failed to connect to server %q at %s: %w", name, serverCfg.URL, err)
		}

		info := &mcpServerInfo{
			session: session,
			tools:   make(map[string]bool),
		}

		for tool, err := range session.Tools(ctx, &mcp.ListToolsParams{}) {
			if err != nil {
				session.Close()
				for _, info := range servers {
					info.session.Close()
				}
				return nil, fmt.Errorf("failed to list tools from server %q: %w", name, err)
			}
			info.tools[tool.Name] = true
		}

		servers[name] = info
	}

	return servers, nil
}

// closeAllSessions closes all MCP sessions in the map
func closeAllSessions(servers map[string]*mcpServerInfo) {
	for _, info := range servers {
		info.session.Close()
	}
}

func verifyMCPConnectivity(ctx context.Context, mcpConfig *MCPConfig) error {
	servers, err := connectAndListTools(ctx, mcpConfig)
	if err != nil {
		return err
	}
	defer closeAllSessions(servers)

	for name, info := range servers {
		serverCfg := mcpConfig.MCPServers[name]
		fmt.Fprintf(os.Stderr, "Connected to MCP server %q (%s), found %d tools\n", name, serverCfg.URL, len(info.tools))
	}

	return nil
}

func verifyAllowedTools(ctx context.Context, mcpConfig *MCPConfig, allowedTools []string) error {
	servers, err := connectAndListTools(ctx, mcpConfig)
	if err != nil {
		return err
	}
	defer closeAllSessions(servers)

	// Collect all available tools from all servers
	availableTools := make(map[string]bool)
	for name, info := range servers {
		for toolName := range info.tools {
			availableTools[toolName] = true
			// Also add with server prefix for namespaced tools
			availableTools[fmt.Sprintf("%s__%s", name, toolName)] = true
		}
	}

	// Check that all allowed tools are available
	for _, tool := range allowedTools {
		// Extract just the tool name if it has a prefix (e.g., "mcp__server__tool")
		toolName := tool
		parts := strings.Split(tool, "__")
		if len(parts) >= 2 {
			toolName = parts[len(parts)-1]
		}

		if !availableTools[toolName] && !availableTools[tool] {
			return fmt.Errorf("allowed tool %q is not available from any MCP server", tool)
		}
	}

	return nil
}

func executeToolCalls(ctx context.Context, mcpConfig *MCPConfig, toolCalls []ToolCallSpec) error {
	for _, tc := range toolCalls {
		// Check for context cancellation before each tool call
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Find the server URL
		serverURL := ""
		if tc.Server != "" {
			if serverCfg, ok := mcpConfig.MCPServers[tc.Server]; ok {
				serverURL = serverCfg.URL
			}
		} else {
			// Deterministically select the first server (alphabetically by name)
			// when no specific server is specified
			var serverNames []string
			for name := range mcpConfig.MCPServers {
				serverNames = append(serverNames, name)
			}
			sort.Strings(serverNames)
			if len(serverNames) > 0 {
				serverURL = mcpConfig.MCPServers[serverNames[0]].URL
			}
		}

		if serverURL == "" {
			return fmt.Errorf("no MCP server found for tool call %q", tc.Name)
		}

		// Connect to the MCP server and call the tool
		result, err := callTool(ctx, serverURL, tc.Name, tc.Arguments)
		if err != nil {
			if tc.ExpectError {
				fmt.Fprintf(os.Stderr, "Tool %q returned expected error: %v\n", tc.Name, err)
				continue
			}
			return fmt.Errorf("failed to call tool %q: %w", tc.Name, err)
		}

		if tc.ExpectError {
			return fmt.Errorf("tool %q was expected to error but succeeded", tc.Name)
		}

		// Log the result for debugging
		if result != nil && len(result.Content) > 0 {
			fmt.Fprintf(os.Stderr, "Tool %q result: %v\n", tc.Name, result.Content)
		}
	}

	return nil
}

func connectToMCP(ctx context.Context, serverURL string) (*mcp.ClientSession, error) {
	transport := &mcp.StreamableClientTransport{
		Endpoint: serverURL,
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mock-agent",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return session, nil
}

func callTool(ctx context.Context, serverURL, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	session, err := connectToMCP(ctx, serverURL)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}
