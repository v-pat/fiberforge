package blueprint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecordAndListAppliedBlueprints(t *testing.T) {
	tempDir := t.TempDir()

	initialYAML := `appName: testapp
database: postgres
port: 8080
`
	if err := os.WriteFile(filepath.Join(tempDir, "fiberforge.yaml"), []byte(initialYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := AppliedBlueprintRecord{
		ID:        "stripe-billing",
		Version:   "1.0.0",
		AppliedAt: "2026-10-26T12:00:00Z",
		Files:     []string{"pkg/stripe/stripe.go"},
		Models:    []string{"StripeCustomer", "Subscription", "Invoice"},
	}

	if err := RecordAppliedBlueprint(tempDir, rec); err != nil {
		t.Fatalf("RecordAppliedBlueprint failed: %v", err)
	}

	applied, err := ListAppliedBlueprints(tempDir)
	if err != nil {
		t.Fatalf("ListAppliedBlueprints failed: %v", err)
	}

	if len(applied) != 1 {
		t.Fatalf("expected 1 applied blueprint, got %d", len(applied))
	}

	if applied[0].ID != "stripe-billing" {
		t.Errorf("expected ID stripe-billing, got %s", applied[0].ID)
	}

	isApplied, record, err := IsBlueprintApplied(tempDir, "stripe-billing")
	if err != nil || !isApplied {
		t.Fatalf("expected IsBlueprintApplied to be true, got %v (err: %v)", isApplied, err)
	}
	if record.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", record.Version)
	}

	// Idempotent update: update record
	rec.Version = "1.1.0"
	if err := RecordAppliedBlueprint(tempDir, rec); err != nil {
		t.Fatalf("updating RecordAppliedBlueprint failed: %v", err)
	}

	appliedUpdated, _ := ListAppliedBlueprints(tempDir)
	if len(appliedUpdated) != 1 {
		t.Errorf("expected still 1 record after update, got %d", len(appliedUpdated))
	}
	if appliedUpdated[0].Version != "1.1.0" {
		t.Errorf("expected updated version 1.1.0, got %s", appliedUpdated[0].Version)
	}
}
