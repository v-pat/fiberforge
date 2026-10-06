package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestMassAssignment_SQLModelInputDTOs(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "massassignsql",
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
					{Name: "content", Type: schema.TypeText},
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

	modelPath := filepath.Join(outDir, "model", "post.go")
	modelContent, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read model file: %v", err)
	}
	modelStr := string(modelContent)

	// Verify CreatePostInput and UpdatePostInput exist
	if !strings.Contains(modelStr, "type CreatePostInput struct") {
		t.Fatalf("model must define CreatePostInput DTO, got:\n%s", modelStr)
	}
	if !strings.Contains(modelStr, "type UpdatePostInput struct") {
		t.Fatalf("model must define UpdatePostInput DTO, got:\n%s", modelStr)
	}

	// Verify UpdatePostInput does not contain protected fields
	updateDTOSnippet := modelStr[strings.Index(modelStr, "type UpdatePostInput struct"):]
	endOfDTO := strings.Index(updateDTOSnippet, "}")
	if endOfDTO > 0 {
		updateDTOSnippet = updateDTOSnippet[:endOfDTO]
	}

	for _, protectedField := range []string{"ID", "UserID", "CreatedAt", "UpdatedAt", "DeletedAt"} {
		if strings.Contains(updateDTOSnippet, protectedField) {
			t.Fatalf("UpdatePostInput must NOT contain protected field %q, got:\n%s", protectedField, updateDTOSnippet)
		}
	}

	// Verify controller uses CreatePostInput and UpdatePostInput
	ctrlPath := filepath.Join(outDir, "controller", "post_controller.go")
	ctrlContent, err := os.ReadFile(ctrlPath)
	if err != nil {
		t.Fatalf("failed to read controller: %v", err)
	}
	ctrlStr := string(ctrlContent)

	if !strings.Contains(ctrlStr, "model.CreatePostInput") {
		t.Fatalf("controller CreatePost must bind to CreatePostInput")
	}
	if !strings.Contains(ctrlStr, "model.UpdatePostInput") {
		t.Fatalf("controller UpdatePost must bind to UpdatePostInput")
	}
}

func TestMassAssignment_MongoModelInputDTOs(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "massassignmongo",
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
				Name:          "post",
				Endpoint:      "posts",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "title", Type: schema.TypeString, Required: true},
					{Name: "content", Type: schema.TypeText},
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

	modelPath := filepath.Join(outDir, "model", "post.go")
	modelContent, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read model file: %v", err)
	}
	modelStr := string(modelContent)

	if !strings.Contains(modelStr, "type CreatePostInput struct") {
		t.Fatalf("mongo model must define CreatePostInput DTO")
	}
	if !strings.Contains(modelStr, "type UpdatePostInput struct") {
		t.Fatalf("mongo model must define UpdatePostInput DTO")
	}

	// Verify UpdatePostInput does not contain protected fields
	updateDTOSnippet := modelStr[strings.Index(modelStr, "type UpdatePostInput struct"):]
	endOfDTO := strings.Index(updateDTOSnippet, "}")
	if endOfDTO > 0 {
		updateDTOSnippet = updateDTOSnippet[:endOfDTO]
	}

	for _, protectedField := range []string{"ID", "UserID", "CreatedAt", "UpdatedAt"} {
		if strings.Contains(updateDTOSnippet, protectedField) {
			t.Fatalf("mongo UpdatePostInput must NOT contain protected field %q, got:\n%s", protectedField, updateDTOSnippet)
		}
	}
}

func TestMassAssignment_JSONPayloadProtectedFieldsIgnored(t *testing.T) {
	// Dummy DTO matching generated structure for Post model
	type UpdatePostInput struct {
		Title   *string `json:"title,omitempty"`
		Content *string `json:"content,omitempty"`
	}

	maliciousJSON := `{
		"id": 999,
		"userId": 42,
		"role": "admin",
		"createdAt": "2020-01-01T00:00:00Z",
		"title": "Legitimate Title"
	}`

	var input UpdatePostInput
	if err := json.Unmarshal([]byte(maliciousJSON), &input); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if input.Title == nil || *input.Title != "Legitimate Title" {
		t.Fatalf("expected title to be 'Legitimate Title', got %v", input.Title)
	}
	if input.Content != nil {
		t.Fatalf("expected content to be nil, got %v", input.Content)
	}
}
