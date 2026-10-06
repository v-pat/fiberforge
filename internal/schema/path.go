package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var appNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// ValidateAppName verifies that appName is a valid identifier without path traversal characters.
func ValidateAppName(name string) error {
	if !appNameRe.MatchString(name) || strings.Contains(name, "..") {
		return fmt.Errorf("appName %q contains invalid characters (must match %s without '..')", name, appNameRe.String())
	}
	return nil
}

// ResolveSafePath ensures targetPath resolves to a canonical location strictly within rootDir.
// If rootDir is empty, the current working directory is used.
// It safely handles relative traversals, absolute escapes, and symlink targets.
func ResolveSafePath(targetPath, rootDir string) (string, error) {
	if rootDir == "" {
		rootDir = os.Getenv("FIBERFORGE_WORKSPACE_ROOT")
	}

	cleanTarget := filepath.Clean(targetPath)
	if rootDir == "" && filepath.IsAbs(cleanTarget) {
		canonicalTarget, err := filepath.EvalSymlinks(cleanTarget)
		if err != nil {
			return filepath.Abs(cleanTarget)
		}
		return filepath.Abs(canonicalTarget)
	}

	if rootDir == "" {
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	canonicalRoot, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		canonicalRoot, err = filepath.Abs(rootDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve root directory %q: %w", rootDir, err)
		}
	} else {
		canonicalRoot, err = filepath.Abs(canonicalRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute root %q: %w", canonicalRoot, err)
		}
	}

	if !filepath.IsAbs(cleanTarget) {
		cleanTarget = filepath.Join(canonicalRoot, cleanTarget)
	}

	// Canonicalize existing ancestors to resolve any symlinks along the path
	curr := cleanTarget
	var uncreated []string
	for {
		fi, err := os.Lstat(curr)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				resolved, err := filepath.EvalSymlinks(curr)
				if err != nil {
					return "", fmt.Errorf("failed to resolve symlink %q: %w", curr, err)
				}
				curr = resolved
			}
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		uncreated = append([]string{filepath.Base(curr)}, uncreated...)
		curr = parent
	}

	canonicalCurr, err := filepath.EvalSymlinks(curr)
	if err == nil {
		curr = canonicalCurr
	}
	curr, err = filepath.Abs(curr)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path %q: %w", curr, err)
	}

	finalPath := curr
	for _, part := range uncreated {
		finalPath = filepath.Join(finalPath, part)
	}
	finalPath = filepath.Clean(finalPath)

	rel, err := filepath.Rel(canonicalRoot, finalPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q resolves to %q which is outside root %q", targetPath, finalPath, canonicalRoot)
	}

	return finalPath, nil
}
