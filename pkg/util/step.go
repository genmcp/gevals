package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Step struct {
	Inline string `json:"inline"`
	File   string `json:"file"`
}

func (s *Step) IsEmpty() bool {
	if s == nil {
		return true
	}

	return s.File == "" && s.Inline == ""
}

func (s *Step) Run(ctx context.Context) (string, error) {
	var cmd *exec.Cmd
	var err error

	if s.Inline != "" {
		cmd, err = s.createInlineCommand(ctx)
		if err != nil {
			return "", err
		}
	} else {
		cmd, err = s.createFileCommand(ctx)
		if err != nil {
			return "", err
		}
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// createInlineCommand executes inline scripts with shebang support.
// Scripts with shebangs are written to temp files in the current directory to preserve relative paths.
func (s *Step) createInlineCommand(ctx context.Context) (*exec.Cmd, error) {
	if strings.HasPrefix(strings.TrimSpace(s.Inline), "#!") {
		tmpFile, err := os.CreateTemp(".", ".mcpchecker-step-*.sh")
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
		go func() {
			<-ctx.Done()
			os.Remove(tmpPath)
		}()
		return cmd, nil
	}

	shell := GetShell()
	cmd := exec.CommandContext(ctx, shell)
	cmd.Stdin = strings.NewReader(s.Inline)
	return cmd, nil
}

// createFileCommand executes a script file directly to respect its shebang.
func (s *Step) createFileCommand(ctx context.Context) (*exec.Cmd, error) {
	if err := ensureExecutable(s.File); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, s.File)
	// Set working directory to the script's directory so relative paths work
	cmd.Dir = filepath.Dir(s.File)
	return cmd, nil
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

func (s *Step) GetValue() (string, error) {
	if s.Inline != "" {
		return s.Inline, nil
	}

	b, err := os.ReadFile(s.File)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// GetShell returns the shell to use for executing scripts.
// It checks the SHELL environment variable and defaults to /usr/bin/bash if not set.
func GetShell() string {
	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		shell = "/usr/bin/bash"
	}

	return shell
}
