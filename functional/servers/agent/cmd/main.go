// Package main provides the mock agent binary entry point.
// The mock agent is configured via a JSON file specified by the
// MOCK_AGENT_CONFIG environment variable or --config flag.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mcpchecker/mcpchecker/functional/servers/agent"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := agent.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
