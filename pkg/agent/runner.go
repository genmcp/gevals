package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type Runner interface {
	RunTask(ctx context.Context, task Task) (AgentResult, error)
}

type Task interface {
	GetMcpServerFiles() ([]string, error)
	GetMcpServers() []McpServer
	GetPrompt() string
}

type McpServer interface {
	GetName() string
	GetAllowedToolNames() []string
}

type AgentResult interface {
	GetOutput() string
}

type agentSpecRunner struct {
	*AgentSpec
}

type agentSpecRunnerResult struct {
	commandOuput string
}

func (a *agentSpecRunnerResult) GetOutput() string {
	return a.commandOuput
}

func NewRunnerForSpec(spec *AgentSpec) (Runner, error) {
	if spec == nil {
		return nil, fmt.Errorf("cannot create a Runner for a nil AgentSpec")
	}

	return &agentSpecRunner{
		AgentSpec: spec,
	}, nil
}

func (a *agentSpecRunner) RunTask(ctx context.Context, task Task) (AgentResult, error) {
	argTemplateMcpServer, err := template.ParseGlob(a.Commands.ArgTemplateMcpServer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argTemplateMcpServer: %w", err)
	}

	argTemplateAllowedTools, err := template.ParseGlob(a.Commands.ArgTemplateAllowedTools)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argTemplateAllowedTools: %w", err)
	}

	runPrompt, err := template.ParseGlob(a.Commands.RunPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse runPrompt: %w", err)
	}

	var serverFiles []string
	filesRaw, err := task.GetMcpServerFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get the mcp server files: %w", err)
	}

	for _, f := range filesRaw {
		tmp := struct {
			File string
		}{
			File: f,
		}

		formatted := bytes.NewBuffer(nil)
		err := argTemplateMcpServer.Execute(formatted, tmp)
		if err != nil {
			return nil, fmt.Errorf("failed to execute argTemplateMcpServer: %w", err)
		}

		serverFiles = append(serverFiles, formatted.String())
	}

	var allowedTools []string
	for _, s := range task.GetMcpServers() {
		for _, t := range s.GetAllowedToolNames() {
			tmp := struct {
				ServerName string
				ToolName   string
			}{
				ServerName: s.GetName(),
				ToolName:   t,
			}

			formatted := bytes.NewBuffer(nil)
			err := argTemplateAllowedTools.Execute(formatted, tmp)
			if err != nil {
				return nil, fmt.Errorf("failed to execute argTemplateAllowedTools: %w", err)
			}

			allowedTools = append(allowedTools, formatted.String())
		}
	}

	tmp := struct {
		McpServerFileArgs string
		AllowedToolArgs   string
		Prompt            string
	}{
		McpServerFileArgs: strings.Join(serverFiles, " "),
		AllowedToolArgs:   strings.Join(allowedTools, " "),
		Prompt:            task.GetPrompt(),
	}

	formatted := bytes.NewBuffer(nil)
	err = runPrompt.Execute(formatted, tmp)
	if err != nil {
		return nil, fmt.Errorf("failed to execute runPrompt: %w", err)
	}

	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		shell = "/usr/bin/bash"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", formatted.String())

	res, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	return &agentSpecRunnerResult{
		commandOuput: string(res),
	}, nil
}
