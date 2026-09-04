package engine

import (
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

// fullConfig returns a config exercising most features.
func fullConfig() *schema.Config {
	return &schema.Config{
		AppName:  "blog",
		Database: "postgres",
		Features: schema.Features{
			Auth:       true,
			Docker:     true,
			Migrations: true,
			Swagger:    true,
			CORS:       true,
			Testing:    true,
			CI:         true,
		},
		Models: []schema.Model{
			{
				Name:          "Post",
				Endpoint:      "posts",
				AuthProtected: true,
				TableName:     "blog_posts",
				Fields: []schema.Field{
					{Name: "Title", Type: schema.TypeString, Required: true, OmitEmpty: true},
					{Name: "Body", Type: schema.TypeText, OmitEmpty: true},
					{Name: "Published", Type: schema.TypeBool, Default: strPtr("false")},
				},
			},
		},
	}
}

func strPtr(s string) *string { return &s }

// expectedFiles lists files that every generated project must contain.
var expectedFiles = []string{
	"go.mod", "README.md", ".gitignore", "main.go", "config/config.go",
	"databases/db.go", "routes/routes.go",
}

func TestGeneratePostgres(t *testing.T) {
	cfg := fullConfig()
	cfg.OutputDir = t.TempDir()
	e := New(cfg)
	if _, err := e.Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	checkProject(t, e.Dir())
}

func TestGenerateMongo(t *testing.T) {
	cfg := fullConfig()
	cfg.Database = "mongodb"
	cfg.OutputDir = t.TempDir()
	e := New(cfg)
	if _, err := e.Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	checkProject(t, e.Dir())
}

func TestGenerateMinimalNoFeatures(t *testing.T) {
	cfg := &schema.Config{
		AppName:  "store",
		Database: "mysql",
		Models: []schema.Model{
			{Name: "Item", Endpoint: "items", Fields: []schema.Field{
				{Name: "SKU", Type: schema.TypeString, Required: true, Unique: true},
			}},
		},
	}
	cfg.OutputDir = t.TempDir()
	e := New(cfg)
	if _, err := e.Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	checkProject(t, e.Dir())
	if _, err := os.Stat(filepath.Join(e.Dir(), "auth")); err == nil {
		t.Error("auth dir should not exist when auth feature is off")
	}
}

func checkProject(t *testing.T, dir string) {
	t.Helper()
	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected generated file %s: %v", f, err)
		}
	}
	assertGoFmtClean(t, dir)
}

func TestTableNameAndOmitEmpty(t *testing.T) {
	cfg := fullConfig()
	cfg.OutputDir = t.TempDir()
	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	modelFile, err := os.ReadFile(filepath.Join(cfg.OutputDir, "model", "post.go"))
	if err != nil {
		t.Fatalf("reading model/post.go: %v", err)
	}
	src := string(modelFile)
	if !strings.Contains(src, `func (Post) TableName() string`) {
		t.Error("expected TableName() method in model/post.go")
	}
	if !strings.Contains(src, `return "blog_posts"`) {
		t.Error("expected TableName() to return blog_posts")
	}
	if !strings.Contains(src, `json:"Title,omitempty"`) {
		t.Error("expected omitempty json tag on Title")
	}
}

func TestRelationships(t *testing.T) {
	cfg := &schema.Config{
		AppName:  "forum",
		Database: "postgres",
		Models: []schema.Model{
			{Name: "User", Endpoint: "users", Fields: []schema.Field{
				{Name: "Email", Type: schema.TypeString, Required: true},
			}, Relationships: []schema.Relationship{
				{Type: schema.HasMany, Model: "post"},
			}},
			{Name: "Post", Endpoint: "posts", Fields: []schema.Field{
				{Name: "Title", Type: schema.TypeString, Required: true},
			}, Relationships: []schema.Relationship{
				{Type: schema.BelongsTo, Model: "user"},
				{Type: schema.ManyToMany, Model: "tag"},
			}},
			{Name: "Tag", Endpoint: "tags", Fields: []schema.Field{
				{Name: "Label", Type: schema.TypeString},
			}},
		},
	}
	cfg.OutputDir = t.TempDir()
	if _, err := New(cfg).Generate(); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	user, err := os.ReadFile(filepath.Join(cfg.OutputDir, "model", "user.go"))
	if err != nil {
		t.Fatalf("reading model/user.go: %v", err)
	}
	if !regexp.MustCompile(`\bPosts\s+\[\]Post\b`).MatchString(string(user)) {
		t.Error("expected hasMany Posts slice on User")
	}
	post, err := os.ReadFile(filepath.Join(cfg.OutputDir, "model", "post.go"))
	if err != nil {
		t.Fatalf("reading model/post.go: %v", err)
	}
	src := string(post)
	if !regexp.MustCompile(`\bUserID\s+uint\b`).MatchString(src) {
		t.Error("expected belongsTo UserID FK on Post")
	}
	if !regexp.MustCompile(`\bUser\s+User\b`).MatchString(src) {
		t.Error("expected belongsTo User association on Post")
	}
	if !regexp.MustCompile(`\bTags\s+\[\]Tag\b`).MatchString(src) {
		t.Error("expected manyToMany Tags slice on Post")
	}
	assertGoFmtClean(t, cfg.OutputDir)
}

func TestAuthUserModelValidation(t *testing.T) {
	cfg := &schema.Config{
		AppName:   "x",
		Framework: "fiber",
		Database:  "postgres",
		Features:  schema.Features{Auth: true},
		Models: []schema.Model{
			{Name: "User", Endpoint: "users", Fields: []schema.Field{
				{Name: "Email", Type: schema.TypeString},
			}},
		},
	}
	if err := schema.Validate(cfg); err == nil {
		t.Error("expected error when auth user model lacks password field")
	}
	cfg.Models[0].Fields = append(cfg.Models[0].Fields, schema.Field{Name: "Password", Type: schema.TypePassword})
	if err := schema.Validate(cfg); err != nil {
		t.Errorf("auth user with email+password should validate: %v", err)
	}
}

func assertGoFmtClean(t *testing.T, dir string) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := format.Source(src); err != nil {
			t.Errorf("%s is not gofmt-clean: %v", rel(dir, path), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking generated dir: %v", err)
	}
}

func rel(base, path string) string {
	r, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return r
}

func TestAppModule(t *testing.T) {
	cases := map[string]string{
		"blog":    "blog",
		"MyApp":   "myapp",
		"My-App":  "my-app",
		"blog_v2": "blog_v2",
	}
	for in, want := range cases {
		cfg := &schema.Config{AppName: in}
		if got := New(cfg).AppModule(); got != want {
			t.Errorf("AppModule(%q) = %q, want %q", in, got, want)
		}
	}
}
