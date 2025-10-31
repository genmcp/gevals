package llmjudge

import (
	"fmt"
	"os"
)

const (
	EvaluationModeExact    = "EXACT"
	EvaluationModeContains = "CONTAINS"
)

type LLMJudgeEvalConfig struct {
	Env *LLMJudgeEnvConfig `json:"env,omitempty"`
}

type LLMJudgeEnvConfig struct {
	BaseUrlKey   string `json:"baseUrlKey"`
	ApiKeyKey    string `json:"apiKeyKey"`
	ModelNameKey string `json:"modelNameKey"`
}

type LLMJudgeTaskConfig struct {
	Contains string `json:"contains,omitempty"`
	Exact    string `json:"exact,omitempty"`
}

func (cfg *LLMJudgeEvalConfig) BaseUrl() string {
	return os.Getenv(cfg.Env.BaseUrlKey)
}

func (cfg *LLMJudgeEvalConfig) ApiKey() string {
	return os.Getenv(cfg.Env.ApiKeyKey)
}

func (cfg *LLMJudgeEvalConfig) ModelName() string {
	return os.Getenv(cfg.Env.ModelNameKey)
}

func (cfg *LLMJudgeTaskConfig) EvaluationMode() string {
	if cfg.Exact != "" {
		return EvaluationModeExact
	}

	return EvaluationModeContains
}

func (cfg *LLMJudgeTaskConfig) ReferenceAnswer() string {
	if cfg.Exact != "" {
		return cfg.Exact
	}

	return cfg.Contains
}

func (cfg *LLMJudgeTaskConfig) Validate() error {
	if cfg.Exact == "" && cfg.Contains == "" {
		return fmt.Errorf("one of contains or exact must be specified")
	}

	if cfg.Exact != "" && cfg.Contains != "" {
		return fmt.Errorf("only one of contains or exact can be specified, not both")
	}

	return nil
}
