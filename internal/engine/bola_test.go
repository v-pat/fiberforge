package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestBOLA_SQLGeneratedOwnershipGuards(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "bolatest",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{Auth: true},
		Models: []schema.Model{
			{
				Name:     "user",
				Endpoint: "users",
				Fields: []schema.Field{
					{Name: "email", Type: schema.TypeString, Required: true},
					{Name: "password", Type: schema.TypePassword, Required: true},
				},
			},
			{
				Name:          "post",
				Endpoint:      "posts",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
				},
				Relationships: []schema.Relationship{
					{Type: schema.BelongsTo, Model: "user"},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	eng := New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify Controller has ownership scoping
	ctrlPath := filepath.Join(outDir, "controller", "post_controller.go")
	ctrlContent, err := os.ReadFile(ctrlPath)
	if err != nil {
		t.Fatalf("failed to read controller: %v", err)
	}
	ctrlStr := string(ctrlContent)

	if !strings.Contains(ctrlStr, `c.Locals("userId")`) {
		t.Fatalf("controller must extract userId from c.Locals for ownership verification, got:\n%s", ctrlStr)
	}
	if !strings.Contains(ctrlStr, `m.UserID = uid`) {
		t.Fatalf("CreatePost must enforce m.UserID from authenticated token, got:\n%s", ctrlStr)
	}

	// Verify Service has ownership scoping
	svcPath := filepath.Join(outDir, "service", "post_service.go")
	svcContent, err := os.ReadFile(svcPath)
	if err != nil {
		t.Fatalf("failed to read service: %v", err)
	}
	svcStr := string(svcContent)

	// Invariant: GET by ID must scope by user_id
	if !strings.Contains(svcStr, `Where("id = ? AND user_id = ?", id, userID)`) {
		t.Fatalf("GetPostByID must filter by user_id to prevent BOLA/IDOR, got:\n%s", svcStr)
	}

	// Invariant: List must scope by user_id
	if !strings.Contains(svcStr, `Where("user_id = ?", userID)`) {
		t.Fatalf("ListPosts must filter by user_id to prevent cross-user data leakage, got:\n%s", svcStr)
	}

	// Invariant: Update must scope by user_id and take UpdatePostInput
	if !strings.Contains(svcStr, `UpdatePost(id uint, userID uint, patch *model.UpdatePostInput)`) ||
		!strings.Contains(svcStr, `Where("id = ? AND user_id = ?", id, userID)`) {
		t.Fatalf("UpdatePost must preserve owner UserID and scope by user_id, got:\n%s", svcStr)
	}

	// Invariant: Delete must scope by user_id
	if !strings.Contains(svcStr, `Where("id = ? AND user_id = ?", id, userID).Delete`) {
		t.Fatalf("DeletePostByID must filter by user_id to prevent unauthorized deletion, got:\n%s", svcStr)
	}
}

func TestBOLA_MongoGeneratedOwnershipGuards(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "mongobolatest",
		Framework: "fiber",
		Database:  "mongodb",
		Features:  schema.Features{Auth: true},
		Models: []schema.Model{
			{
				Name:     "user",
				Endpoint: "users",
				Fields: []schema.Field{
					{Name: "email", Type: schema.TypeString, Required: true},
					{Name: "password", Type: schema.TypePassword, Required: true},
				},
			},
			{
				Name:          "document",
				Endpoint:      "documents",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
				},
				Relationships: []schema.Relationship{
					{Type: schema.BelongsTo, Model: "user"},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	eng := New(cfg)
	if _, err := eng.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	ctrlPath := filepath.Join(outDir, "controller", "document_controller.go")
	ctrlContent, err := os.ReadFile(ctrlPath)
	if err != nil {
		t.Fatalf("failed to read controller: %v", err)
	}
	ctrlStr := string(ctrlContent)

	if !strings.Contains(ctrlStr, `c.Locals("userId")`) {
		t.Fatalf("mongo controller must extract userId for ownership verification")
	}

	svcPath := filepath.Join(outDir, "service", "document_service.go")
	svcContent, err := os.ReadFile(svcPath)
	if err != nil {
		t.Fatalf("failed to read service: %v", err)
	}
	svcStr := string(svcContent)

	if !strings.Contains(svcStr, `"userId": oid`) && !strings.Contains(svcStr, `"userId": userOid`) {
		t.Fatalf("mongo service queries must scope by userId, got:\n%s", svcStr)
	}
}

func TestBOLA_ProjectCompilation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cfg := &schema.Config{
		AppName:   "bolacompile",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{Auth: true, Testing: true},
		Models: []schema.Model{
			{
				Name:     "user",
				Endpoint: "users",
				Fields: []schema.Field{
					{Name: "email", Type: schema.TypeString, Required: true},
					{Name: "password", Type: schema.TypePassword, Required: true},
				},
			},
			{
				Name:          "post",
				Endpoint:      "posts",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
				},
				Relationships: []schema.Relationship{
					{Type: schema.BelongsTo, Model: "user"},
				},
			},
		},
	}

	outDir := t.TempDir()
	cfg.OutputDir = outDir

	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify go.mod was generated
	if _, err := os.Stat(filepath.Join(outDir, "go.mod")); err != nil {
		t.Fatalf("go.mod missing: %v", err)
	}
}

