package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSafePath_Containment(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fiberforge-path-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure canonical path for tmpDir
	canonicalRoot, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		canonicalRoot = tmpDir
	}

	tests := []struct {
		name      string
		target    string
		shouldErr bool
	}{
		{"valid relative subdir", "my-app", false},
		{"valid nested subdir", "nested/deep/app", false},
		{"valid current dir", ".", false},
		{"valid empty dir", "", false},
		{"traversal out 1 level", "../escape", true},
		{"traversal out 2 levels", "../../escape", true},
		{"deep traversal", "../../../tmp/escape", true},
		{"sneaky traversal inside", "foo/../../escape", true},
		{"absolute path outside root", "/etc/passwd", true},
		{"absolute tmp outside", os.TempDir(), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := ResolveSafePath(tc.target, canonicalRoot)
			if tc.shouldErr {
				if err == nil {
					t.Fatalf("expected error for target %q, got resolved path %q", tc.target, resolved)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for target %q: %v", tc.target, err)
				}
				rel, err := filepath.Rel(canonicalRoot, resolved)
				if err != nil || rel == ".." || len(rel) >= 2 && rel[:2] == ".." {
					t.Fatalf("resolved path %q escaped canonical root %q", resolved, canonicalRoot)
				}
			}
		})
	}
}

func TestResolveSafePath_SymlinkEscape(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fiberforge-symlink-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	outsideDir, err := os.MkdirTemp("", "fiberforge-outside-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outsideDir)

	symlinkPath := filepath.Join(tmpDir, "symlink_outside")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skip("symlinks not supported on this platform/filesystem")
	}

	// Attempting to resolve a path traversing through the symlink to outsideDir
	_, err = ResolveSafePath(filepath.Join("symlink_outside", "subfolder"), tmpDir)
	if err == nil {
		t.Fatal("expected error when resolving path through symlink pointing outside root, but got nil")
	}
}

func TestValidateAppName(t *testing.T) {
	valid := []string{"myapp", "my_app", "my-app", "app123", "app-v1.0"}
	for _, v := range valid {
		if err := ValidateAppName(v); err != nil {
			t.Errorf("expected valid appName %q, got error: %v", v, err)
		}
	}

	invalid := []string{
		"../../escape",
		"foo/bar",
		"foo\\bar",
		"my app",
		"app;whoami",
		"app`whoami`",
		"app\nnewline",
	}
	for _, inv := range invalid {
		if err := ValidateAppName(inv); err == nil {
			t.Errorf("expected invalid appName %q to fail validation, but it passed", inv)
		}
	}
}
