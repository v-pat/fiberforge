package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestExplicitOwnership_OwnerTrue(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "ownertest",
		Framework: "fiber",
		Database:  "postgres",
		Features: schema.Features{
			Auth: true,
		},
		Models: []schema.Model{
			{
				Name:     "document",
				Endpoint: "documents",
				Owner:    true, // Explicit ownership declaration
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
					{Name: "content", Type: schema.TypeText},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 1. Controller must enforce m.UserID from authenticated token
	ctrlPath := filepath.Join(outDir, "controller", "document_controller.go")
	ctrlBytes, err := os.ReadFile(ctrlPath)
	if err != nil {
		t.Fatalf("failed to read document_controller.go: %v", err)
	}
	ctrlStr := string(ctrlBytes)

	if !strings.Contains(ctrlStr, "m.UserID = uid") {
		t.Fatalf("CreateDocument must set m.UserID from authenticated session token, got:\n%s", ctrlStr)
	}
	if !strings.Contains(ctrlStr, `c.Locals("userId").(uint)`) {
		t.Fatalf("controller must extract authenticated userId from context locals")
	}

	// 2. Service must scope queries by user_id
	svcPath := filepath.Join(outDir, "service", "document_service.go")
	svcBytes, err := os.ReadFile(svcPath)
	if err != nil {
		t.Fatalf("failed to read document_service.go: %v", err)
	}
	svcStr := string(svcBytes)

	if !strings.Contains(svcStr, `Where("user_id = ?", userID)`) {
		t.Fatalf("ListDocuments must filter by user_id, got:\n%s", svcStr)
	}
	if !strings.Contains(svcStr, `Where("id = ? AND user_id = ?", id, userID)`) {
		t.Fatalf("GetDocumentByID / UpdateDocument must scope query by user_id, got:\n%s", svcStr)
	}

	// 3. Model must contain UserID field, and DTOs must omit UserID
	modelPath := filepath.Join(outDir, "model", "document.go")
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read document.go: %v", err)
	}
	modelStr := string(modelBytes)

	if !strings.Contains(modelStr, "UserID") {
		t.Fatalf("model Document must define UserID field")
	}
	if strings.Contains(modelStr, "CreateDocumentInput struct {\n\tUserID") {
		t.Fatalf("CreateDocumentInput must NOT expose UserID for client assignment")
	}
}

func TestExplicitOwnership_SharedAuthenticatedModel(t *testing.T) {
	// A model with auth: true but NOT owner: true and NO belongsTo: user
	// is a shared authenticated resource (e.g. product catalog)
	cfg := &schema.Config{
		AppName:   "sharedtest",
		Framework: "fiber",
		Database:  "postgres",
		Features: schema.Features{
			Auth: true,
		},
		Models: []schema.Model{
			{
				Name:          "product",
				Endpoint:      "products",
				AuthProtected: true, // Requires JWT to access
				Owner:         false, // Not user-owned
				Fields: []schema.Field{
					{Name: "sku", Type: schema.TypeString, Required: true},
					{Name: "name", Type: schema.TypeString, Required: true},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	ctrlPath := filepath.Join(outDir, "controller", "product_controller.go")
	ctrlBytes, err := os.ReadFile(ctrlPath)
	if err != nil {
		t.Fatalf("failed to read product_controller.go: %v", err)
	}
	ctrlStr := string(ctrlBytes)

	// Invariant: Shared resource must NOT force m.UserID
	if strings.Contains(ctrlStr, "m.UserID = uid") {
		t.Fatalf("shared model product should not inject m.UserID")
	}

	// Routes must protect product with JWT middleware
	routesPath := filepath.Join(outDir, "routes", "routes.go")
	routesBytes, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("failed to read routes.go: %v", err)
	}
	routesStr := string(routesBytes)

	if !strings.Contains(routesStr, `Use(middleware.JWT(auth.Secret()))`) {
		t.Fatalf("routes for authProtected model must be wrapped in JWT middleware")
	}
}

func TestExplicitOwnership_AmbiguousFieldNameDoesNotInferOwnership(t *testing.T) {
	// A model with ambiguous field name like 'authorId' or 'creatorId'
	// must NOT be inferred as user-owned unless owner: true or belongsTo: user is explicitly declared
	cfg := &schema.Config{
		AppName:   "ambiguoustest",
		Framework: "fiber",
		Database:  "postgres",
		Features: schema.Features{
			Auth: true,
		},
		Models: []schema.Model{
			{
				Name:          "article",
				Endpoint:      "articles",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
					{Name: "authorId", Type: schema.TypeString},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	ctrlPath := filepath.Join(outDir, "controller", "article_controller.go")
	ctrlBytes, err := os.ReadFile(ctrlPath)
	if err != nil {
		t.Fatalf("failed to read article_controller.go: %v", err)
	}
	ctrlStr := string(ctrlBytes)

	// Invariant: authorId does NOT trigger automatic user ownership
	if strings.Contains(ctrlStr, "m.UserID = uid") {
		t.Fatalf("ambiguous field authorId must not silently infer user ownership without explicit declaration")
	}
}

func TestExplicitOwnership_OwnerRequiresAuthFeature(t *testing.T) {
	yamlContent := `
appName: invalidowner
database: postgres
features:
  auth: false
models:
  - name: note
    owner: true
    fields:
      - name: text
        type: string
`
	_, err := schema.Parse([]byte(yamlContent))
	if err == nil {
		t.Fatalf("schema.Parse must reject owner: true when features.auth is false")
	}
	if !strings.Contains(err.Error(), "features.auth is not enabled") {
		t.Fatalf("expected error explaining features.auth requirement, got: %v", err)
	}
}
