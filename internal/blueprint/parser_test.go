package blueprint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest_Valid(t *testing.T) {
	yamlData := `
id: resend-mailer
version: "1.0.0"
title: "Resend Mailer Blueprint"
description: "Send transactional emails via Resend"
category: "Communications"
tags: [email, resend]
target:
  language: go
  framework: fiber
env:
  - key: RESEND_API_KEY
    required: true
    description: "API Key"
dependencies:
  - package: github.com/resend/resend-go/v2
    version: v2.6.0
models:
  - name: EmailLog
    fields:
      - name: ToEmail
        type: string
        required: true
hooks:
  routes:
    - path: /webhooks/resend
      method: POST
      handler: mailer.HandleWebhook
files:
  - source: files/pkg/mailer/mailer.go
    target: pkg/mailer/mailer.go
`
	bp, err := ParseManifest([]byte(yamlData))
	if err != nil {
		t.Fatalf("unexpected error parsing valid manifest: %v", err)
	}

	if bp.ID != "resend-mailer" {
		t.Errorf("expected id 'resend-mailer', got %q", bp.ID)
	}
	if len(bp.Env) != 1 || bp.Env[0].Key != "RESEND_API_KEY" {
		t.Errorf("expected RESEND_API_KEY env var, got %+v", bp.Env)
	}
	if len(bp.Models) != 1 || bp.Models[0].Endpoint != "emaillogs" {
		t.Errorf("expected default endpoint 'emaillogs', got %q", bp.Models[0].Endpoint)
	}
	if len(bp.Hooks.Routes) != 1 || bp.Hooks.Routes[0].Method != "POST" {
		t.Errorf("expected route hook POST, got %+v", bp.Hooks.Routes)
	}
}

func TestParseManifest_PathTraversalRejection(t *testing.T) {
	traversals := []string{
		"../outside.go",
		"/etc/passwd",
		"pkg/../../outside.go",
	}

	for _, badPath := range traversals {
		yamlData := `
id: malicious-bp
version: "1.0.0"
title: "Malicious Blueprint"
description: "Tests traversal"
files:
  - source: payload.go
    target: ` + badPath + `
`
		_, err := ParseManifest([]byte(yamlData))
		if err == nil {
			t.Errorf("expected path traversal error for target %q, got nil", badPath)
		}
	}
}

func TestParseManifest_InvalidEnv(t *testing.T) {
	yamlData := `
id: invalid-env
version: "1.0.0"
title: "Invalid Env"
description: "Tests env"
env:
  - key: "invalid key with spaces"
    required: true
`
	_, err := ParseManifest([]byte(yamlData))
	if err == nil {
		t.Errorf("expected invalid env error, got nil")
	}
}

func TestParseBundle(t *testing.T) {
	tempDir := t.TempDir()

	manifest := `
id: test-bundle
version: "1.0.0"
title: "Test Bundle"
description: "A bundle test"
files:
  - source: files/sample.txt
    target: sample.txt
`
	if err := os.WriteFile(filepath.Join(tempDir, "blueprint.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	filesDir := filepath.Join(tempDir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "sample.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	bp, err := ParseBundle(tempDir)
	if err != nil {
		t.Fatalf("unexpected error parsing bundle: %v", err)
	}

	if len(bp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(bp.Files))
	}
	if bp.Files[0].Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", bp.Files[0].Content)
	}
}
