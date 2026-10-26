package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/v-pat/fiberforge/internal/schema"
	"github.com/v-pat/fiberforge/internal/verifier"
)

var (
	verifySkipTests bool
	verifyNoRace    bool
	verifyNoTidy    bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify [dir]",
	Short: "Verify a Go project (runs tidy, vet, build, and tests)",
	Long: `Verify validates that a Go Fiber project is 100% compilable and passing tests.
It executes 'go mod tidy', 'go vet ./...', 'go build ./...', and 'go test -race ./...'
and displays structured diagnostics if any errors or test failures occur.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		root := os.Getenv("FIBERFORGE_WORKSPACE_ROOT")
		if root == "" {
			root, _ = os.Getwd()
		}
		safeDir, err := schema.ResolveSafePath(targetDir, root)
		if err != nil {
			return fmt.Errorf("target directory %q is outside workspace: %w", targetDir, err)
		}

		fmt.Printf("🔍 Verifying Go project in %s...\n", safeDir)

		res, err := verifier.Verify(context.Background(), safeDir, verifier.Options{
			SkipTests: verifySkipTests,
			Race:      !verifyNoRace,
			RunTidy:   !verifyNoTidy,
		})
		if err != nil {
			return fmt.Errorf("verification aborted: %w", err)
		}

		fmt.Println(res.FormatMarkdown())

		if !res.Success {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	verifyCmd.Flags().BoolVar(&verifySkipTests, "skip-tests", false, "Skip running go test ./...")
	verifyCmd.Flags().BoolVar(&verifyNoRace, "no-race", false, "Disable race detector in tests")
	verifyCmd.Flags().BoolVar(&verifyNoTidy, "no-tidy", false, "Skip running go mod tidy")
	rootCmd.AddCommand(verifyCmd)
}
