package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/modules"
	"github.com/v-pat/fiberforge/internal/schema"
)

var (
	addModelTargetDir string
	addModelEndpoint  string
	addDryRun         bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Incrementally add models or feature modules to an existing project",
	Long: `add allows you to incrementally append entity models or pre-packaged
feature modules (e.g. stripe-billing, ai-inference, rbac, s3-storage) to an existing FiberForge project
without wiping out your custom code.`,
}

var addModelCmd = &cobra.Command{
	Use:   "model <ModelName>",
	Short: "Add a new entity model to an existing project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		targetDir := addModelTargetDir
		if targetDir == "" {
			targetDir = "."
		}

		endpoint := addModelEndpoint
		if endpoint == "" {
			endpoint = strings.ToLower(schema.Plural(name))
		}

		m := schema.Model{
			Name:     name,
			Endpoint: endpoint,
			Fields: []schema.Field{
				{Name: "Title", Type: schema.TypeString, Required: true},
			},
		}

		files, err := engine.AddModelWithOptions(targetDir, m, addDryRun)
		if err != nil {
			return fmt.Errorf("add model failed: %w", err)
		}

		if addDryRun {
			fmt.Printf("Dry-run preview for adding model %q to %s:\n", name, targetDir)
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
			return nil
		}

		fmt.Fprintf(os.Stderr, "✓ model %q incrementally added to %s\n", name, targetDir)
		return nil
	},
}

var addModuleCmd = &cobra.Command{
	Use:   "module <module-name>",
	Short: "Inject a pre-packaged feature module (e.g. stripe-billing, ai-inference, rbac, s3-storage, audit-log)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		moduleName := args[0]
		targetDir := addModelTargetDir
		if targetDir == "" {
			targetDir = "."
		}

		added, files, err := modules.ApplyWithOptions(targetDir, moduleName, addDryRun)
		if err != nil {
			return fmt.Errorf("add module failed: %w", err)
		}

		if addDryRun {
			fmt.Printf("Dry-run preview for applying module %q to %s:\nModels: %s\nFiles to generate:\n", moduleName, targetDir, strings.Join(added, ", "))
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
			return nil
		}

		fmt.Fprintf(os.Stderr, "✓ module %q successfully applied to %s (added models: %s)\n", moduleName, targetDir, strings.Join(added, ", "))
		return nil
	},
}

func init() {
	addModelCmd.Flags().StringVar(&addModelTargetDir, "dir", ".", "target project directory")
	addModelCmd.Flags().StringVar(&addModelEndpoint, "endpoint", "", "custom API endpoint route (defaults to plural model name)")
	addModelCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "preview files to be created and AST route diffs without writing")

	addModuleCmd.Flags().StringVar(&addModelTargetDir, "dir", ".", "target project directory")
	addModuleCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "preview files to be created and AST route diffs without writing")

	addCmd.AddCommand(addModelCmd)
	addCmd.AddCommand(addModuleCmd)
	rootCmd.AddCommand(addCmd)
}
