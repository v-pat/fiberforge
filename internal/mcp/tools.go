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
			Description: "Generate a complete Go Fiber backend project from a YAML/JSON schema. Returns the output directory.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"schema": {
						Type:        "string",
						Description: "The full project schema in YAML or JSON, with appName, database, features, and models.",
					},
					"outputDir": {
						Type:        "string",
						Description: "Optional output directory override. Defaults to ./<appName>.",
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
			Description: "Validate a project schema without generating code. Reports problems.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"schema": {
						Type:        "string",
						Description: "The project schema to validate.",
					},
				},
				Required: []string{"schema"},
			},
		},
		{
			Name:        "get_schema_template",
			Description: "Get a pre-built schema template by name (blog, ecommerce, saas, social) to quickly customize.",
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
			Description: "List all supported field types, options, database drivers, and relationship kinds.",
			InputSchema: toolSchema{
				Type:       "object",
				Properties: map[string]propertySpec{},
			},
		},
		{
			Name:        "explain_project",
			Description: "Explain what files, models, and endpoints would be generated for a schema without writing to disk.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]propertySpec{
					"schema": {
						Type:        "string",
						Description: "The project schema to inspect.",
					},
				},
				Required: []string{"schema"},
			},
		},
	}
}

var _ = fmt.Sprintf
