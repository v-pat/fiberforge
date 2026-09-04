package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
)

var (
	outputDir string
	dryRun    bool
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <config.yaml|config.json>",
	Short: "Generate a complete Go Fiber project from a schema",
	Long: `scaffold reads a YAML or JSON schema describing your project and
generates a complete, production-ready Go Fiber application on disk.
No AI required — the schema drives deterministic code generation.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := schema.Load(args[0])
		if err != nil {
			return err
		}
		if outputDir != "" {
			cfg.OutputDir = outputDir
		}
		eng := engine.New(cfg)
		if dryRun {
			fmt.Printf("Dry run for %s (Output: %s):\n", cfg.AppName, eng.Dir())
			for _, f := range eng.Files() {
				fmt.Printf("  - %s\n", f)
			}
			return nil
		}
		dir, err := eng.Generate()
		if err != nil {
			return fmt.Errorf("scaffold failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ project generated in %s\n", dir)
		return nil
	},
}

func init() {
	scaffoldCmd.Flags().StringVar(&outputDir, "output-dir", "", "override the output directory")
	scaffoldCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview generated files without writing to disk")
	rootCmd.AddCommand(scaffoldCmd)
}
