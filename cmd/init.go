package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactively create a new FiberForge schema and scaffold a project",
	Long: `init launches an interactive terminal wizard that prompts you for app configuration,
feature flags, and model definitions, then saves a fiberforge.yaml schema and option to scaffold immediately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			appName      = "my-app"
			database     = "postgres"
			portStr      = "8080"
			selectedFeat []string
			scaffoldNow  bool = true
		)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("App Name").
					Value(&appName),
				huh.NewSelect[string]().
					Title("Database").
					Options(
						huh.NewOption("PostgreSQL", "postgres"),
						huh.NewOption("MySQL", "mysql"),
						huh.NewOption("MongoDB", "mongodb"),
					).
					Value(&database),
				huh.NewInput().
					Title("Port").
					Value(&portStr),
			),
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Features").
					Options(
						huh.NewOption("JWT Authentication", "auth"),
						huh.NewOption("Docker & Docker Compose", "docker"),
						huh.NewOption("SQL Migrations", "migrations"),
						huh.NewOption("Swagger Docs (OpenAPI 3.0)", "swagger"),
						huh.NewOption("Rate Limiting Middleware", "rateLimit"),
						huh.NewOption("CORS Middleware", "cors"),
						huh.NewOption("Request Logger", "logging"),
						huh.NewOption("Smoke Unit Testing", "testing"),
						huh.NewOption("GitHub Actions CI", "ci"),
					).
					Value(&selectedFeat),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		port, _ := strconv.Atoi(portStr)
		if port == 0 {
			port = 8080
		}

		featMap := map[string]bool{}
		for _, f := range selectedFeat {
			featMap[f] = true
		}

		cfg := &schema.Config{
			AppName:   appName,
			Framework: "fiber",
			Database:  database,
			Port:      port,
			Features: schema.Features{
				Auth:       featMap["auth"],
				Docker:     featMap["docker"],
				Migrations: featMap["migrations"],
				Swagger:    featMap["swagger"],
				RateLimit:  featMap["rateLimit"],
				CORS:       featMap["cors"],
				Logging:    featMap["logging"],
				Testing:    featMap["testing"],
				CI:         featMap["ci"],
			},
		}

		// Model builder loop
		for {
			var addAnother bool
			confirmForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Add a model to your schema?").
						Value(&addAnother),
				),
			)
			if err := confirmForm.Run(); err != nil || !addAnother {
				break
			}

			var (
				modelName string
				endpoint  string
			)
			modelForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Model Name (e.g. Post)").
						Value(&modelName),
					huh.NewInput().
						Title("Endpoint prefix (e.g. posts)").
						Value(&endpoint),
				),
			)
			if err := modelForm.Run(); err != nil {
				return err
			}

			if endpoint == "" {
				endpoint = strings.ToLower(schema.Plural(modelName))
			}

			model := schema.Model{
				Name:     schema.Pascal(modelName),
				Endpoint: endpoint,
			}

			// Field builder loop
			for {
				var addField bool
				fieldConfirmForm := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title(fmt.Sprintf("Add a field to model %s?", model.Name)).
							Value(&addField),
					),
				)
				if err := fieldConfirmForm.Run(); err != nil || !addField {
					break
				}

				var (
					fieldName string
					fieldType = "string"
					fieldOpts []string
				)

				fieldForm := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Field Name").
							Value(&fieldName),
						huh.NewSelect[string]().
							Title("Field Type").
							Options(
								huh.NewOption("string", "string"),
								huh.NewOption("text", "text"),
								huh.NewOption("int", "int"),
								huh.NewOption("int64", "int64"),
								huh.NewOption("float", "float"),
								huh.NewOption("bool", "bool"),
								huh.NewOption("time", "time"),
								huh.NewOption("uuid", "uuid"),
								huh.NewOption("json", "json"),
								huh.NewOption("password", "password"),
							).
							Value(&fieldType),
						huh.NewMultiSelect[string]().
							Title("Field Options").
							Options(
								huh.NewOption("Required (NOT NULL)", "required"),
								huh.NewOption("Unique", "unique"),
								huh.NewOption("Omit empty in JSON", "omitempty"),
								huh.NewOption("Sensitive (never serialized)", "sensitive"),
							).
							Value(&fieldOpts),
					),
				)
				if err := fieldForm.Run(); err != nil {
					return err
				}

				optsMap := map[string]bool{}
				for _, opt := range fieldOpts {
					optsMap[opt] = true
				}

				f := schema.Field{
					Name:      schema.Pascal(fieldName),
					Type:      schema.FieldType(fieldType),
					Required:  optsMap["required"],
					Unique:    optsMap["unique"],
					OmitEmpty: optsMap["omitempty"],
					Sensitive: optsMap["sensitive"],
				}
				model.Fields = append(model.Fields, f)
			}

			cfg.Models = append(cfg.Models, model)
		}

		if len(cfg.Models) == 0 {
			cfg.Models = append(cfg.Models, schema.Model{
				Name:     "Post",
				Endpoint: "posts",
				Fields: []schema.Field{
					{Name: "Title", Type: schema.TypeString, Required: true},
					{Name: "Content", Type: schema.TypeText},
				},
			})
		}

		outData, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to encode schema: %w", err)
		}

		schemaFile := "fiberforge.yaml"
		if err := os.WriteFile(schemaFile, outData, 0o644); err != nil {
			return fmt.Errorf("failed to save %s: %w", schemaFile, err)
		}
		fmt.Printf("✓ Saved schema to %s\n", schemaFile)

		scaffoldForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Scaffold project now?").
					Value(&scaffoldNow),
			),
		)
		if err := scaffoldForm.Run(); err == nil && scaffoldNow {
			eng := engine.New(cfg)
			dir, err := eng.Generate()
			if err != nil {
				return fmt.Errorf("scaffold failed: %w", err)
			}
			fmt.Printf("✓ Project scaffolded into %s\n", dir)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
