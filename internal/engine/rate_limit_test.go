package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestRateLimit_AuthEndpointProtectionWhenGlobalDisabled(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "authratetest",
		Framework: "fiber",
		Database:  "postgres",
		Features: schema.Features{
			Auth:      true,
			RateLimit: false, // Global rate limiting is explicitly disabled
		},
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

	routesPath := filepath.Join(outDir, "routes", "routes.go")
	content, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("failed to read routes.go: %v", err)
	}

	routesStr := string(content)

	// Invariant: Dedicated auth limiter must exist even when global rate limiting is disabled
	if !strings.Contains(routesStr, "authLimiter") {
		t.Fatalf("vulnerability MED-05: auth endpoints must have dedicated rate limiter in routes.go")
	}
	if !strings.Contains(routesStr, `api.Post("/auth/register", authLimiter, controller.Register)`) {
		t.Fatalf("register endpoint must be protected by authLimiter")
	}
	if !strings.Contains(routesStr, `api.Post("/auth/login", authLimiter, controller.Login)`) {
		t.Fatalf("login endpoint must be protected by authLimiter")
	}

	// Invariant: Global rate limiter in main.go should be absent when features.rateLimit is false
	mainPath := filepath.Join(outDir, "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	if strings.Contains(string(mainContent), "limiter.New") {
		t.Fatalf("main.go should not configure global limiter when features.rateLimit is false")
	}
}

func TestRateLimit_BothGlobalAndAuthProtected(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "bothratetest",
		Framework: "fiber",
		Database:  "postgres",
		Features: schema.Features{
			Auth:      true,
			RateLimit: true, // Both enabled
		},
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

	routesPath := filepath.Join(outDir, "routes", "routes.go")
	content, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("failed to read routes.go: %v", err)
	}
	if !strings.Contains(string(content), "authLimiter") {
		t.Fatalf("routes.go must have authLimiter")
	}

	mainPath := filepath.Join(outDir, "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), "limiter.New") {
		t.Fatalf("main.go must configure global limiter when features.rateLimit is true")
	}
}
