package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestCORS_GeneratedMain_NoWildcard(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "corstest",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{CORS: true},
		Models: []schema.Model{
			{
				Name:     "item",
				Endpoint: "items",
				Fields: []schema.Field{
					{Name: "name", Type: schema.TypeString},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	mainPath := filepath.Join(outDir, "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	mainStr := string(content)

	// Invariant: Must not hardcode wildcard AllowOrigins: "*"
	if strings.Contains(mainStr, `AllowOrigins: "*"`) || strings.Contains(mainStr, `AllowOrigins:     "*"`) {
		t.Fatalf("vulnerability MED-02 detected: found wildcard AllowOrigins: \"*\" in main.go")
	}

	// Invariant: Must use configurable CORS_ALLOWED_ORIGINS
	if !strings.Contains(mainStr, `os.Getenv("CORS_ALLOWED_ORIGINS")`) {
		t.Fatalf("main.go must configure CORS origins from environment variable")
	}

	// Invariant: Must enable AllowCredentials safely with non-wildcard origins
	if !strings.Contains(mainStr, `AllowCredentials: true`) {
		t.Fatalf("main.go should configure AllowCredentials")
	}

	// Check .env.example
	envPath := filepath.Join(outDir, ".env.example")
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env.example: %v", err)
	}
	if !strings.Contains(string(envContent), "CORS_ALLOWED_ORIGINS=") {
		t.Fatalf(".env.example must document CORS_ALLOWED_ORIGINS configuration")
	}
}

func TestCORS_OriginMatchingLogic(t *testing.T) {
	isOriginAllowed := func(allowedOrigins string, requestOrigin string) bool {
		origins := strings.Split(allowedOrigins, ",")
		for _, o := range origins {
			if strings.TrimSpace(o) == requestOrigin {
				return true
			}
		}
		return false
	}

	originsConfig := "http://localhost:3000, http://127.0.0.1:8080, https://app.example.com"

	t.Run("Allowed single origin", func(t *testing.T) {
		if !isOriginAllowed(originsConfig, "http://localhost:3000") {
			t.Fatalf("expected http://localhost:3000 to be allowed")
		}
	})

	t.Run("Allowed second origin", func(t *testing.T) {
		if !isOriginAllowed(originsConfig, "https://app.example.com") {
			t.Fatalf("expected https://app.example.com to be allowed")
		}
	})

	t.Run("Disallowed origin", func(t *testing.T) {
		if isOriginAllowed(originsConfig, "http://evil.com") {
			t.Fatalf("http://evil.com must not be allowed")
		}
	})

	t.Run("Disallowed subdomain origin", func(t *testing.T) {
		if isOriginAllowed(originsConfig, "https://sub.app.example.com") {
			t.Fatalf("subdomains not explicitly configured must not be allowed")
		}
	})
}
