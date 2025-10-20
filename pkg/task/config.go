package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	SetupScript   *util.Step `json:"setup,omitempty"`
	CleanupScript *util.Step `json:"cleanup,omitempty"`
	VerifyScript  *util.Step `json:"verify,omitempty"`
	Prompt        *util.Step `json:"prompt,omitempty"`
}

func (t *TaskSpec) UnmarshalJSON(data []byte) error {
	return util.UnmarshalWithKind(data, t, KindTask)
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
	if err := resolveStepPath(spec.Steps.VerifyScript, basePath); err != nil {
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
