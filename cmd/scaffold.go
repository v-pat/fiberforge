package cmd

import (
	"encoding/json"
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
	jsonOutput   bool
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
			if jsonOutput {
				return printJSON(map[string]any{
					"dryRun":   true,
					"appName":  cfg.AppName,
					"database": cfg.Database,
					"files":    eng.Files(),
				})
			}
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
		if jsonOutput {
			return printJSON(map[string]any{
				"success":   true,
				"outputDir": dir,
				"appName":   cfg.AppName,
				"database":  cfg.Database,
				"files":     eng.Files(),
			})
		}
		fmt.Fprintf(os.Stderr, "✓ project generated in %s\n", dir)
		return nil
	},
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	scaffoldCmd.Flags().StringVar(&outputDir, "output-dir", "", "override the output directory")
	scaffoldCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview generated files without writing to disk")
	scaffoldCmd.Flags().StringVarP(&templateName, "template", "t", "", "scaffold from a pre-built template (e.g. blog, ecommerce, saas, social)")
	scaffoldCmd.Flags().BoolVar(&jsonOutput, "json", false, "output results as JSON (for agent/script consumption)")

	rootCmd.AddCommand(scaffoldCmd)
}
