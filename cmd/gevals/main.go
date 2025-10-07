package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/genmcp/gevals/pkg/agent"
)

var (
	mcpURL       string
	prompt       string
	openaiURL    string
	openaiKey    string
	model        string
	systemPrompt string
)

var rootCmd = &cobra.Command{
	Use:   "gevals-cli",
	Short: "A CLI tool that connects to an MCP server and runs an OpenAI agent",
	Long: `gevals-cli is a command-line interface that connects to a Model Context Protocol (MCP)
server and uses OpenAI's API to run an intelligent agent. The agent can interact with
tools provided by the MCP server to accomplish tasks.`,
	Example: `  gevals-cli --mcp-url http://localhost:3000 --prompt "What files are in the current directory?"
  gevals-cli --mcp-url http://localhost:3000 --prompt "Read the README file" --model gpt-4o`,
	RunE: runAgent,
}

func init() {
	// Required flags
	rootCmd.Flags().StringVar(&mcpURL, "mcp-url", "", "MCP server URL (required)")
	rootCmd.Flags().StringVar(&prompt, "prompt", "", "Prompt to send to the agent (required)")

	// Optional flags with environment variable defaults
	rootCmd.Flags().StringVar(&openaiURL, "openai-url", getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"), "OpenAI API base URL")
	rootCmd.Flags().StringVar(&openaiKey, "openai-key", getEnvOrDefault("OPENAI_API_KEY", ""), "OpenAI API key")
	rootCmd.Flags().StringVar(&model, "model", getEnvOrDefault("OPENAI_MODEL", "gpt-4"), "OpenAI model to use")
	rootCmd.Flags().StringVar(&systemPrompt, "system", getEnvOrDefault("SYSTEM_PROMPT", ""), "System prompt for the agent")

	// Mark required flags
	rootCmd.MarkFlagRequired("mcp-url")
	rootCmd.MarkFlagRequired("prompt")
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Validate OpenAI API key
	if openaiKey == "" {
		return fmt.Errorf("OpenAI API key must be provided via --openai-key flag or OPENAI_API_KEY environment variable")
	}

	// Create context
	ctx := context.Background()

	// Create the OpenAI agent
	fmt.Printf("Creating OpenAI agent with model: %s\n", model)
	agentInstance, err := agent.NewOpenAIAgent(openaiURL, openaiKey, model, systemPrompt)
	if err != nil {
		return fmt.Errorf("failed to create OpenAI agent: %w", err)
	}

	// Ensure cleanup
	defer func() {
		if err := agentInstance.Close(); err != nil {
			log.Printf("Warning: Failed to close agent cleanly: %v", err)
		}
	}()

	// Add the MCP server
	fmt.Printf("Connecting to MCP server: %s\n", mcpURL)
	if err := agentInstance.AddMCPServer(ctx, mcpURL); err != nil {
		return fmt.Errorf("failed to add MCP server: %w", err)
	}

	// Run the agent with the provided prompt
	fmt.Printf("Running agent with prompt: %s\n\n", prompt)

	result, err := agentInstance.Run(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Output the result
	fmt.Println("Agent Response:")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(result)

	return nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}