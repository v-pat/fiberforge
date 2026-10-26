package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVerify_NonExistentDir(t *testing.T) {
	ctx := context.Background()
	_, err := Verify(ctx, "/non/existent/path/for/sure", Options{})
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestVerify_NoGoMod(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	_, err := Verify(ctx, tmpDir, Options{})
	if err == nil {
		t.Fatal("expected error when no go.mod is present, got nil")
	}
}

func TestVerify_ValidSmallGoProject(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Write go.mod
	goMod := "module example.com/testpkg\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Write simple main.go
	mainGo := `package main

import "fmt"

func Add(a, b int) int {
	return a + b
}

func main() {
	fmt.Println(Add(1, 2))
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Write main_test.go
	mainTest := `package main

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", got)
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(mainTest), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := Verify(ctx, tmpDir, Options{
		SkipTests: false,
		Race:      false,
		RunTidy:   true,
		Timeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected Verify error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected project verification to pass, failed with summary: %s", res.Summary)
	}
	if res.TestsPassed != 1 {
		t.Errorf("expected 1 test passed, got %d", res.TestsPassed)
	}
	if res.TestsFailed != 0 {
		t.Errorf("expected 0 tests failed, got %d", res.TestsFailed)
	}

	md := res.FormatMarkdown()
	if len(md) == 0 {
		t.Error("expected non-empty markdown output")
	}
}

func TestVerify_CompilationFailure(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	goMod := "module example.com/broken\n\ngo 1.23\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	brokenGo := `package main

func main() {
	thisIsACompileErrorUndefinedFunction()
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(brokenGo), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := Verify(ctx, tmpDir, Options{
		SkipTests: false,
		Race:      false,
		RunTidy:   false,
		Timeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Success {
		t.Fatal("expected verification to fail for broken compilation, but it passed")
	}

	md := res.FormatMarkdown()
	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}
