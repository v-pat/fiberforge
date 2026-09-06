package modules_test

import (
	"testing"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/modules"
	"github.com/v-pat/fiberforge/internal/schema"
)

func TestModulesCatalog(t *testing.T) {
	list := modules.List()
	if len(list) != 22 {
		t.Errorf("expected 22 modules, got %d", len(list))
	}

	stripe, err := modules.Get("stripe-billing")
	if err != nil {
		t.Fatalf("failed to get stripe-billing: %v", err)
	}
	if stripe.Name != "stripe-billing" {
		t.Errorf("expected stripe-billing, got %s", stripe.Name)
	}

	ai, err := modules.Get("ai-inference")
	if err != nil {
		t.Fatalf("failed to get ai-inference: %v", err)
	}
	if ai.Name != "ai-inference" {
		t.Errorf("expected ai-inference, got %s", ai.Name)
	}
}

func TestApplyModule(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &schema.Config{
		AppName:  "storeapp",
		Database: "postgres",
		Models: []schema.Model{
			{Name: "User", Endpoint: "users"},
		},
	}
	cfg.OutputDir = tmpDir
	eng := engine.New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("failed initial generate: %v", err)
	}

	added, err := modules.Apply(tmpDir, "stripe-billing")
	if err != nil {
		t.Fatalf("Apply module failed: %v", err)
	}
	if len(added) == 0 {
		t.Errorf("expected added models, got 0")
	}
}

func TestApplyModuleDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &schema.Config{
		AppName:  "rbacapp",
		Database: "postgres",
		Models: []schema.Model{
			{Name: "User", Endpoint: "users"},
		},
	}
	cfg.OutputDir = tmpDir
	eng := engine.New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("failed initial generate: %v", err)
	}

	added, files, err := modules.ApplyWithOptions(tmpDir, "rbac", true)
	if err != nil {
		t.Fatalf("ApplyWithOptions dryRun failed: %v", err)
	}
	if len(added) == 0 {
		t.Errorf("expected added models in dryRun, got 0")
	}
	if len(files) == 0 {
		t.Errorf("expected affected file paths preview in dryRun, got 0")
	}
}
