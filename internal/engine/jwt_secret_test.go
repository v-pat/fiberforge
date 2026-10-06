package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestMongoJWTSecret_NoHardcodedFallback(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "mongotest",
		Framework: "fiber",
		Database:  "mongodb",
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

	eng := New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	jwtPath := filepath.Join(outDir, "auth", "jwt.go")
	content, err := os.ReadFile(jwtPath)
	if err != nil {
		t.Fatalf("failed to read generated auth/jwt.go: %v", err)
	}

	jwtStr := string(content)

	// Invariant: No hardcoded secret fallback
	if strings.Contains(jwtStr, "change-me-in-production") {
		t.Fatalf("vulnerability HIGH-01 detected: found hardcoded JWT secret fallback in %s", jwtPath)
	}

	// Invariant: Must fail safely if JWT_SECRET is unset
	expectedFail := `log.Fatal("JWT_SECRET environment variable is not set")`
	if !strings.Contains(jwtStr, expectedFail) {
		t.Fatalf("expected fail-safe check %q in %s, got:\n%s", expectedFail, jwtPath, jwtStr)
	}
}

func TestSQLJWTSecret_NoHardcodedFallback(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "sqltest",
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

	eng := New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	jwtPath := filepath.Join(outDir, "auth", "jwt.go")
	content, err := os.ReadFile(jwtPath)
	if err != nil {
		t.Fatalf("failed to read generated auth/jwt.go: %v", err)
	}

	jwtStr := string(content)

	if strings.Contains(jwtStr, "change-me-in-production") {
		t.Fatalf("vulnerability HIGH-01 detected: found hardcoded JWT secret fallback in %s", jwtPath)
	}

	expectedFail := `log.Fatal("JWT_SECRET environment variable is not set")`
	if !strings.Contains(jwtStr, expectedFail) {
		t.Fatalf("expected fail-safe check %q in %s, got:\n%s", expectedFail, jwtPath, jwtStr)
	}
}
