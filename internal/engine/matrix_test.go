package engine

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

// TestMatrixCompilation generates projects for different databases and feature sets,
// then runs "go mod tidy" and "go build ./..." to ensure the templates produce valid Go code.
func TestMatrixCompilation(t *testing.T) {
	if testing.Short() || os.Getenv("CI") == "" {
		t.Skip("skipping compilation matrix in short mode or local environment")
	}

	testCases := []struct {
		name     string
		database string
		features schema.Features
	}{
		{
			name:     "Postgres_Full",
			database: "postgres",
			features: schema.Features{Auth: true, Docker: true, Testing: true, Swagger: true, Migrations: true, RateLimit: true, CORS: true, Logging: true, CI: true},
		},
		{
			name:     "MySQL_Minimal",
			database: "mysql",
			features: schema.Features{},
		},
		{
			name:     "Mongo_Auth_Test",
			database: "mongodb",
			features: schema.Features{Auth: true, Testing: true, Logging: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &schema.Config{
				AppName:  "matrixtest",
				Database: tc.database,
				Features: tc.features,
				Models: []schema.Model{
					{
						Name:          "Post",
						Endpoint:      "posts",
						AuthProtected: tc.features.Auth,
						Fields: []schema.Field{
							{Name: "Title", Type: schema.TypeString, Required: true},
							{Name: "Content", Type: schema.TypeText},
						},
					},
				},
			}

			// Always add User model if auth is enabled
			if tc.features.Auth {
				cfg.Models = append(cfg.Models, schema.Model{
					Name:     "User",
					Endpoint: "users",
					Fields: []schema.Field{
						{Name: "Email", Type: schema.TypeString, Required: true, Unique: true},
						{Name: "Password", Type: schema.TypePassword, Required: true},
					},
				})
				cfg.Models[0].Relationships = []schema.Relationship{
					{Type: schema.BelongsTo, Model: "user"},
				}
			}

			outDir := t.TempDir()
			cfg.OutputDir = outDir

			if _, err := New(cfg).Generate(); err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Run go mod tidy
			tidyCmd := exec.Command("go", "mod", "tidy")
			tidyCmd.Dir = outDir
			var tidyOut bytes.Buffer
			tidyCmd.Stdout = &tidyOut
			tidyCmd.Stderr = &tidyOut
			if err := tidyCmd.Run(); err != nil {
				t.Fatalf("go mod tidy failed:\n%s\nError: %v", tidyOut.String(), err)
			}

			// Run go build ./...
			buildCmd := exec.Command("go", "build", "./...")
			buildCmd.Dir = outDir
			var buildOut bytes.Buffer
			buildCmd.Stdout = &buildOut
			buildCmd.Stderr = &buildOut
			if err := buildCmd.Run(); err != nil {
				t.Fatalf("go build failed:\n%s\nError: %v", buildOut.String(), err)
			}

			// Run go test ./... if testing is enabled
			if tc.features.Testing {
				testCmd := exec.Command("go", "test", "./...")
				testCmd.Dir = outDir
				var testOut bytes.Buffer
				testCmd.Stdout = &testOut
				testCmd.Stderr = &testOut
				if err := testCmd.Run(); err != nil {
					t.Fatalf("go test failed:\n%s\nError: %v", testOut.String(), err)
				}
			}
		})
	}
}
