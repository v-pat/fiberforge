package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/v-pat/fiberforge/examples"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
)

// Server is a minimal MCP server exposing FiberForge tools over stdio using
// JSON-RPC 2.0. It lets an AI agent generate projects by providing a schema.
//
// The server is provider-agnostic: it never calls an LLM itself. The agent
// (which already has model access) calls the tools; the engine keeps code
// generation deterministic.
type Server struct {
	in  io.Reader
	out io.Writer
}

// NewServer creates an MCP server reading from in and writing to out.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{in: in, out: out}
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
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{"name": "fiberforge", "version": "1.0.0"},
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
		result = map[string]any{"resources": []any{}}
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

	return map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": fmt.Sprintf("Project generated successfully at %s. Set %s in your environment to use it, then run `go run .`.", dir, cfg.AppName),
			},
		},
		"outputDir":    dir,
		"appName":      cfg.AppName,
		"database":     cfg.Database,
		"modelCount":   len(cfg.Models),
		"featureCount": featureCount(cfg),
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
