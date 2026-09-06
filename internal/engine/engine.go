package engine

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/v-pat/fiberforge/internal/schema"
)

// Engine turns a schema.Config into a generated project on disk.
type Engine struct {
	cfg *schema.Config
	dir string // output directory prefix
}

// New creates an Engine for the given config. The output directory is
// config.OutputDir if set, otherwise "./<appName>".
func New(cfg *schema.Config) *Engine {
	dir := cfg.OutputDir
	if dir == "" {
		dir = "./" + cfg.AppName
	}
	return &Engine{cfg: cfg, dir: dir}
}

// Config exposes the underlying config to the MCP layer.
func (e *Engine) Config() *schema.Config { return e.cfg }

// Dir returns the output directory.
func (e *Engine) Dir() string { return e.dir }

// AppModule returns the Go module path used for generated imports. Workspace
// module directories and import paths must match, so it is derived solely from
// the app name with no extra suffixes.
func (e *Engine) AppModule() string {
	return sanitizeModule(e.cfg.AppName)
}

// sanitizeModule makes a safe Go module identifier from an arbitrary app name.
func sanitizeModule(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.ToLower(b.String())
}

// funcs returns the template function map shared by all generators.
func (e *Engine) funcs() template.FuncMap {
	return template.FuncMap{
		"pascal": schema.Pascal,
		"camel":  schema.Camel,
		"lower":  schema.Lower,
		"plural": schema.Plural,
	}
}

// render executes a named template with the shared func map.
func (e *Engine) render(name, source string, data any) (string, error) {
	t, err := template.New(name).Funcs(e.funcs()).Parse(source)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// write ensures dirs exist and writes a file with the given content. Go source
// files are formatted with gofmt so the generated output is clean.
func (e *Engine) write(relPath, content string) error {
	abs := filepath.Join(e.dir, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	data := []byte(content)
	if filepath.Ext(relPath) == ".go" {
		formatted, err := format.Source(data)
		if err != nil {
			_ = os.WriteFile(abs, data, 0o644)
			return fmt.Errorf("gofmt %s: %w", relPath, err)
		}
		data = formatted
	}
	return os.WriteFile(abs, data, 0o644)
}

// Files returns the list of relative file paths that Generate produces.
func (e *Engine) Files() []string {
	var files []string
	files = append(files, "go.mod", "README.md", ".gitignore", ".env.example", "fiberforge.yaml", "main.go", filepath.Join("config", "config.go"), filepath.Join("databases", "db.go"))
	for _, m := range e.cfg.Models {
		name := strings.ToLower(schema.Pascal(m.Name))
		files = append(files, filepath.Join("model", name+".go"))
		if !(e.cfg.Features.Auth && strings.EqualFold(m.Name, "user")) {
			files = append(files, filepath.Join("service", name+"_service.go"))
			files = append(files, filepath.Join("controller", name+"_controller.go"))
		}
	}
	if e.cfg.Features.Auth {
		files = append(files, "auth/password.go", "auth/jwt.go", "auth/store.go", "middleware/jwt.go", "controller/auth_controller.go", "service/user_service.go")
	}
	files = append(files, filepath.Join("routes", "routes.go"))
	if e.cfg.Features.Migrations && !e.isMongo() {
		for i, m := range e.cfg.Models {
			name := fmt.Sprintf("%06d_%s", i+1, strings.ToLower(schema.Pascal(m.Name)))
			files = append(files, filepath.Join("migrations", name+".up.sql"))
			files = append(files, filepath.Join("migrations", name+".down.sql"))
		}
	}
	if e.cfg.Features.Docker {
		files = append(files, "Dockerfile", "docker-compose.yml")
	}
	if e.cfg.Features.CI {
		files = append(files, filepath.Join(".github", "workflows", "ci.yml"))
	}
	if e.cfg.Features.Docker || e.cfg.Features.Migrations {
		files = append(files, "Makefile")
	}
	if e.cfg.Features.Testing {
		for _, m := range e.cfg.Models {
			if !(strings.EqualFold(m.Name, "user") && e.cfg.Features.Auth) {
				files = append(files, filepath.Join("controller", strings.ToLower(schema.Pascal(m.Name))+"_test.go"))
			}
		}
		if e.cfg.Features.Auth {
			files = append(files, filepath.Join("controller", "auth_test.go"))
		}
	}
	if e.cfg.Features.Swagger {
		files = append(files, filepath.Join("docs", "swagger.json"))
	}
	return files
}

// dbDriver returns the gorm/migrate driver name for the database, or "" for mongo.
func (e *Engine) dbDriver() string {
	if e.cfg.Database == "postgres" {
		return "postgres"
	}
	if e.cfg.Database == "mysql" {
		return "mysql"
	}
	return ""
}

// isMongo reports whether the target database is MongoDB.
func (e *Engine) isMongo() bool { return e.cfg.Database == "mongodb" }
