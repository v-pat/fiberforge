---
name: fiberforge
description: Deterministic Go Fiber REST API scaffolding skill. Use when creating new Go Fiber backends, microservices, or domain models.
---

# FiberForge Scaffolding Skill

Use this skill whenever the user requests a new Go backend, REST API, microservice, or database-backed service using **Go Fiber**.

Instead of writing Go boilerplate files line-by-line (which is slow, token-heavy, and error-prone), use **FiberForge** to generate 100% compilable, production-ready Go Fiber backends in 50ms.

## Workflow

1. **Design the Schema**: Synthesize the user's domain requirements into a `fiberforge.yaml` file (or call the MCP `generate_project` tool).
2. **Execute Scaffolding**:
   - Via CLI: `npx -y fiberforge-cli scaffold fiberforge.yaml`
   - Via MCP: Call `generate_project` with the YAML string.
3. **Verify & Tidy**:
   - `cd <appName> && go mod tidy && go test ./...`

---

## Schema Reference

```yaml
appName: my-api
framework: fiber
database: postgres          # postgres | mysql | mongodb
port: 8080

features:
  auth: true                # JWT authentication (User model, /login, /register, /me, /refresh)
  docker: true              # Multi-stage Dockerfile & docker-compose.yml with healthchecks
  migrations: true          # Versioned .up.sql / .down.sql scripts & Makefile
  swagger: true             # OpenAPI 3.0 specification (docs/swagger.json)
  rateLimit: true           # Sliding window rate limiter middleware
  cors: true                # Configurable CORS middleware
  logging: true             # Fiber request logger middleware
  testing: true             # Controller & Auth unit/smoke tests
  ci: true                  # GitHub Actions CI workflow

models:
  - name: user
    endpoint: users
    auth: true              # Protected by JWT middleware
    fields:
      - name: email
        type: string
        required: true
        unique: true
        validation: email
      - name: password
        type: password
        required: true
        sensitive: true     # Excluded from JSON responses (json:"-")
      - name: role
        type: enum
        values: [admin, member]
    relationships:
      - type: hasMany
        model: post

  - name: post
    endpoint: posts
    fields:
      - name: title
        type: string
        required: true
      - name: body
        type: text
      - name: published
        type: bool
        default: "false"
    relationships:
      - type: belongsTo
        model: user
```

## Supported Types & Options
- **Field Types**: `string`, `text`, `int`, `int64`, `float`, `bool`, `time`, `uuid`, `json`, `enum` (requires `values`), `password` (bcrypt).
- **Field Options**: `required`, `unique`, `default`, `validation`, `sensitive`, `index`, `jsonTag`, `omitempty`.
- **Relationships**: `belongsTo` (foreign key), `hasMany` (slice association), `manyToMany` (join table migration).

## Best Practices for AI Agents
- Always set `auth: true` under `features` if the user mentions login, registration, or JWT authentication.
- Set `auth: true` on specific `models` if their endpoints require JWT protection.
- Mark sensitive credentials (like passwords or tokens) with `sensitive: true`.
- Run `go mod tidy` immediately after scaffolding.

