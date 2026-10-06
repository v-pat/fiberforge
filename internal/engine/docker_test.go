package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestDocker_NonRootRuntimeUser(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "dockertest",
		Framework: "fiber",
		Database:  "postgres",
		Port:      8080,
		Features:  schema.Features{Docker: true},
		Models: []schema.Model{
			{
				Name:     "item",
				Endpoint: "items",
				Fields: []schema.Field{
					{Name: "name", Type: schema.TypeString},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	dockerfilePath := filepath.Join(outDir, "Dockerfile")
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read generated Dockerfile: %v", err)
	}

	dockerStr := string(content)

	// Invariant: Non-root user created
	if !strings.Contains(dockerStr, "addgroup -S appgroup") || !strings.Contains(dockerStr, "adduser -S appuser -G appgroup") {
		t.Fatalf("Dockerfile must create unprivileged user/group, got:\n%s", dockerStr)
	}

	// Invariant: Switched to non-root USER
	if !strings.Contains(dockerStr, "USER appuser") {
		t.Fatalf("Dockerfile must switch to USER appuser before entrypoint")
	}

	// Verify USER directive comes before ENTRYPOINT
	userIndex := strings.Index(dockerStr, "USER appuser")
	entrypointIndex := strings.Index(dockerStr, "ENTRYPOINT")
	if userIndex == -1 || entrypointIndex == -1 || userIndex > entrypointIndex {
		t.Fatalf("USER directive must precede ENTRYPOINT directive")
	}
}
