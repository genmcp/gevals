package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
	"github.com/genmcp/gevals/pkg/mcp"
)

type Agent interface {
	Run(ctx context.Context, prompt string) (string, error)
}

type openaiAgent struct {
	client       *openai.Client
	mcpClients   []*mcp.Client
	model        shared.ChatModel
	systemPrompt string
}

func NewOpenAIAgent(url, apiKey, model, systemPrompt string) (*openaiAgent, error) {
	if url == "" || apiKey == "" {
		return nil, fmt.Errorf("both url and API key must be provided to create an openai agent")
	}

	var chatModel shared.ChatModel
	if model == "" {
		chatModel = openai.ChatModelGPT4 // default model
	} else {
		chatModel = shared.ChatModel(model)
	}

	client := openai.NewClient(
		option.WithBaseURL(url),
		option.WithAPIKey(apiKey),
	)

	return &openaiAgent{
		client:       &client,
		mcpClients:   make([]*mcp.Client, 0),
		model:        chatModel,
		systemPrompt: systemPrompt,
	}, nil
}

// AddMCPServer adds an MCP server to the agent
func (o *openaiAgent) AddMCPServer(ctx context.Context, serverURL string) error {
	mcpClient, err := mcp.NewClient(ctx, serverURL)
	if err != nil {
		return fmt.Errorf("failed to create MCP client for %s: %w", serverURL, err)
	}

	// Load available tools from the MCP server
	if err := mcpClient.LoadTools(ctx); err != nil {
		mcpClient.Close()
		return fmt.Errorf("failed to load MCP tools from %s: %w", serverURL, err)
	}

	o.mcpClients = append(o.mcpClients, mcpClient)
	return nil
}

func (o *openaiAgent) Run(ctx context.Context, prompt string) (string, error) {
	// Start conversation with system prompt (if provided) and user's prompt
	messages := []openai.ChatCompletionMessageParamUnion{}

	if o.systemPrompt != "" {
		messages = append(messages, openai.SystemMessage(o.systemPrompt))
	}

	messages = append(messages, openai.UserMessage(prompt))

	// Get available tools from all MCP clients
	var tools []openai.ChatCompletionToolUnionParam
	for _, mcpClient := range o.mcpClients {
		clientTools := mcpClient.GetTools()
		tools = append(tools, clientTools...)
	}

	// Agent loop - continue until we get a final response without tool calls
	for {
		params := openai.ChatCompletionNewParams{
			Model:    o.model,
			Messages: messages,
		}

		// Add tools if available
		if len(tools) > 0 {
			params.Tools = tools
		}

		// Make the chat completion request
		completion, err := o.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return "", fmt.Errorf("failed to create chat completion: %w", err)
		}

		if len(completion.Choices) == 0 {
			return "", fmt.Errorf("no completion choices returned")
		}

		choice := completion.Choices[0]
		message := choice.Message

		// Add the assistant's message to the conversation
		assistantMessage := openai.AssistantMessage(message.Content)
		messages = append(messages, assistantMessage)

		// If there are no tool calls, we're done
		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}

		// Execute tool calls and add results to conversation
		for _, toolCall := range message.ToolCalls {
			if toolCall.Function.Name == "" {
				continue
			}

			// Parse tool arguments
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("failed to parse tool arguments: %w", err)
			}

			// Find which MCP client has this tool and execute it
			result, err := o.callToolOnAnyClient(ctx, toolCall.Function.Name, args)
			if err != nil {
				result = fmt.Sprintf("Error calling tool: %v", err)
			}

			// Add tool result to conversation
			messages = append(messages, openai.ToolMessage(result, toolCall.ID))
		}
	}
}

// callToolOnAnyClient finds the MCP client that has the specified tool and calls it
func (o *openaiAgent) callToolOnAnyClient(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	// Search through all MCP clients to find one that has this tool
	for _, mcpClient := range o.mcpClients {
		tools := mcpClient.GetTools()
		for _, tool := range tools {
			// Check if this is a function tool with the matching name
			if funcDef := tool.GetFunction(); funcDef != nil && funcDef.Name == toolName {
				// Found the tool, call it on this client
				return mcpClient.CallTool(ctx, toolName, arguments)
			}
		}
	}

	return "", fmt.Errorf("tool %s not found in any MCP client", toolName)
}

// Close closes the agent and any associated resources
func (o *openaiAgent) Close() error {
	var errs []error
	for _, mcpClient := range o.mcpClients {
		if err := mcpClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d MCP clients: %v", len(errs), errs)
	}

	return nil
}