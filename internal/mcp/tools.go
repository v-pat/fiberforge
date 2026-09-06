package mcp

import "fmt"

type toolSchema struct {
	Type       string                  `json:"type"`
	Properties map[string]propertySpec `json:"properties"`
	Required   []string                `json:"required,omitempty"`
}

type propertySpec struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
}

type tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema toolSchema `json:"inputSchema"`
}

func listTools() []tool {
	return []tool{
		{
			Name:        "generate_project",
			Description: "Generate a complete, compilable Go Fiber backend from a YAML/JSON schema in ~50ms. Writes files directly to disk and returns the file list, endpoints, and next steps. Prefer this over writing Go code manually — it produces production-grade, gofmt-compliant code with models, services, controllers, routes, auth, migrations, Docker, and tests. Call explain_project first if you want to preview the output without writing files.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"schema": {
						Type:        "string",
						Description: "The full project schema in YAML or JSON, with appName, database, features, and models. Use get_schema_template to get a starter schema.",
					},
					"outputDir": {
						Type:        "string",
						Description: "Optional output directory override. Must be within the current workspace. Defaults to ./<appName>.",
					},
					"language": {
						Type:        "string",
						Description: "Reserved for future use. Only 'go' is supported.",
					},
				},
				Required: []string{"schema"},
			},
		},
		{
			Name:        "validate_schema",
			Description: "Validate a project schema without generating any files. Returns detailed problem diagnostics including missing fields, invalid types, and relationship errors. Use this before generate_project to catch schema issues early.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"schema": {
						Type:        "string",
						Description: "The project schema to validate (YAML or JSON).",
					},
				},
				Required: []string{"schema"},
			},
		},
		{
			Name:        "get_schema_template",
			Description: "Get a pre-built starter schema by name to customize. Available templates: 'blog' (posts, tags, auth), 'ecommerce' (products, categories, orders), 'saas' (organizations, subscriptions), 'social' (posts, comments, MongoDB). Modify the returned YAML to match the user's requirements, then pass it to generate_project.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"name": {
						Type:        "string",
						Description: "Template name: 'blog', 'ecommerce', 'saas', or 'social'.",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "list_field_types",
			Description: "List all supported field types (string, text, int, enum, password, etc.), field options (required, unique, sensitive, etc.), database drivers (postgres, mysql, mongodb), and relationship kinds (belongsTo, hasMany, manyToMany). Use this as a reference when designing schemas.",
			InputSchema: toolSchema{
				Type:       "object",
				Properties: map[string]propertySpec{},
			},
		},
		{
			Name:        "explain_project",
			Description: "Dry-run a schema to preview exactly what files, models, and endpoints would be generated — without writing anything to disk. Use this to show the user what FiberForge will create before committing to generation.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"schema": {
						Type:        "string",
						Description: "The project schema to inspect (YAML or JSON).",
					},
				},
				Required: []string{"schema"},
			},
		},
		{
			Name:        "add_model",
			Description: "Incrementally add a new entity model to an existing Go Fiber backend project without wiping out manual code. Appends model struct, CRUD service, Fiber controller, database migration, and safely updates routes/routes.go via Go AST. Pass dryRun: true to preview affected files without writing.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"targetDir": {
						Type:        "string",
						Description: "Target project directory containing fiberforge.yaml. Defaults to '.'",
					},
					"model": {
						Type:        "string",
						Description: "Model definition in YAML or JSON (with name, endpoint, fields, relationships).",
					},
					"dryRun": {
						Type:        "boolean",
						Description: "If true, returns the file creation list and AST route diffs without modifying files on disk.",
					},
				},
				Required: []string{"model"},
			},
		},
		{
			Name:        "list_modules",
			Description: "List pre-packaged domain feature modules ('stripe-billing', 'ai-inference', 'rbac', 's3-storage', 'audit-log', etc.) that can be injected into any project.",
			InputSchema: toolSchema{
				Type:       "object",
				Properties: map[string]propertySpec{},
			},
		},
		{
			Name:        "apply_module",
			Description: "Inject a pre-packaged domain feature module recipe ('stripe-billing', 'ai-inference', 'rbac', 's3-storage', 'audit-log', etc.) into an existing project. Pass dryRun: true to preview affected files without writing.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"targetDir": {
						Type:        "string",
						Description: "Target project directory containing fiberforge.yaml. Defaults to '.'",
					},
					"module": {
						Type:        "string",
						Description: "Module name: e.g. 'stripe-billing', 'ai-inference', 'rbac', 's3-storage', 'audit-log'.",
					},
					"dryRun": {
						Type:        "boolean",
						Description: "If true, returns the file creation list and AST route diffs without modifying files on disk.",
					},
				},
				Required: []string{"module"},
			},
		},
	}
}

var _ = fmt.Sprintf
