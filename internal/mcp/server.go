package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/v-pat/fiberforge/examples"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/modules"
	"github.com/v-pat/fiberforge/internal/schema"
)

// Server is a minimal MCP server exposing FiberForge tools over stdio using
// JSON-RPC 2.0. It lets an AI agent generate projects by providing a schema.
//
// The server is provider-agnostic: it never calls an LLM itself. The agent
// (which already has model access) calls the tools; the engine keeps code
// generation deterministic.
type Server struct {
	in      io.Reader
	out     io.Writer
	Version string
}

// NewServer creates an MCP server reading from in and writing to out.
func NewServer(in io.Reader, out io.Writer, version string) *Server {
	if version == "" {
		version = "dev"
	}
	return &Server{in: in, out: out, Version: version}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve runs the request/response loop until EOF.
func (s *Server) Serve() error {
	sc := bufio.NewScanner(s.in)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		s.dispatch(req)
	}
	return sc.Err()
}

func (s *Server) dispatch(req rpcRequest) {
	var result any
	var rerr *rpcError
	id := req.ID

	switch req.Method {
	case "initialize":
		result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{"name": "fiberforge", "version": s.Version},
		}
	case "notifications/initialized", "notifications/cancelled":
		// Fire and forget; no reply.
		return
	case "ping":
		result = map[string]any{}
	case "tools/list":
		result = map[string]any{"tools": listTools()}
	case "tools/call":
		result, rerr = s.callTool(req.Params)
	case "resources/list", "tools/resources/list":
		result = map[string]any{"resources": listResources()}
	case "resources/read":
		result, rerr = s.readResource(req.Params)
	default:
		rerr = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}

	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result, Error: rerr}
	s.write(resp)
}

func (s *Server) callTool(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
	}

	switch p.Name {
	case "generate_project":
		return s.generateProject(p.Arguments)
	case "validate_schema":
		return s.validateSchema(p.Arguments)
	case "get_schema_template":
		return s.getSchemaTemplate(p.Arguments)
	case "list_field_types":
		return s.listFieldTypes()
	case "explain_project":
		return s.explainProject(p.Arguments)
	case "add_model":
		return s.addModel(p.Arguments)
	case "list_modules":
		return s.listModules()
	case "apply_module":
		return s.applyModule(p.Arguments)
	default:
		return nil, &rpcError{Code: -32601, Message: "unknown tool: " + p.Name}
	}
}

func emptyArgs(args json.RawMessage) error {
	if len(args) == 0 || string(args) == "{}" {
		return fmt.Errorf("missing required argument 'schema'")
	}
	return nil
}

func (s *Server) generateProject(args json.RawMessage) (any, *rpcError) {
	if err := emptyArgs(args); err != nil {
		return nil, &rpcError{Code: -32602, Message: err.Error()}
	}
	var input struct {
		Schema   string `json:"schema"`
		Output   string `json:"outputDir,omitempty"`
		Language string `json:"language,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid arguments: " + err.Error()}
	}
	if input.Schema == "" {
		return nil, &rpcError{Code: -32602, Message: "argument 'schema' is required (YAML or JSON)"}
	}

	cfg, err := schema.Parse([]byte(input.Schema))
	if err != nil {
		return map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Invalid schema: " + err.Error(),
				},
			},
			"isError": true,
		}, nil
	}
	if input.Output != "" {
		cfg.OutputDir = input.Output
	}

	// Workspace root guard: ensure output directory stays within CWD.
	if cfg.OutputDir != "" {
		absOut, err := filepath.Abs(cfg.OutputDir)
		if err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid outputDir: " + err.Error()}
		}
		cwd, err := os.Getwd()
		if err != nil {
			return nil, &rpcError{Code: -32603, Message: "cannot determine working directory: " + err.Error()}
		}
		if !strings.HasPrefix(absOut, cwd+string(filepath.Separator)) && absOut != cwd {
			return map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": fmt.Sprintf("outputDir %q resolves to %q which is outside the workspace root %q. Generation refused for safety.", cfg.OutputDir, absOut, cwd),
					},
				},
				"isError": true,
			}, nil
		}
	}

	eng := engine.New(cfg)
	dir, err := eng.Generate()
	if err != nil {
		return map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Generation failed: " + err.Error(),
				},
			},
			"isError": true,
		}, nil
	}

	// Collect endpoints for structured output.
	var endpoints []string
	for _, m := range cfg.Models {
		endpoints = append(endpoints, "/api/"+m.Endpoint)
	}
	if cfg.Features.Auth {
		endpoints = append(endpoints, "/api/auth/register", "/api/auth/login", "/api/auth/me", "/api/auth/refresh")
	}
	endpoints = append(endpoints, "/health/live", "/health/ready")

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": fmt.Sprintf("Project generated successfully at %s. Run: cd %s && go mod tidy && go run .", dir, dir),
			},
		},
		"outputDir":    dir,
		"appName":      cfg.AppName,
		"database":     cfg.Database,
		"modelCount":   len(cfg.Models),
		"featureCount": featureCount(cfg),
		"files":        eng.Files(),
		"endpoints":    endpoints,
		"nextSteps": []string{
			fmt.Sprintf("cd %s && go mod tidy", dir),
			"go test ./...",
			"go run .",
		},
	}, nil
}

func (s *Server) validateSchema(args json.RawMessage) (any, *rpcError) {
	if err := emptyArgs(args); err != nil {
		return nil, &rpcError{Code: -32602, Message: err.Error()}
	}
	var input struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid arguments: " + err.Error()}
	}
	valid := true
	var problems []string
	cfg, err := schema.Parse([]byte(input.Schema))
	if err != nil {
		valid = false
		problems = append(problems, err.Error())
	} else {
		problems = append(problems, fmt.Sprintf("valid: %d model(s), database=%s", len(cfg.Models), cfg.Database))
	}
	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": strings.Join(problems, "\n"),
			},
		},
		"valid":    valid,
		"problems": problems,
	}, nil
}

func (s *Server) getSchemaTemplate(args json.RawMessage) (any, *rpcError) {
	var input struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(args, &input)
	name := strings.ToLower(input.Name)

	content, err := examples.Get(name)
	if err != nil {
		return map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Unknown template %q. Available: blog, ecommerce, saas, social", name),
				},
			},
			"isError": true,
		}, nil
	}
	tmpl := string(content)

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": tmpl,
			},
		},
		"template": name,
		"schema":   tmpl,
	}, nil
}

func (s *Server) listFieldTypes() (any, *rpcError) {
	info := `Supported Field Types:
- string: VARCHAR(255) / string
- text: TEXT / string
- int: INT / int
- int64: BIGINT / int64
- float: DOUBLE PRECISION / float64
- bool: BOOLEAN / bool
- time: TIMESTAMP / time.Time
- uuid: VARCHAR(255) / uuid.UUID
- json: JSON / map[string]interface{}
- enum: VARCHAR(255) / string (requires values: [...])
- password: VARCHAR(255) / string (bcrypt hashed)

Supported Field Options:
- required: NOT NULL
- unique: UNIQUE / uniqueIndex
- omitempty: json tag gains ,omitempty
- sensitive: json tag becomes json:"-"
- index: gorm:"index"

Supported Databases:
- postgres (GORM)
- mysql (GORM)
- mongodb (mgm)

Supported Relationships:
- belongsTo (adds foreign key + association)
- hasMany (adds association slice)
- manyToMany (adds join table + association slice)`

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": info,
			},
		},
	}, nil
}

func (s *Server) explainProject(args json.RawMessage) (any, *rpcError) {
	if err := emptyArgs(args); err != nil {
		return nil, &rpcError{Code: -32602, Message: err.Error()}
	}
	var input struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid arguments: " + err.Error()}
	}

	cfg, err := schema.Parse([]byte(input.Schema))
	if err != nil {
		return map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Invalid schema: " + err.Error(),
				},
			},
			"isError": true,
		}, nil
	}

	eng := engine.New(cfg)
	files := eng.Files()

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Project Plan for %s:\n", cfg.AppName))
	sb.WriteString(fmt.Sprintf("Database: %s\n", cfg.Database))
	sb.WriteString(fmt.Sprintf("Models (%d):\n", len(cfg.Models)))
	for _, m := range cfg.Models {
		sb.WriteString(fmt.Sprintf("  - %s (/api/%s) [%d fields]\n", m.Name, m.Endpoint, len(m.Fields)))
	}
	sb.WriteString(fmt.Sprintf("Generated Files (%d):\n", len(files)))
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": sb.String(),
			},
		},
		"files":      files,
		"appName":    cfg.AppName,
		"database":   cfg.Database,
		"modelCount": len(cfg.Models),
	}, nil
}

func (s *Server) addModel(args json.RawMessage) (any, *rpcError) {
	var input struct {
		TargetDir string `json:"targetDir"`
		Model     string `json:"model"`
		DryRun    bool   `json:"dryRun"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid arguments: " + err.Error()}
	}
	if input.Model == "" {
		return nil, &rpcError{Code: -32602, Message: "argument 'model' is required (YAML or JSON string)"}
	}
	targetDir := input.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	var m schema.Model
	if err := yaml.Unmarshal([]byte(input.Model), &m); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid model syntax: " + err.Error()}
	}
	if m.Name == "" {
		return nil, &rpcError{Code: -32602, Message: "model 'name' is required"}
	}

	files, err := engine.AddModelWithOptions(targetDir, m, input.DryRun)
	if err != nil {
		return map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Failed to add model: " + err.Error(),
				},
			},
			"isError": true,
		}, nil
	}

	textMsg := fmt.Sprintf("Model %q successfully added to %s. Model struct, CRUD service, Fiber controller, DB migration, and route registration were updated.", m.Name, targetDir)
	if input.DryRun {
		textMsg = fmt.Sprintf("Dry-run preview for adding model %q to %s:\nFiles to be generated:\n  - %s", m.Name, targetDir, strings.Join(files, "\n  - "))
	}

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": textMsg,
			},
		},
		"modelName": m.Name,
		"targetDir": targetDir,
		"dryRun":    input.DryRun,
		"files":     files,
	}, nil
}

func (s *Server) listModules() (any, *rpcError) {
	mods := modules.List()
	sb := strings.Builder{}
	sb.WriteString("Available Feature Modules:\n")
	for _, m := range mods {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s [%d model(s)]\n", m.Name, m.Category, m.Description, len(m.Models)))
	}

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": sb.String(),
			},
		},
		"modules": mods,
	}, nil
}

func (s *Server) applyModule(args json.RawMessage) (any, *rpcError) {
	var input struct {
		TargetDir string `json:"targetDir"`
		Module    string `json:"module"`
		DryRun    bool   `json:"dryRun"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid arguments: " + err.Error()}
	}
	if input.Module == "" {
		return nil, &rpcError{Code: -32602, Message: "argument 'module' is required"}
	}
	targetDir := input.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	added, files, err := modules.ApplyWithOptions(targetDir, input.Module, input.DryRun)
	if err != nil {
		return map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Failed to apply module: " + err.Error(),
				},
			},
			"isError": true,
		}, nil
	}

	textMsg := fmt.Sprintf("Module %q successfully applied to %s. Added models: %s", input.Module, targetDir, strings.Join(added, ", "))
	if input.DryRun {
		textMsg = fmt.Sprintf("Dry-run preview for applying module %q to %s:\nModels to be added: %s\nFiles to be generated:\n  - %s", input.Module, targetDir, strings.Join(added, ", "), strings.Join(files, "\n  - "))
	}

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": textMsg,
			},
		},
		"module":      input.Module,
		"addedModels": added,
		"targetDir":   targetDir,
		"dryRun":      input.DryRun,
		"files":       files,
	}, nil
}

func featureCount(cfg *schema.Config) int {
	f := cfg.Features
	n := 0
	for _, b := range []bool{f.Auth, f.Docker, f.Migrations, f.Swagger, f.RateLimit, f.CORS, f.Logging, f.Testing, f.CI} {
		if b {
			n++
		}
	}
	return n
}

func (s *Server) write(resp rpcResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = s.out.Write(append(data, '\n'))
}

// --- MCP Resources ---

type resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

func listResources() []resource {
	return []resource{
		{
			URI:         "fiberforge://templates/blog",
			Name:        "Blog Template",
			Description: "Pre-built schema for a blog with posts, tags, and JWT auth (PostgreSQL).",
			MimeType:    "text/yaml",
		},
		{
			URI:         "fiberforge://templates/ecommerce",
			Name:        "E-Commerce Template",
			Description: "Pre-built schema for products, categories, orders, and payments (PostgreSQL).",
			MimeType:    "text/yaml",
		},
		{
			URI:         "fiberforge://templates/saas",
			Name:        "SaaS Template",
			Description: "Pre-built schema for multi-tenant SaaS with organizations and subscriptions (PostgreSQL).",
			MimeType:    "text/yaml",
		},
		{
			URI:         "fiberforge://templates/social",
			Name:        "Social Template",
			Description: "Pre-built schema for a social feed with posts and comments (MongoDB).",
			MimeType:    "text/yaml",
		},
		{
			URI:         "fiberforge://reference/field-types",
			Name:        "Field Types Reference",
			Description: "All supported field types, field options, database drivers, and relationship kinds.",
			MimeType:    "text/plain",
		},
	}
}

func (s *Server) readResource(params json.RawMessage) (any, *rpcError) {
	var p struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
	}

	switch p.URI {
	case "fiberforge://templates/blog",
		"fiberforge://templates/ecommerce",
		"fiberforge://templates/saas",
		"fiberforge://templates/social":
		// Extract template name from URI.
		name := p.URI[len("fiberforge://templates/"):]
		content, err := examples.Get(name)
		if err != nil {
			return nil, &rpcError{Code: -32602, Message: "unknown template: " + name}
		}
		return map[string]any{
			"contents": []any{
				map[string]any{
					"uri":      p.URI,
					"mimeType": "text/yaml",
					"text":     string(content),
				},
			},
		}, nil

	case "fiberforge://reference/field-types":
		// Reuse the same info text from listFieldTypes.
		result, _ := s.listFieldTypes()
		info := result.(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
		return map[string]any{
			"contents": []any{
				map[string]any{
					"uri":      p.URI,
					"mimeType": "text/plain",
					"text":     info,
				},
			},
		}, nil

	default:
		return nil, &rpcError{Code: -32602, Message: "unknown resource URI: " + p.URI}
	}
}
