package blueprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestApply_DryRunAndExecute(t *testing.T) {
	tempDir := t.TempDir()

	// Scaffold basic dummy project
	yamlConfig := `appName: testapi
database: postgres
port: 8080
`
	if err := os.WriteFile(filepath.Join(tempDir, "fiberforge.yaml"), []byte(yamlConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	routesDir := filepath.Join(tempDir, "routes")
	if err := os.MkdirAll(routesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initialRoutes := `package routes

import "github.com/gofiber/fiber/v2"

func Routes(app *fiber.App) {
	api := app.Group("/api")
}
`
	if err := os.WriteFile(filepath.Join(routesDir, "routes.go"), []byte(initialRoutes), 0o644); err != nil {
		t.Fatal(err)
	}

	// Register a test blueprint
	bp := &Blueprint{
		ID:          "custom-cache",
		Version:     "1.0.0",
		Title:       "Custom Redis Cache",
		Description: "High speed caching",
		Category:    "Caching",
		Target: TargetSpec{
			Language:  "go",
			Framework: "fiber",
		},
		Env: []EnvVar{
			{Key: "REDIS_CACHE_URL", Required: true, Default: "localhost:6379", Description: "Redis Host"},
		},
		Files: []FileMapping{
			{Source: "pkg/cache/cache.go", Target: "pkg/cache/cache.go", Content: "package cache\n\nfunc Ping() string { return \"pong\" }"},
		},
		Models: []schema.Model{
			{
				Name:     "CacheKey",
				Endpoint: "cache-keys",
				Fields: []schema.Field{
					{Name: "Key", Type: schema.TypeString, Required: true},
					{Name: "Val", Type: schema.TypeText},
				},
			},
		},
		Hooks: Hooks{
			Routes: []RouteHook{
				{Path: "/cache/flush", Method: "POST", Handler: "cache.Flush"},
			},
		},
		AgentPrompt: "Cache helper installed at pkg/cache.",
	}
	Register(bp)

	// 1. DryRun
	dryRes, err := Apply(tempDir, "custom-cache", true, true)
	if err != nil {
		t.Fatalf("dry run Apply failed: %v", err)
	}
	if len(dryRes.AddedModels) != 1 || dryRes.AddedModels[0] != "CacheKey" {
		t.Errorf("expected added model CacheKey in dry run, got %+v", dryRes.AddedModels)
	}
	if len(dryRes.GeneratedFiles) == 0 {
		t.Errorf("expected generated files in dry run")
	}

	// Ensure files were NOT written during dry run
	if _, err := os.Stat(filepath.Join(tempDir, "pkg", "cache", "cache.go")); err == nil {
		t.Errorf("pkg/cache/cache.go should NOT exist after dry-run")
	}

	// 2. Real Apply
	execRes, err := Apply(tempDir, "custom-cache", false, true)
	if err != nil {
		t.Fatalf("real Apply failed: %v", err)
	}
	if len(execRes.AddedModels) != 1 {
		t.Errorf("expected 1 added model, got %d", len(execRes.AddedModels))
	}

	// Check file on disk
	content, err := os.ReadFile(filepath.Join(tempDir, "pkg", "cache", "cache.go"))
	if err != nil {
		t.Fatalf("failed reading created file: %v", err)
	}
	if !strings.Contains(string(content), "func Ping()") {
		t.Errorf("expected file content to contain func Ping(), got:\n%s", string(content))
	}

	// Check .env.example
	envContent, err := os.ReadFile(filepath.Join(tempDir, ".env.example"))
	if err != nil {
		t.Fatalf("failed reading .env.example: %v", err)
	}
	if !strings.Contains(string(envContent), "REDIS_CACHE_URL=localhost:6379") {
		t.Errorf("expected .env.example to contain REDIS_CACHE_URL, got:\n%s", string(envContent))
	}

	// Check routes.go AST grafting
	routesContent, _ := os.ReadFile(filepath.Join(routesDir, "routes.go"))
	if !strings.Contains(string(routesContent), `api.Post("/cache/flush", cache.Flush)`) {
		t.Errorf("expected route grafted into routes.go, got:\n%s", string(routesContent))
	}

	// Check manifest fiberforge.yaml recording
	isApplied, record, err := IsBlueprintApplied(tempDir, "custom-cache")
	if err != nil || !isApplied {
		t.Fatalf("expected blueprint recorded in fiberforge.yaml: %v", err)
	}
	if record.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", record.Version)
	}
}
