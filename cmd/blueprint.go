package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/v-pat/fiberforge/internal/blueprint"
)

var (
	bpTargetDir   string
	bpCategory    string
	bpInstalled   bool
	bpDryRun      bool
	bpSkipVerify  bool
	bpRefresh     bool
)

var blueprintCmd = &cobra.Command{
	Use:     "blueprint",
	Aliases: []string{"blueprints", "recipe", "recipes"},
	Short:   "Discover, inspect, and apply architectural feature blueprints",
	Long: `Blueprints are pre-packaged, production-grade architectural components
(e.g. stripe-billing, ai-inference, rbac, s3-storage, audit-log) that can be
cleanly cast into any FiberForge Go project.`,
}

var blueprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available or installed blueprints",
	RunE: func(cmd *cobra.Command, args []string) error {
		if bpInstalled {
			records, err := blueprint.ListAppliedBlueprints(bpTargetDir)
			if err != nil {
				return fmt.Errorf("listing installed blueprints: %w", err)
			}
			if len(records) == 0 {
				fmt.Printf("No blueprints currently recorded in %s/fiberforge.yaml\n", bpTargetDir)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tVERSION\tAPPLIED AT\tMODELS")
			for _, r := range records {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Version, r.AppliedAt, strings.Join(r.Models, ", "))
			}
			return w.Flush()
		}

		summaries := blueprint.List()
		if bpCategory != "" {
			summaries = blueprint.Search("", bpCategory, "")
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tCATEGORY\tMODELS\tENV REQS")
		for _, s := range summaries {
			envStr := "-"
			if len(s.RequiredEnv) > 0 {
				envStr = strings.Join(s.RequiredEnv, ", ")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", s.ID, s.Title, s.Category, s.ModelCount, envStr)
		}
		return w.Flush()
	},
}

var blueprintSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search blueprints by keyword, tag, or category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		matches := blueprint.Search(query, bpCategory, "")
		if len(matches) == 0 {
			fmt.Printf("No blueprints found matching %q\n", query)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tCATEGORY\tDESCRIPTION")
		for _, m := range matches {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.ID, m.Title, m.Category, m.Description)
		}
		return w.Flush()
	},
}

var blueprintInspectCmd = &cobra.Command{
	Use:   "inspect <blueprint-id>",
	Short: "Show detailed specifications, models, and env variables for a blueprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		bp, err := blueprint.Get(id)
		if err != nil {
			return err
		}

		fmt.Printf("Blueprint: %s (%s)\n", bp.Title, bp.ID)
		fmt.Printf("Version:   %s\n", bp.Version)
		fmt.Printf("Category:  %s\n", bp.Category)
		fmt.Printf("Summary:   %s\n\n", bp.Description)

		if len(bp.Env) > 0 {
			fmt.Println("Environment Variables:")
			for _, e := range bp.Env {
				req := "optional"
				if e.Required {
					req = "required"
				}
				fmt.Printf("  - %s (%s): %s\n", e.Key, req, e.Description)
			}
			fmt.Println()
		}

		if len(bp.Models) > 0 {
			fmt.Printf("Database Models (%d):\n", len(bp.Models))
			for _, m := range bp.Models {
				fmt.Printf("  - %s (endpoint: /%s, fields: %d)\n", m.Name, m.Endpoint, len(m.Fields))
			}
			fmt.Println()
		}

		if len(bp.Files) > 0 {
			fmt.Println("Files Injected:")
			for _, f := range bp.Files {
				fmt.Printf("  - %s\n", f.Target)
			}
			fmt.Println()
		}

		if len(bp.Hooks.Routes) > 0 {
			fmt.Println("Routes Hooked:")
			for _, r := range bp.Hooks.Routes {
				fmt.Printf("  - %s %s -> %s\n", strings.ToUpper(r.Method), r.Path, r.Handler)
			}
			fmt.Println()
		}

		if bp.AgentPrompt != "" {
			fmt.Printf("Usage Instructions:\n%s\n", strings.TrimSpace(bp.AgentPrompt))
		}

		return nil
	},
}

var blueprintApplyCmd = &cobra.Command{
	Use:   "apply <blueprint-id>",
	Short: "Cast a blueprint into an existing FiberForge project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		targetDir := bpTargetDir
		if targetDir == "" {
			targetDir = "."
		}

		res, err := blueprint.Apply(targetDir, id, bpDryRun, bpSkipVerify)
		if err != nil {
			return fmt.Errorf("applying blueprint %q failed: %w", id, err)
		}

		if bpDryRun {
			fmt.Printf("Dry-run preview for applying blueprint %q to %s:\n", id, targetDir)
			if len(res.AddedModels) > 0 {
				fmt.Printf("Models to be generated: %s\n", strings.Join(res.AddedModels, ", "))
			}
			fmt.Println("Files to be created:")
			for _, f := range res.GeneratedFiles {
				fmt.Printf("  - %s\n", f)
			}
			if len(res.InjectedRoutes) > 0 {
				fmt.Println("Routes to be hooked:")
				for _, r := range res.InjectedRoutes {
					fmt.Printf("  - %s\n", r)
				}
			}
			return nil
		}

		fmt.Fprintf(os.Stderr, "✓ blueprint %q successfully forged into %s\n", id, targetDir)
		if len(res.AddedModels) > 0 {
			fmt.Fprintf(os.Stderr, "  Added models: %s\n", strings.Join(res.AddedModels, ", "))
		}
		if len(res.InjectedRoutes) > 0 {
			fmt.Fprintf(os.Stderr, "  Injected routes: %s\n", strings.Join(res.InjectedRoutes, ", "))
		}
		if res.Verification != nil {
			if res.Verification.Success {
				fmt.Fprintln(os.Stderr, "  Verification: ✓ go mod tidy & build checks passed")
			} else {
				fmt.Fprintf(os.Stderr, "  Verification warning: %s\n", res.Verification.Summary)
			}
		}

		return nil
	},
}

func init() {
	blueprintListCmd.Flags().StringVar(&bpTargetDir, "dir", ".", "target project directory")
	blueprintListCmd.Flags().StringVar(&bpCategory, "category", "", "filter by category")
	blueprintListCmd.Flags().BoolVar(&bpInstalled, "installed", false, "list only blueprints recorded in fiberforge.yaml")

	blueprintSearchCmd.Flags().StringVar(&bpCategory, "category", "", "filter by category")

	blueprintApplyCmd.Flags().StringVar(&bpTargetDir, "dir", ".", "target project directory")
	blueprintApplyCmd.Flags().BoolVar(&bpDryRun, "dry-run", false, "preview generated files and route diffs without modifying files")
	blueprintApplyCmd.Flags().BoolVar(&bpSkipVerify, "skip-verify", false, "skip running automated tidy/build verification")

	blueprintCmd.AddCommand(blueprintListCmd)
	blueprintCmd.AddCommand(blueprintSearchCmd)
	blueprintCmd.AddCommand(blueprintInspectCmd)
	blueprintCmd.AddCommand(blueprintApplyCmd)

	rootCmd.AddCommand(blueprintCmd)
}
