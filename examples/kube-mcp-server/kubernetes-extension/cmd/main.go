package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mcpchecker/mcpchecker/examples/kube-mcp-server/kubernetes-extension/pkg/extension"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ext := extension.New()

	if err := ext.Run(ctx); err != nil {
		log.Fatalf("extension error: %v", err)
	}
}
