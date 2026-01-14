package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TODO: Add template support for File and Inline fields once we figure out
// how to handle escaping conflicts between template syntax and shell escapes.
type ScriptStepConfig struct {
	File            string `json:"file,omitempty"`
	Inline          string `json:"inline,omitempty"`
	Timeout         string `json:"timeout,omitempty"`
	ContinueOnError bool   `json:"continueOnError,omitempty"`
}

type ScriptStep struct {
	File            string
	Inline          string
	Timeout         time.Duration
	ContinueOnError bool
}

var _ StepRunner = &ScriptStep{}

func ParseScriptStep(raw json.RawMessage) (StepRunner, error) {
	cfg := &ScriptStepConfig{}

	err := json.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}

	return NewScriptStep(cfg)
}

func NewScriptStep(cfg *ScriptStepConfig) (*ScriptStep, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	step := &ScriptStep{
		File:            cfg.File,
		Inline:          cfg.Inline,
		ContinueOnError: cfg.ContinueOnError,
	}

	if cfg.Timeout != "" {
		timeout, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout: %w", err)
		}
		step.Timeout = timeout
	} else {
		step.Timeout = DefaultTimeout
	}

	return step, nil
}

func (s *ScriptStep) Execute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	for k, v := range input.Env {
		err := os.Setenv(k, v)
		if err != nil {
			return nil, fmt.Errorf("failed to set env var '%s' to value '%s': %w", k, v, err)
		}
	}
	defer func() {
		for k := range input.Env {
			_ = os.Unsetenv(k)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	var cmd *exec.Cmd
	var err error

	if s.Inline != "" {
		cmd, err = s.createInlineCommand(ctx, input.Workdir)
	} else {
		cmd, err = s.createFileCommand(ctx, input.Workdir)
	}
	if err != nil {
		return s.handleError(err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return s.handleError(fmt.Errorf("script execution failed: %w\noutput: %s", err, string(out)))
	}

	return &StepOutput{
		Type:    "script",
		Success: true,
		Message: string(out),
	}, nil
}

// createInlineCommand executes inline scripts with shebang support.
// Scripts with shebangs are written to temp files in the current directory to preserve relative paths.
func (s *ScriptStep) createInlineCommand(ctx context.Context, workdir string) (*exec.Cmd, error) {
	if strings.HasPrefix(strings.TrimSpace(s.Inline), "#!") {
		tmpFile, err := os.CreateTemp(workdir, ".gevals-step-*.sh")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp script file: %w", err)
		}
		tmpPath := tmpFile.Name()

		if _, err := tmpFile.WriteString(s.Inline); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return nil, fmt.Errorf("failed to write temp script: %w", err)
		}
		tmpFile.Close()

		if err := ensureExecutable(tmpPath); err != nil {
			os.Remove(tmpPath)
			return nil, err
		}

		cmd := exec.CommandContext(ctx, tmpPath)
		cmd.Dir = workdir
		go func() {
			<-ctx.Done()
			os.Remove(tmpPath)
		}()
		return cmd, nil
	}

	shell := getShell()
	cmd := exec.CommandContext(ctx, shell)
	cmd.Stdin = strings.NewReader(s.Inline)
	cmd.Dir = workdir
	return cmd, nil
}

// createFileCommand executes a script file directly to respect its shebang.
func (s *ScriptStep) createFileCommand(ctx context.Context, workdir string) (*exec.Cmd, error) {
	file := s.File

	// If workdir is set and file is relative, resolve it
	if workdir != "" && !filepath.IsAbs(file) {
		file = filepath.Join(workdir, file)
	}

	if err := ensureExecutable(file); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, file)
	// Set working directory to the script's directory so relative paths work
	cmd.Dir = filepath.Dir(file)
	return cmd, nil
}

func (s *ScriptStep) handleError(err error) (*StepOutput, error) {
	if s.ContinueOnError {
		return &StepOutput{
			Type:    "script",
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return nil, err
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Mode()&0100 != 0 {
		return nil
	}

	if err := os.Chmod(path, info.Mode()|0111); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	return nil
}

func (cfg *ScriptStepConfig) Validate() error {
	numDefined := 0
	if cfg.File != "" {
		numDefined++
	}
	if cfg.Inline != "" {
		numDefined++
	}

	if numDefined != 1 {
		return fmt.Errorf("exactly one of 'file' or 'inline' must be defined on script step")
	}

	return nil
}

func getShell() string {
	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		shell = "/usr/bin/bash"
	}

	return shell
}
