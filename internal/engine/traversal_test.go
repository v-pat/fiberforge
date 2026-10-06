package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestEngine_PathTraversalRefused(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "testapp",
		Database:  "postgres",
		OutputDir: "../../outside_test_dir",
		Models: []schema.Model{
			{
				Name:     "Item",
				Endpoint: "items",
				Fields: []schema.Field{
					{Name: "Name", Type: schema.TypeString},
				},
			},
		},
	}

	eng := New(cfg)
	_, err := eng.Generate()
	if err == nil {
		t.Fatal("expected eng.Generate() to fail for traversal OutputDir, but it succeeded")
	}

	// Verify no directory was created
	if _, statErr := os.Stat("../../outside_test_dir"); statErr == nil {
		_ = os.RemoveAll("../../outside_test_dir")
		t.Fatal("directory was unexpectedly created outside workspace root")
	}
}

func TestEngine_WritePathTraversalRefused(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fiberforge-write-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &schema.Config{
		AppName:   "testapp",
		Database:  "postgres",
		OutputDir: tmpDir,
	}

	eng := New(cfg)
	// Attempt to call write with a path traversal relPath
	err = eng.write("../escaped_file.txt", "malicious content")
	if err == nil {
		t.Fatal("expected eng.write to reject relative path traversal, but it succeeded")
	}

	escapedFile := filepath.Join(tmpDir, "..", "escaped_file.txt")
	if _, statErr := os.Stat(escapedFile); statErr == nil {
		_ = os.Remove(escapedFile)
		t.Fatal("escaped file was created on disk")
	}
}
