package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/acp-go-sdk"
	"github.com/mcpchecker/mcpchecker/pkg/acpclient"
	"github.com/mcpchecker/mcpchecker/pkg/mcpproxy"
)

type acpRunner struct {
	name       string
	cfg        *acpclient.AcpConfig
	mcpServers mcpproxy.ServerManager
}

var _ Runner = &acpRunner{}

func NewAcpRunner(cfg *acpclient.AcpConfig, name string) Runner {
	return &acpRunner{
		name: name,
		cfg:  cfg,
	}
}

func (r *acpRunner) RunTask(ctx context.Context, prompt string) (AgentResult, error) {
	client := acpclient.NewClient(ctx, r.cfg)
	defer client.Close(ctx)

	err := client.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start acp client: %w", err)
	}

	result, err := client.Run(ctx, prompt, r.mcpServers)
	if err != nil {
		return nil, fmt.Errorf("failed to run acp agent: %w", err)
	}

	return &acpRunnerResult{
		updates: result,
	}, nil
}

func (r *acpRunner) WithMcpServerInfo(mcpServers mcpproxy.ServerManager) Runner {
	return &acpRunner{
		name:       r.name,
		cfg:        r.cfg,
		mcpServers: mcpServers,
	}
}

func (r *acpRunner) AgentName() string {
	return r.name
}

type acpRunnerResult struct {
	updates []acp.SessionUpdate
}

var _ AgentResult = &acpRunnerResult{}

func (res *acpRunnerResult) GetOutput() string {
	if len(res.updates) == 0 {
		return "got no output from acp agent"
	}

	out, err := json.Marshal(res.updates)
	if err != nil {
		text := res.updates[len(res.updates)-1].AgentMessageChunk.Content.Text
		if text != nil {
			return text.Text
		}

		return "unable to get agent output from last acp update"
	}

	return string(out)
}
