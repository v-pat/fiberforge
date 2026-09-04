package templates

// SwaggerJSONTemplate renders a basic OpenAPI 3.0 document describing all
// generated endpoints.
const SwaggerJSONTemplate = `{
  "openapi": "3.0.0",
  "info": {
    "title": "{{.AppName}}",
    "version": "1.0.0"
  },
  "paths": {
{{if .Auth}}    "/api/auth/register": {
      "post": {
        "summary": "Register a new user",
        "responses": { "201": { "description": "Created" } }
      }
    },
    "/api/auth/login": {
      "post": {
        "summary": "Login user",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/auth/me": {
      "get": {
        "summary": "Get current user profile",
        "responses": { "200": { "description": "OK" } }
      }
    },
    "/api/auth/refresh": {
      "post": {
        "summary": "Refresh access token",
        "responses": { "200": { "description": "OK" } }
      }
    },
{{end}}{{range .Models}}    "/api/{{.Endpoint}}": {
      "get": {
        "summary": "List {{.Name}}s",
        "responses": { "200": { "description": "OK" } }
      },
      "post": {
        "summary": "Create {{.Name}}",
        "requestBody": { "content": { "application/json": { "schema": { "$ref": "#/components/schemas/{{.Name}}" } } } },
        "responses": { "201": { "description": "Created" } }
      }
    },
    "/api/{{.Endpoint}}/{id}": {
      "get": {
        "summary": "Get {{.Name}} by id",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "OK" } }
      },
      "put": {
        "summary": "Update {{.Name}}",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "OK" } }
      },
      "delete": {
        "summary": "Delete {{.Name}}",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "OK" } }
      }
    }{{if not .Last}},{{end}}
{{end}}  },
  "components": {
    "schemas": {
{{range .Models}}      "{{.Name}}": {
        "type": "object",
        "properties": {
          "id": { "type": "string" }
        }
      }{{if not .Last}},{{end}}
{{end}}    }
  }
}
`
