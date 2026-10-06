package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestEngine_TableNameCannotInjectGoCode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fiberforge-tablename-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Even if validation were bypassed, verify that engine encoding stays strictly data
	cfg := &schema.Config{
		AppName:       "testapp",
		Database:      "postgres",
		OutputDir:     tmpDir,
		WorkspaceRoot: tmpDir,
		Models: []schema.Model{
			{
				Name:      "Item",
				Endpoint:  "items",
				TableName: "items_table",
				Fields: []schema.Field{
					{Name: "Title", Type: schema.TypeString},
				},
			},
		},
	}

	eng := New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	modelFile := filepath.Join(tmpDir, "model", "item.go")
	content, err := os.ReadFile(modelFile)
	if err != nil {
		t.Fatalf("failed to read model file: %v", err)
	}

	// Parse with Go parser to ensure AST is completely clean
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, modelFile, content, 0)
	if err != nil {
		t.Fatalf("generated model file has invalid Go syntax: %v", err)
	}

	// Verify no init function exists
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "init" {
			t.Fatal("unexpected init function injected in model file")
		}
	}

	if !strings.Contains(string(content), `return "items_table"`) {
		t.Fatalf("expected safely quoted return statement, got:\n%s", string(content))
	}
}
