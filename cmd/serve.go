package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/v-pat/fiberforge/internal/mcp"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as an MCP server over stdio",
	Long: `serve starts FiberForge as a Model Context Protocol (MCP) server that
agents can connect to. It exposes tools that generate projects from a schema.
The server is provider-agnostic: it never calls an LLM itself — the agent
provides the schema and the engine generates deterministic code.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := mcp.NewServer(os.Stdin, os.Stdout, Version)
		return srv.Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
