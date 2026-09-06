package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
)

func TestASTRouteInjection(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &schema.Config{
		AppName:  "testapp",
		Database: "postgres",
		Models: []schema.Model{
			{Name: "User", Endpoint: "users"},
		},
	}

	eng := engine.New(cfg)
	cfg.OutputDir = tmpDir
	eng = engine.New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("failed initial generate: %v", err)
	}

	routesFile := filepath.Join(tmpDir, "routes", "routes.go")
	newModel := schema.Model{
		Name:     "Comment",
		Endpoint: "comments",
	}

	err := engine.RegisterRouteInAST(routesFile, newModel, false)
	if err != nil {
		t.Fatalf("RegisterRouteInAST failed: %v", err)
	}

	content, err := os.ReadFile(routesFile)
	if err != nil {
		t.Fatalf("failed to read routes.go: %v", err)
	}

	src := string(content)
	if !strings.Contains(src, "commentGroup") || !strings.Contains(src, `api.Group`) {
		t.Errorf("expected commentGroup registration in routes.go, got:\n%s", src)
	}
	if !strings.Contains(src, `CreateComment`) {
		t.Errorf("expected CreateComment handler in routes.go, got:\n%s", src)
	}

	// Test Idempotency: calling RegisterRouteInAST second time should not fail or duplicate
	err = engine.RegisterRouteInAST(routesFile, newModel, false)
	if err != nil {
		t.Fatalf("RegisterRouteInAST second call failed: %v", err)
	}

	content2, _ := os.ReadFile(routesFile)
	count := strings.Count(string(content2), "commentGroup :=")
	if count != 1 {
		t.Errorf("expected commentGroup := declared exactly once, got %d times:\n%s", count, string(content2))
	}
}
