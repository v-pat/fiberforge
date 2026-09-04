package schema

import (
	"strings"
	"testing"
)

func validYAML() string {
	return `appName: blog
database: postgres
models:
  - name: user
    endpoint: users
    fields:
      - name: email
        type: string
        required: true
  - name: post
    endpoint: posts
    fields:
      - name: title
        type: string
`
}

func TestParseValidYAML(t *testing.T) {
	cfg, err := Parse([]byte(validYAML()))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cfg.AppName != "blog" {
		t.Errorf("AppName = %q, want blog", cfg.AppName)
	}
	if cfg.Framework != "fiber" {
		t.Errorf("Framework = %q, want default fiber", cfg.Framework)
	}
	if cfg.Database != "postgres" {
		t.Errorf("Database = %q, want postgres", cfg.Database)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want default 8080", cfg.Port)
	}
	if len(cfg.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(cfg.Models))
	}
}

func TestParseJSON(t *testing.T) {
	src := `{"appName":"api","database":"mongodb","features":{"auth":true},"models":[{"name":"doc","endpoint":"docs","fields":[{"name":"body","type":"text"}]}]}`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cfg.AppName != "api" {
		t.Errorf("AppName = %q, want api", cfg.AppName)
	}
	if cfg.Database != "mongodb" {
		t.Errorf("Database = %q, want mongodb", cfg.Database)
	}
	if !cfg.Features.Auth {
		t.Error("Features.Auth = false, want true")
	}
}

func TestParseMissingDatabase(t *testing.T) {
	_, err := Parse([]byte(`appName: x
models: []`))
	if err == nil {
		t.Fatal("expected error for missing database")
	}
	if !strings.Contains(err.Error(), "database") {
		t.Errorf("error should mention database, got: %v", err)
	}
}

func TestParseMissingAppName(t *testing.T) {
	_, err := Parse([]byte("database: mysql\nmodels: []"))
	if err == nil {
		t.Fatal("expected error for missing appName")
	}
	if !strings.Contains(err.Error(), "appName") {
		t.Errorf("error should mention appName, got: %v", err)
	}
}

func TestParseUnsupportedDatabase(t *testing.T) {
	_, err := Parse([]byte("appName: x\ndatabase: sqlite\nmodels: []"))
	if err == nil {
		t.Fatal("expected error for unsupported database")
	}
	if !strings.Contains(err.Error(), "unsupported database") {
		t.Errorf("error should mention unsupported database, got: %v", err)
	}
}

func TestParseUnsupportedFramework(t *testing.T) {
	_, err := Parse([]byte("appName: x\ndatabase: postgres\nframework: echo\nmodels: []"))
	if err == nil {
		t.Fatal("expected error for unsupported framework")
	}
}

func TestParseDuplicateModels(t *testing.T) {
	src := `appName: x
database: postgres
models:
  - name: User
    endpoint: users
    fields: [{name: email, type: string}]
  - name: user
    endpoint: users2
    fields: [{name: n, type: string}]
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("expected duplicate model error")
	}
}

func TestParseMissingModelEndpoint(t *testing.T) {
	src := `appName: x
database: postgres
models:
  - name: thing
    fields: [{name: n, type: string}]
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("expected missing endpoint error")
	}
}

func TestParseEnumNeedsValues(t *testing.T) {
	src := `appName: x
database: postgres
models:
  - name: thing
    endpoint: things
    fields:
      - name: status
        type: enum
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("expected enum-needs-values error")
	}
}

func TestParseUnsupportedFieldType(t *testing.T) {
	src := `appName: x
database: postgres
models:
  - name: thing
    endpoint: things
    fields:
      - name: n
        type: blob
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("expected unsupported field type error")
	}
}

func TestParseRelationshipUnknownModel(t *testing.T) {
	src := `appName: x
database: postgres
models:
  - name: article
    endpoint: articles
    fields: [{name: t, type: string}]
    relationships:
      - type: belongsTo
        model: ghost
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("expected unknown relationship model error")
	}
}

func TestApplyDefaultsDatabaseLowercased(t *testing.T) {
	cfg := &Config{AppName: "x", Database: "Postgres", Models: []Model{}}
	if err := ApplyDefaults(cfg); err != nil {
		t.Fatalf("ApplyDefaults: %v", err)
	}
	if cfg.Database != "postgres" {
		t.Errorf("Database = %q, want lowercased postgres", cfg.Database)
	}
}
