package cmd

import (
	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags. Defaults to "dev" for local builds.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "fiberforge",
	Short:         "FiberForge — scaffold production-ready Go Fiber backends",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	rootCmd.Version = Version
	return rootCmd.Execute()
}
