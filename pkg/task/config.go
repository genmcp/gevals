package task

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/genmcp/gevals/pkg/llmjudge"
	"github.com/genmcp/gevals/pkg/util"
	"sigs.k8s.io/yaml"
)

const (
	KindTask         = "Task"
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

type TaskSpec struct {
	Metadata TaskMetadata `json:"metadata"`
	Steps    TaskSteps    `json:"steps"`
}

type TaskMetadata struct {
	Name       string `json:"name"`
	Difficulty string `json:"difficulty"`
}

type TaskSteps struct {
	SetupScript   *util.Step  `json:"setup,omitempty"`
	CleanupScript *util.Step  `json:"cleanup,omitempty"`
	VerifyScript  *VerifyStep `json:"verify,omitempty"`
	Prompt        *util.Step  `json:"prompt,omitempty"`
	Assertions    *TaskAssertions `json:"assertions,omitempty"`
}

type VerifyStep struct {
	*util.Step
	*llmjudge.LLMJudgeTaskConfig
}

// TaskAssertions defines assertions for a task (duplicated from eval package to avoid circular dependency)
type TaskAssertions struct {
	// Tool assertions
	ToolsUsed    []ToolAssertion `json:"toolsUsed,omitempty"`
	RequireAny   []ToolAssertion `json:"requireAny,omitempty"`
	ToolsNotUsed []ToolAssertion `json:"toolsNotUsed,omitempty"`
	MinToolCalls *int            `json:"minToolCalls,omitempty"`
	MaxToolCalls *int            `json:"maxToolCalls,omitempty"`

	// Resource assertions
	ResourcesRead    []ResourceAssertion `json:"resourcesRead,omitempty"`
	ResourcesNotRead []ResourceAssertion `json:"resourcesNotRead,omitempty"`

	// Prompt assertions
	PromptsUsed    []PromptAssertion `json:"promptsUsed,omitempty"`
	PromptsNotUsed []PromptAssertion `json:"promptsNotUsed,omitempty"`

	// Order assertions
	CallOrder []CallOrderAssertion `json:"callOrder,omitempty"`

	// Efficiency assertions
	NoDuplicateCalls bool `json:"noDuplicateCalls,omitempty"`
}

type ToolAssertion struct {
	Server string `json:"server"`

	// Exactly one of Tool or ToolPattern should be set
	// If neither is set, matches any tool from the server
	Tool        string `json:"tool,omitempty"`
	ToolPattern string `json:"toolPattern,omitempty"` // regex pattern
}

type ResourceAssertion struct {
	Server string `json:"server"`

	// Exactly one of URI or URIPattern should be set
	// If neither is set, matches any resource from the server
	URI        string `json:"uri,omitempty"`
	URIPattern string `json:"uriPattern,omitempty"` // regex pattern
}

type PromptAssertion struct {
	Server string `json:"server"`

	// Exactly one of Prompt or PromptPattern should be set
	// If neither is set, matches any prompt from the server
	Prompt        string `json:"prompt,omitempty"`
	PromptPattern string `json:"promptPattern,omitempty"`
}

type CallOrderAssertion struct {
	Type   string `json:"type"` // "tool", "resource", "prompt"
	Server string `json:"server"`
	Name   string `json:"name"`
}

func (v *VerifyStep) IsEmpty() bool {
	if v == nil {
		return true
	}

	hasStep := v.Step != nil && !v.Step.IsEmpty()
	hasJudgeConfig := v.LLMJudgeTaskConfig != nil

	return !hasStep && !hasJudgeConfig
}

func (v *VerifyStep) Validate() error {
	if v == nil {
		return fmt.Errorf("verify step is nil")
	}

	hasStep := v.Step != nil && !v.Step.IsEmpty()
	hasJudgeConfig := v.LLMJudgeTaskConfig != nil

	// Must have exactly one verification method
	if !hasStep && !hasJudgeConfig {
		return fmt.Errorf("verify.inline, verify.file, verify.exact, or verify.contains must be set")
	}

	if hasStep && hasJudgeConfig {
		return fmt.Errorf("cannot specify both a verify script (inline/file) and llm judge config (exact/contains)")
	}

	// Validate LLM judge config if present
	if hasJudgeConfig {
		if err := v.LLMJudgeTaskConfig.Validate(); err != nil {
			return fmt.Errorf("invalid llm judge config: %w", err)
		}
	}

	return nil
}

func (t *TaskSpec) UnmarshalJSON(data []byte) error {
	type Doppleganger TaskSpec

	tmp := (*Doppleganger)(t)
	return util.UnmarshalWithKind(data, tmp, KindTask)
}

func Read(data []byte, basePath string) (*TaskSpec, error) {
	spec := &TaskSpec{}

	err := yaml.Unmarshal(data, spec)
	if err != nil {
		return nil, err
	}

	// Convert all relative file paths to absolute paths
	if err := resolveStepPath(spec.Steps.SetupScript, basePath); err != nil {
		return nil, fmt.Errorf("failed to resolve setup script path: %w", err)
	}
	if err := resolveStepPath(spec.Steps.CleanupScript, basePath); err != nil {
		return nil, fmt.Errorf("failed to resolve cleanup script path: %w", err)
	}
	if err := resolveStepPath(spec.Steps.VerifyScript.Step, basePath); err != nil {
		return nil, fmt.Errorf("failed to resolve verify script path: %w", err)
	}
	if err := resolveStepPath(spec.Steps.Prompt, basePath); err != nil {
		return nil, fmt.Errorf("failed to resolve prompt path: %w", err)
	}

	return spec, nil
}

func resolveStepPath(step *util.Step, basePath string) error {
	if step == nil || step.File == "" {
		return nil
	}

	// If the path is already absolute, leave it as-is
	if filepath.IsAbs(step.File) {
		return nil
	}

	// Convert relative path to absolute path based on the YAML file's directory
	absPath := filepath.Join(basePath, step.File)
	step.File = absPath

	return nil
}

func FromFile(path string) (*TaskSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s' for taskspec: %w", path, err)
	}

	// Convert to absolute path to ensure basePath is absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for '%s': %w", path, err)
	}

	basePath := filepath.Dir(absPath)

	return Read(data, basePath)
}
