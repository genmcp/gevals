package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root geval command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "geval",
		Short: "MCP evaluation framework",
		Long: `geval is a framework for evaluating MCP agents against tasks.
It runs agents through defined tasks and validates their behavior using assertions.`,
	}

	// Add subcommands
	rootCmd.AddCommand(NewEvalCmd())
	rootCmd.AddCommand(NewViewCmd())

	return rootCmd
}

// Execute runs the root command
func Execute() error {
	return NewRootCmd().Execute()
}
