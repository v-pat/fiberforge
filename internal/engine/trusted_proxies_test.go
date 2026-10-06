package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestTrustedProxies_GenerationAndConfig(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "proxytest",
		Framework: "fiber",
		Database:  "postgres",
		Features: schema.Features{
			Auth:      true,
			RateLimit: true,
		},
		Models: []schema.Model{
			{
				Name:     "item",
				Endpoint: "items",
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
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
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	mainStr := string(mainContent)

	// Invariant: main.go checks TRUSTED_PROXIES and enables Fiber trusted proxy mechanism safely
	if !strings.Contains(mainStr, `os.Getenv("TRUSTED_PROXIES")`) {
		t.Fatalf("main.go must check TRUSTED_PROXIES environment variable, got:\n%s", mainStr)
	}
	if !strings.Contains(mainStr, "fiberCfg.EnableTrustedProxyCheck = true") {
		t.Fatalf("main.go must set EnableTrustedProxyCheck when TRUSTED_PROXIES is configured")
	}
	if !strings.Contains(mainStr, "fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor") {
		t.Fatalf("main.go must set ProxyHeader to HeaderXForwardedFor when TRUSTED_PROXIES is configured")
	}
	if !strings.Contains(mainStr, "fiberCfg.TrustedProxies = proxies") {
		t.Fatalf("main.go must set TrustedProxies to parsed proxy list")
	}

	envPath := filepath.Join(outDir, ".env.example")
	envContent, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env.example: %v", err)
	}

	envStr := string(envContent)
	if !strings.Contains(envStr, "TRUSTED_PROXIES=") {
		t.Fatalf(".env.example must document TRUSTED_PROXIES")
	}
	if !strings.Contains(envStr, "NEVER set to unrestricted ranges") {
		t.Fatalf(".env.example must warn against setting unrestricted proxy ranges")
	}
}
