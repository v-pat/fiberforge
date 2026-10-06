package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestErrorInformationDisclosure_SQLAndMongo(t *testing.T) {
	for _, db := range []string{"postgres", "mongodb"} {
		t.Run(db, func(t *testing.T) {
			cfg := &schema.Config{
				AppName:   "errtest",
				Framework: "fiber",
				Database:  db,
				Features:  schema.Features{Auth: true},
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

			itemControllerPath := filepath.Join(outDir, "controller", "item_controller.go")
			controllerBytes, err := os.ReadFile(itemControllerPath)
			if err != nil {
				t.Fatalf("failed to read item_controller.go: %v", err)
			}
			controllerSrc := string(controllerBytes)

			authControllerPath := filepath.Join(outDir, "controller", "auth_controller.go")
			authControllerBytes, err := os.ReadFile(authControllerPath)
			if err != nil {
				t.Fatalf("failed to read auth_controller.go: %v", err)
			}
			authControllerSrc := string(authControllerBytes)

			// In item_controller.go, SendError must never pass err.Error() directly to the client
			for _, line := range strings.Split(controllerSrc, "\n") {
				if strings.Contains(line, "SendError(") && strings.Contains(line, "err.Error()") {
					t.Errorf("found raw error disclosure in controller line: %s", line)
				}
			}

			// In auth_controller.go, SendError must never pass err.Error() directly to the client
			for _, line := range strings.Split(authControllerSrc, "\n") {
				if strings.Contains(line, "SendError(") && strings.Contains(line, "err.Error()") {
					t.Errorf("found raw error disclosure in auth controller line: %s", line)
				}
			}

			// Verify slog.Error is used for server-side logging of failures
			if !strings.Contains(controllerSrc, "slog.Error(") {
				t.Errorf("item_controller.go should log internal errors using slog.Error")
			}
		})
	}
}
