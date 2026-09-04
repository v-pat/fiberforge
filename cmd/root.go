package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "fiberforge",
	Short:         "FiberForge — scaffold production-ready Go Fiber backends",
	Version:       "v1.0.0",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
