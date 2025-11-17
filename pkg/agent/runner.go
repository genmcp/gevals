package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/genmcp/gevals/pkg/mcpproxy"
)

type Runner interface {
	RunTask(ctx context.Context, prompt string) (AgentResult, error)
	WithMcpServerInfo(mcpInfo McpServerInfo) Runner
}

type McpServerInfo interface {
	GetMcpServerFiles() ([]string, error)
	GetMcpServers() []mcpproxy.Server
}

type AgentResult interface {
	GetOutput() string
}

type agentSpecRunner struct {
	*AgentSpec
	mcpInfo McpServerInfo
}

type agentSpecRunnerResult struct {
	commandOutput string
}

func (a *agentSpecRunnerResult) GetOutput() string {
	return a.commandOutput
}

func NewRunnerForSpec(spec *AgentSpec) (Runner, error) {
	if spec == nil {
		return nil, fmt.Errorf("cannot create a Runner for a nil AgentSpec")
	}

	return &agentSpecRunner{
		AgentSpec: spec,
	}, nil
}

func (a *agentSpecRunner) RunTask(ctx context.Context, prompt string) (AgentResult, error) {
	debugDir := ""
	if os.Getenv("GEVALS_DEBUG") != "" {
		if dir, err := os.MkdirTemp("", "gevals-debug-"); err == nil {
			debugDir = dir
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to create debug directory: %v\n", err)
		}
	}

	// Create an empty temporary directory for agent execution to isolate it from source code
	tempDir, err := os.MkdirTemp("", "gevals-agent-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory for agent execution: %w", err)
	}
	executionSucceeded := false
	defer func() {
		// Clean up temp directory unless execution failed OR GEVALS_DEBUG is set
		// In that case, preserve it for debugging
		shouldPreserve := !executionSucceeded || os.Getenv("GEVALS_DEBUG") != ""
		if !shouldPreserve {
			_ = os.RemoveAll(tempDir)
		} else {
			var reason string
			if !executionSucceeded && os.Getenv("GEVALS_DEBUG") != "" {
				reason = "execution failed and GEVALS_DEBUG is set"
			} else if !executionSucceeded {
				reason = "execution failed"
			} else {
				reason = "GEVALS_DEBUG is set"
			}
			fmt.Fprintf(os.Stderr, "Preserving temporary directory %s because %s\n", tempDir, reason)
		}
	}()

	argTemplateMcpServer, err := template.New("argTemplateMcpServer").Parse(a.Commands.ArgTemplateMcpServer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argTemplateMcpServer: %w", err)
	}

	argTemplateAllowedTools, err := template.New("argTemplateAllowedTools").Parse(a.Commands.ArgTemplateAllowedTools)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argTemplateAllowedTools: %w", err)
	}

	runPrompt, err := template.New("runPrompt").Parse(a.Commands.RunPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse runPrompt: %w", err)
	}

	var serverFiles []string
	filesRaw, err := a.mcpInfo.GetMcpServerFiles()
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
	for _, s := range a.mcpInfo.GetMcpServers() {
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

	// Default to space separator if not specified
	allowedToolsSeparator := " "
	if a.Commands.AllowedToolsJoinSeparator != nil {
		allowedToolsSeparator = *a.Commands.AllowedToolsJoinSeparator
	}

	tmp := struct {
		McpServerFileArgs string
		AllowedToolArgs   string
		Prompt            string
	}{
		McpServerFileArgs: strings.Join(serverFiles, " "),
		AllowedToolArgs:   strings.Join(allowedTools, allowedToolsSeparator),
		Prompt:            prompt,
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
	cmd.Dir = tempDir
	envVars := os.Environ()
	if debugDir != "" {
		envVars = append(envVars, fmt.Sprintf("GEVALS_DEBUG_DIR=%s", debugDir))
		envVars = append(envVars, "GEVALS_DEBUG=1")
	}
	cmd.Env = envVars

	res, err := cmd.CombinedOutput()
	if err != nil {
		debugSuffix := ""
		if debugDir != "" {
			debugSuffix = fmt.Sprintf("\n\ndebug artifacts preserved at: %s", debugDir)
		}
		// executionSucceeded remains false, so tempDir will be preserved
		tempDirSuffix := fmt.Sprintf("\n\ntemporary directory preserved at: %s", tempDir)
		return nil, fmt.Errorf("failed to run command: %s -c %q: %w.\n\noutput: %s%s%s", shell, formatted.String(), err, res, debugSuffix, tempDirSuffix)
	}

	executionSucceeded = true

	if debugDir != "" {
		_ = os.RemoveAll(debugDir)
	}

	output := string(res)
	// If GEVALS_DEBUG is set, append temp directory info to output so it appears in JSON log
	if os.Getenv("GEVALS_DEBUG") != "" {
		output += fmt.Sprintf("\n\ntemporary directory preserved at: %s", tempDir)
	}

	return &agentSpecRunnerResult{
		commandOutput: output,
	}, nil
}

func (a *agentSpecRunner) WithMcpServerInfo(mcpInfo McpServerInfo) Runner {
	return &agentSpecRunner{
		AgentSpec: a.AgentSpec,
		mcpInfo:   mcpInfo,
	}
}
