package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/introspect"
	"github.com/v-pat/fiberforge/internal/schema"
)

var (
	introspectDSN      string
	introspectSQLFile  string
	introspectDriver   string
	introspectAppName  string
	introspectOutput   string
	introspectScaffold bool
)

var introspectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Reverse-engineer a schema or project from an existing database or SQL file",
	Long: `Introspect connects to an existing PostgreSQL or MySQL database (or parses an existing
SQL DDL file) and automatically reverse-engineers a fiberforge.yaml schema with models, fields,
types, and foreign-key relationships.

You can optionally pass --scaffold to immediately generate the Go Fiber API from the introspected schema.`,
	Example: `  # Introspect from PostgreSQL connection string
  fiberforge introspect --dsn "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

  # Introspect from SQL DDL dump file
  fiberforge introspect --sql ./schema.sql --app-name myapi

  # Introspect and immediately scaffold Go backend
  fiberforge introspect --dsn "postgres://..." --scaffold`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if introspectDSN == "" && introspectSQLFile == "" {
			return fmt.Errorf("must specify either --dsn or --sql file for introspection")
		}

		appName := introspectAppName
		if appName == "" {
			appName = "introspected-api"
		}

		var cfg *schema.Config
		var err error

		if introspectSQLFile != "" {
			data, err := os.ReadFile(introspectSQLFile)
			if err != nil {
				return fmt.Errorf("failed to read SQL file %s: %w", introspectSQLFile, err)
			}
			driver := introspectDriver
			if driver == "" {
				driver = "postgres"
			}
			cfg, err = introspect.IntrospectSQL(string(data), appName, driver)
			if err != nil {
				return fmt.Errorf("SQL introspection failed: %w", err)
			}
		} else {
			cfg, err = introspect.IntrospectDatabase(context.Background(), introspect.Options{
				AppName:  appName,
				Database: introspectDriver,
				DSN:      introspectDSN,
			})
			if err != nil {
				return fmt.Errorf("database introspection failed: %w", err)
			}
		}

		yamlStr, err := introspect.ToYAML(cfg)
		if err != nil {
			return fmt.Errorf("failed to format schema: %w", err)
		}

		if introspectOutput != "" {
			root := os.Getenv("FIBERFORGE_WORKSPACE_ROOT")
			if root == "" {
				root, _ = os.Getwd()
			}
			safeOut, err := schema.ResolveSafePath(introspectOutput, root)
			if err != nil {
				return fmt.Errorf("output path invalid: %w", err)
			}
			if err := os.WriteFile(safeOut, []byte(yamlStr), 0644); err != nil {
				return fmt.Errorf("failed to write output schema: %w", err)
			}
			fmt.Printf("✓ Introspected schema written to %s (%d models detected)\n", safeOut, len(cfg.Models))
		} else if !introspectScaffold {
			fmt.Println(yamlStr)
		}

		if introspectScaffold {
			fmt.Printf("⚡ Scaffolding Go Fiber project for %s (%d models)...\n", cfg.AppName, len(cfg.Models))
			root := os.Getenv("FIBERFORGE_WORKSPACE_ROOT")
			if root == "" {
				root, _ = os.Getwd()
			}
			targetDir := "./" + cfg.AppName
			safeDir, err := schema.ResolveSafePath(targetDir, root)
			if err != nil {
				return fmt.Errorf("scaffold target invalid: %w", err)
			}
			cfg.OutputDir = safeDir
			cfg.WorkspaceRoot = root

			eng := engine.New(cfg)
			outDir, err := eng.Generate()
			if err != nil {
				return fmt.Errorf("scaffolding failed: %w", err)
			}
			fmt.Printf("✓ Project scaffolded successfully in %s\n", filepath.Clean(outDir))
		}

		return nil
	},
}

func init() {
	introspectCmd.Flags().StringVar(&introspectDSN, "dsn", "", "Database connection URL (PostgreSQL or MySQL)")
	introspectCmd.Flags().StringVar(&introspectSQLFile, "sql", "", "Path to SQL DDL file containing CREATE TABLE statements")
	introspectCmd.Flags().StringVar(&introspectDriver, "driver", "", "Database driver ('postgres' or 'mysql')")
	introspectCmd.Flags().StringVar(&introspectAppName, "app-name", "", "Name of the scaffolded application")
	introspectCmd.Flags().StringVarP(&introspectOutput, "output", "o", "", "Path to write the generated YAML schema")
	introspectCmd.Flags().BoolVar(&introspectScaffold, "scaffold", false, "Immediately scaffold the Go Fiber backend from the introspected schema")

	rootCmd.AddCommand(introspectCmd)
}
