package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldCmd(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "fiberforge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test case 1: Invalid template name
	rootCmd.SetArgs([]string{"scaffold", "--template", "doesnotexist"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	if err := rootCmd.Execute(); err == nil {
		t.Errorf("Expected error for invalid template, got none")
	}

	// Test case 2: Valid template but no dry-run (should generate files)
	outDir := filepath.Join(tempDir, "blog-test")
	rootCmd.SetArgs([]string{"scaffold", "--template", "blog", "--output-dir", outDir})
	out.Reset()
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error for valid template, got: %v", err)
	}

	// Verify some files were created
	if _, err := os.Stat(filepath.Join(outDir, "main.go")); os.IsNotExist(err) {
		t.Errorf("Expected main.go to be generated")
	}

	// Test case 3: Missing arguments
	templateName = ""
	rootCmd.SetArgs([]string{"scaffold"})
	out.Reset()
	if err := rootCmd.Execute(); err == nil {
		t.Errorf("Expected error when no args or flags are provided")
	}
}
