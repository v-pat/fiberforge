package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestSecrets_EnvExamplePlaceholders(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "secretstest",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{Auth: true},
		Models: []schema.Model{
			{
				Name:     "user",
				Endpoint: "users",
				Fields: []schema.Field{
					{Name: "email", Type: schema.TypeString, Required: true},
					{Name: "password", Type: schema.TypePassword, Required: true},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	envPath := filepath.Join(outDir, ".env.example")
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env.example: %v", err)
	}

	envStr := string(envContent)

	// Invariant: No real or default secrets in .env.example
	if strings.Contains(envStr, "DB_PASSWORD=password") {
		t.Fatalf("vulnerability MED-04: .env.example must use placeholder for DB_PASSWORD, got:\n%s", envStr)
	}
	if strings.Contains(envStr, "JWT_SECRET=change-me-in-production") {
		t.Fatalf("vulnerability MED-04: .env.example must use placeholder for JWT_SECRET, got:\n%s", envStr)
	}
}

func TestSecrets_ProductionConfigFailsOnInsecureSecrets(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "prodconfigtest",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{Auth: true},
		Models: []schema.Model{
			{
				Name:     "user",
				Endpoint: "users",
				Fields: []schema.Field{
					{Name: "email", Type: schema.TypeString, Required: true},
					{Name: "password", Type: schema.TypePassword, Required: true},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	cfgPath := filepath.Join(outDir, "config", "config.go")
	cfgContent, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config.go: %v", err)
	}

	cfgStr := string(cfgContent)

	// Invariant: Checks for production environment
	if !strings.Contains(cfgStr, `os.Getenv("ENV")`) || !strings.Contains(cfgStr, `"production"`) {
		t.Fatalf("config.Load must detect production mode, got:\n%s", cfgStr)
	}

	// Invariant: Rejects default passwords in production
	if !strings.Contains(cfgStr, `val == "password"`) {
		t.Fatalf("config.Load must reject default password in production")
	}

	// Invariant: Rejects default JWT secret in production
	if !strings.Contains(cfgStr, `val == "change-me-in-production"`) {
		t.Fatalf("config.Load must reject default JWT secret in production")
	}
}
