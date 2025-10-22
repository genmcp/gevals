package util

import (
	"context"
	"os"
	"os/exec"
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

	shell := GetShell()

	if s.Inline != "" {
		cmd = exec.CommandContext(ctx, shell)
		cmd.Stdin = strings.NewReader(s.Inline)
	} else {
		cmd = exec.CommandContext(ctx, shell, s.File)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
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
