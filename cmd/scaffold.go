package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/v-pat/fiberforge/examples"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
)

var (
	outputDir    string
	dryRun       bool
	templateName string
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <config.yaml|config.json>",
	Short: "Generate a complete Go Fiber project from a schema",
	Long: `scaffold reads a YAML or JSON schema describing your project and
generates a complete, production-ready Go Fiber application on disk.
No AI required — the schema drives deterministic code generation.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg *schema.Config
		var err error

		if templateName != "" {
			content, err := examples.Get(templateName)
			if err != nil {
				return fmt.Errorf("failed to load template: %w", err)
			}
			cfg, err = schema.Parse(content)
			if err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
			}
		} else if len(args) == 1 {
			cfg, err = schema.Load(args[0])
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("must provide either a schema file or a --template flag")
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
	scaffoldCmd.Flags().StringVarP(&templateName, "template", "t", "", "scaffold from a pre-built template (e.g. blog, ecommerce, saas, social)")

	rootCmd.AddCommand(scaffoldCmd)
}
