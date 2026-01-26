package testcase

import (
	"github.com/mcpchecker/mcpchecker/pkg/eval"
	"github.com/mcpchecker/mcpchecker/pkg/llmjudge"
)

// EvalConfig provides a fluent API for building eval configurations.
// An eval defines how to run and verify a set of tasks.
type EvalConfig struct {
	spec *eval.EvalSpec
}

// NewEvalConfig creates a new eval config builder
func NewEvalConfig() *EvalConfig {
	return &EvalConfig{
		spec: &eval.EvalSpec{
			Config: eval.EvalConfig{},
		},
	}
}

// Name sets the eval name
func (ec *EvalConfig) Name(name string) *EvalConfig {
	ec.spec.Metadata.Name = name
	return ec
}

// MCPConfigFile sets the path to the MCP server configuration file
func (ec *EvalConfig) MCPConfigFile(path string) *EvalConfig {
	ec.spec.Config.McpConfigFile = path
	return ec
}

// Agent configures the agent to use for evaluation
func (ec *EvalConfig) Agent(configure func(*AgentRefBuilder)) *EvalConfig {
	builder := &AgentRefBuilder{ref: &eval.AgentRef{}}
	configure(builder)
	ec.spec.Config.Agent = builder.ref
	return ec
}

// FileAgent sets a custom file-based agent
func (ec *EvalConfig) FileAgent(path string) *EvalConfig {
	ec.spec.Config.Agent = &eval.AgentRef{
		Type: "file",
		Path: path,
	}
	return ec
}

// ClaudeCodeAgent sets Claude Code as the agent
func (ec *EvalConfig) ClaudeCodeAgent() *EvalConfig {
	ec.spec.Config.Agent = &eval.AgentRef{
		Type: "builtin.claude-code",
	}
	return ec
}

// OpenAIAgent sets OpenAI-compatible agent with a model
func (ec *EvalConfig) OpenAIAgent(model string) *EvalConfig {
	ec.spec.Config.Agent = &eval.AgentRef{
		Type:  "builtin.openai-agent",
		Model: model,
	}
	return ec
}

// LLMJudge configures the LLM judge for evaluation
func (ec *EvalConfig) LLMJudge(configure func(*LLMJudgeConfigBuilder)) *EvalConfig {
	builder := &LLMJudgeConfigBuilder{config: &llmjudge.LLMJudgeEvalConfig{}}
	configure(builder)
	ec.spec.Config.LLMJudge = builder.config
	return ec
}

// TaskSet adds a task set with optional assertions
func (ec *EvalConfig) TaskSet(configure func(*TaskSetBuilder)) *EvalConfig {
	builder := &TaskSetBuilder{set: eval.TaskSet{}}
	configure(builder)
	ec.spec.Config.TaskSets = append(ec.spec.Config.TaskSets, builder.set)
	return ec
}

// TaskPath adds a single task by path
func (ec *EvalConfig) TaskPath(path string) *EvalConfig {
	ec.spec.Config.TaskSets = append(ec.spec.Config.TaskSets, eval.TaskSet{
		Path: path,
	})
	return ec
}

// TaskGlob adds tasks matching a glob pattern
func (ec *EvalConfig) TaskGlob(pattern string) *EvalConfig {
	ec.spec.Config.TaskSets = append(ec.spec.Config.TaskSets, eval.TaskSet{
		Glob: pattern,
	})
	return ec
}

// Build returns the eval spec
func (ec *EvalConfig) Build() *eval.EvalSpec {
	return ec.spec
}

// AgentRefBuilder builds agent reference configuration
type AgentRefBuilder struct {
	ref *eval.AgentRef
}

// Type sets the agent type
func (b *AgentRefBuilder) Type(t string) *AgentRefBuilder {
	b.ref.Type = t
	return b
}

// Path sets the agent configuration file path
func (b *AgentRefBuilder) Path(p string) *AgentRefBuilder {
	b.ref.Path = p
	return b
}

// Model sets the model name
func (b *AgentRefBuilder) Model(m string) *AgentRefBuilder {
	b.ref.Model = m
	return b
}

// LLMJudgeConfigBuilder builds LLM judge configuration.
// The LLM judge config uses environment variable keys, not direct values.
// Use the Env* methods to set the environment variable key names.
type LLMJudgeConfigBuilder struct {
	config *llmjudge.LLMJudgeEvalConfig
}

// EnvBaseURLKey sets the environment variable key for the base URL
func (b *LLMJudgeConfigBuilder) EnvBaseURLKey(key string) *LLMJudgeConfigBuilder {
	if b.config.Env == nil {
		b.config.Env = &llmjudge.LLMJudgeEnvConfig{}
	}
	b.config.Env.BaseUrlKey = key
	return b
}

// EnvAPIKeyKey sets the environment variable key for the API key
func (b *LLMJudgeConfigBuilder) EnvAPIKeyKey(key string) *LLMJudgeConfigBuilder {
	if b.config.Env == nil {
		b.config.Env = &llmjudge.LLMJudgeEnvConfig{}
	}
	b.config.Env.ApiKeyKey = key
	return b
}

// EnvModelKey sets the environment variable key for the model name
func (b *LLMJudgeConfigBuilder) EnvModelKey(key string) *LLMJudgeConfigBuilder {
	if b.config.Env == nil {
		b.config.Env = &llmjudge.LLMJudgeEnvConfig{}
	}
	b.config.Env.ModelNameKey = key
	return b
}

// UseDefaults sets up the default environment variable keys (OPENAI_BASE_URL, OPENAI_API_KEY, OPENAI_MODEL)
func (b *LLMJudgeConfigBuilder) UseDefaults() *LLMJudgeConfigBuilder {
	if b.config.Env == nil {
		b.config.Env = &llmjudge.LLMJudgeEnvConfig{}
	}
	b.config.Env.BaseUrlKey = "OPENAI_BASE_URL"
	b.config.Env.ApiKeyKey = "OPENAI_API_KEY"
	b.config.Env.ModelNameKey = "OPENAI_MODEL"
	return b
}

// TaskSetBuilder builds a task set configuration
type TaskSetBuilder struct {
	set eval.TaskSet
}

// Path sets a single task path
func (b *TaskSetBuilder) Path(path string) *TaskSetBuilder {
	b.set.Path = path
	b.set.Glob = ""
	return b
}

// Glob sets a glob pattern for matching tasks
func (b *TaskSetBuilder) Glob(pattern string) *TaskSetBuilder {
	b.set.Glob = pattern
	b.set.Path = ""
	return b
}

// LabelSelector sets the label selector for filtering tasks
func (b *TaskSetBuilder) LabelSelector(labels map[string]string) *TaskSetBuilder {
	b.set.LabelSelector = labels
	return b
}

// AddLabelSelector adds a single label to the label selector
func (b *TaskSetBuilder) AddLabelSelector(key, value string) *TaskSetBuilder {
	if b.set.LabelSelector == nil {
		b.set.LabelSelector = make(map[string]string)
	}
	b.set.LabelSelector[key] = value
	return b
}

// Assertions configures assertions for this task set
func (b *TaskSetBuilder) Assertions(configure func(*AssertionsBuilder)) *TaskSetBuilder {
	builder := &AssertionsBuilder{assertions: &eval.TaskAssertions{}}
	configure(builder)
	b.set.Assertions = builder.assertions
	return b
}

// AssertionsBuilder builds task assertions
type AssertionsBuilder struct {
	assertions *eval.TaskAssertions
}

// RequireTool adds a tool that must be used
func (b *AssertionsBuilder) RequireTool(server, tool string) *AssertionsBuilder {
	b.assertions.ToolsUsed = append(b.assertions.ToolsUsed, eval.ToolAssertion{
		Server: server,
		Tool:   tool,
	})
	return b
}

// RequireToolPattern adds a tool pattern that must match
func (b *AssertionsBuilder) RequireToolPattern(server, pattern string) *AssertionsBuilder {
	b.assertions.ToolsUsed = append(b.assertions.ToolsUsed, eval.ToolAssertion{
		Server:      server,
		ToolPattern: pattern,
	})
	return b
}

// RequireAnyTool adds a tool where at least one must be used
func (b *AssertionsBuilder) RequireAnyTool(server, tool string) *AssertionsBuilder {
	b.assertions.RequireAny = append(b.assertions.RequireAny, eval.ToolAssertion{
		Server: server,
		Tool:   tool,
	})
	return b
}

// ForbidTool adds a tool that must not be used
func (b *AssertionsBuilder) ForbidTool(server, tool string) *AssertionsBuilder {
	b.assertions.ToolsNotUsed = append(b.assertions.ToolsNotUsed, eval.ToolAssertion{
		Server: server,
		Tool:   tool,
	})
	return b
}

// ForbidToolPattern adds a tool pattern that must not match
func (b *AssertionsBuilder) ForbidToolPattern(server, pattern string) *AssertionsBuilder {
	b.assertions.ToolsNotUsed = append(b.assertions.ToolsNotUsed, eval.ToolAssertion{
		Server:      server,
		ToolPattern: pattern,
	})
	return b
}

// MinToolCalls sets the minimum number of tool calls required
func (b *AssertionsBuilder) MinToolCalls(n int) *AssertionsBuilder {
	b.assertions.MinToolCalls = &n
	return b
}

// MaxToolCalls sets the maximum number of tool calls allowed
func (b *AssertionsBuilder) MaxToolCalls(n int) *AssertionsBuilder {
	b.assertions.MaxToolCalls = &n
	return b
}

// RequireResource adds a resource that must be read
func (b *AssertionsBuilder) RequireResource(server, uri string) *AssertionsBuilder {
	b.assertions.ResourcesRead = append(b.assertions.ResourcesRead, eval.ResourceAssertion{
		Server: server,
		URI:    uri,
	})
	return b
}

// RequireResourcePattern adds a resource pattern that must match
func (b *AssertionsBuilder) RequireResourcePattern(server, pattern string) *AssertionsBuilder {
	b.assertions.ResourcesRead = append(b.assertions.ResourcesRead, eval.ResourceAssertion{
		Server:     server,
		URIPattern: pattern,
	})
	return b
}

// ForbidResource adds a resource that must not be read
func (b *AssertionsBuilder) ForbidResource(server, uri string) *AssertionsBuilder {
	b.assertions.ResourcesNotRead = append(b.assertions.ResourcesNotRead, eval.ResourceAssertion{
		Server: server,
		URI:    uri,
	})
	return b
}

// RequirePrompt adds a prompt that must be used
func (b *AssertionsBuilder) RequirePrompt(server, prompt string) *AssertionsBuilder {
	b.assertions.PromptsUsed = append(b.assertions.PromptsUsed, eval.PromptAssertion{
		Server: server,
		Prompt: prompt,
	})
	return b
}

// ForbidPrompt adds a prompt that must not be used
func (b *AssertionsBuilder) ForbidPrompt(server, prompt string) *AssertionsBuilder {
	b.assertions.PromptsNotUsed = append(b.assertions.PromptsNotUsed, eval.PromptAssertion{
		Server: server,
		Prompt: prompt,
	})
	return b
}

// CallOrderTool adds a tool to the expected call order
func (b *AssertionsBuilder) CallOrderTool(server, name string) *AssertionsBuilder {
	b.assertions.CallOrder = append(b.assertions.CallOrder, eval.CallOrderAssertion{
		Type:   "tool",
		Server: server,
		Name:   name,
	})
	return b
}

// CallOrderResource adds a resource to the expected call order
func (b *AssertionsBuilder) CallOrderResource(server, name string) *AssertionsBuilder {
	b.assertions.CallOrder = append(b.assertions.CallOrder, eval.CallOrderAssertion{
		Type:   "resource",
		Server: server,
		Name:   name,
	})
	return b
}

// CallOrderPrompt adds a prompt to the expected call order
func (b *AssertionsBuilder) CallOrderPrompt(server, name string) *AssertionsBuilder {
	b.assertions.CallOrder = append(b.assertions.CallOrder, eval.CallOrderAssertion{
		Type:   "prompt",
		Server: server,
		Name:   name,
	})
	return b
}

// NoDuplicateCalls requires that no duplicate tool calls are made
func (b *AssertionsBuilder) NoDuplicateCalls() *AssertionsBuilder {
	b.assertions.NoDuplicateCalls = true
	return b
}

// Re-export types for convenience
type (
	EvalSpec           = eval.EvalSpec
	EvalMetadata       = eval.EvalMetadata
	AgentRef           = eval.AgentRef
	TaskSet            = eval.TaskSet
	TaskAssertions     = eval.TaskAssertions
	ToolAssertion      = eval.ToolAssertion
	ResourceAssertion  = eval.ResourceAssertion
	PromptAssertion    = eval.PromptAssertion
	CallOrderAssertion = eval.CallOrderAssertion
	LLMJudgeEvalConfig = llmjudge.LLMJudgeEvalConfig
	LLMJudgeEnvConfig  = llmjudge.LLMJudgeEnvConfig
)
