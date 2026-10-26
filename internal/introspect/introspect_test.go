package introspect

import (
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestTableToModelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "User"},
		{"posts", "Post"},
		{"order_items", "OrderItem"},
		{"categories", "Category"},
		{"user_addresses", "UserAddress"},
	}

	for _, tt := range tests {
		got := TableToModelName(tt.input)
		if got != tt.want {
			t.Errorf("TableToModelName(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestColumnToFieldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"email", "email"},
		{"first_name", "firstName"},
		{"order_item_id", "orderItemId"},
		{"published_at", "publishedAt"},
	}

	for _, tt := range tests {
		got := ColumnToFieldName(tt.input)
		if got != tt.want {
			t.Errorf("ColumnToFieldName(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestIntrospectSQL_Basic(t *testing.T) {
	sqlInput := `
CREATE TABLE users (
	id BIGSERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL UNIQUE,
	password VARCHAR(255) NOT NULL,
	role VARCHAR(50) DEFAULT 'user',
	created_at TIMESTAMP,
	updated_at TIMESTAMP
);

CREATE TABLE posts (
	id BIGSERIAL PRIMARY KEY,
	title VARCHAR(255) NOT NULL,
	body TEXT,
	published BOOLEAN DEFAULT false,
	user_id BIGINT NOT NULL,
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id)
);
`

	cfg, err := IntrospectSQL(sqlInput, "blog-api", "postgres")
	if err != nil {
		t.Fatalf("IntrospectSQL failed: %v", err)
	}

	if cfg.AppName != "blog-api" {
		t.Errorf("expected appName blog-api, got %s", cfg.AppName)
	}
	if len(cfg.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(cfg.Models))
	}

	userModel := cfg.Models[0]
	if userModel.Name != "User" {
		t.Errorf("expected model User, got %s", userModel.Name)
	}
	if !cfg.Features.Auth {
		t.Error("expected auth feature to be detected from User model with password")
	}

	postModel := cfg.Models[1]
	if postModel.Name != "Post" {
		t.Errorf("expected model Post, got %s", postModel.Name)
	}
	if len(postModel.Relationships) != 1 {
		t.Fatalf("expected 1 relationship on Post, got %d", len(postModel.Relationships))
	}
	if postModel.Relationships[0].Model != "User" {
		t.Errorf("expected relationship to User, got %s", postModel.Relationships[0].Model)
	}
	if postModel.Relationships[0].Type != schema.BelongsTo {
		t.Errorf("expected belongsTo relationship, got %s", postModel.Relationships[0].Type)
	}

	yamlStr, err := ToYAML(cfg)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}
	if len(yamlStr) == 0 {
		t.Error("expected non-empty YAML string")
	}

	// Validate parsed schema passes FiberForge validator!
	parsedCfg, err := schema.Parse([]byte(yamlStr))
	if err != nil {
		t.Fatalf("generated YAML failed schema.Parse: %v\nYAML was:\n%s", err, yamlStr)
	}
	if len(parsedCfg.Models) != 2 {
		t.Errorf("expected 2 models in parsed schema, got %d", len(parsedCfg.Models))
	}
}
