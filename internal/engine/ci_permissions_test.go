package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestGitHubActions_LeastPrivilegePermissions(t *testing.T) {
	// 1. Verify repository's own CI workflow
	ciPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")
	ciContent, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("failed to read .github/workflows/ci.yml: %v", err)
	}

	ciStr := string(ciContent)
	if !strings.Contains(ciStr, "permissions:\n  contents: read") {
		t.Fatalf(".github/workflows/ci.yml must define 'permissions: contents: read', got:\n%s", ciStr)
	}

	// 2. Verify generated project CI workflow
	cfg := &schema.Config{
		AppName:   "citest",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{CI: true},
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

	genCIPath := filepath.Join(outDir, ".github", "workflows", "ci.yml")
	genCIContent, err := os.ReadFile(genCIPath)
	if err != nil {
		t.Fatalf("failed to read generated ci.yml: %v", err)
	}

	genCIStr := string(genCIContent)
	if !strings.Contains(genCIStr, "permissions:\n  contents: read") {
		t.Fatalf("generated ci.yml must define 'permissions: contents: read', got:\n%s", genCIStr)
	}
}
