---
name: fiberforge
description: Deterministic Go Fiber REST API scaffolding skill. Use when creating new Go Fiber backends, microservices, or domain models.
---

# FiberForge Scaffolding Skill

Use this skill whenever the user requests a new Go backend, REST API, microservice, or database-backed service using **Go Fiber**.

Instead of writing Go boilerplate files line-by-line (which is slow, token-heavy, and error-prone), use **FiberForge** to generate 100% compilable, production-ready Go Fiber backends in 50ms.

## Workflow

1. **Design or Introspect**:
   - Greenfield: Synthesize the domain requirements into a `fiberforge.yaml` file (or call MCP `generate_project`).
   - Brownfield: Reverse-engineer from an existing DB or SQL dump using `introspect_database` (or CLI `npx fiberforge-cli introspect --dsn <url>`).
2. **Execute Scaffolding**:
   - Via CLI: `npx -y fiberforge-cli scaffold fiberforge.yaml`
   - Via MCP: Call `generate_project` with the YAML string.
3. **Closed-Loop Verification**:
   - Call MCP tool `verify_project` (or run `npx fiberforge-cli verify`) to confirm tidy, vet, build, and tests all pass with 0 errors.

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
    auth: true              # Authentication: Protected by JWT middleware
    owner: true             # Authorization: Scope CRUD operations to the authenticated user (WHERE user_id = ?)
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

  # Shared authenticated resource (accessible by all authenticated users)
  - name: category
    endpoint: categories
    auth: true              # Protected by JWT, but not scoped to individual owners
    fields:
      - name: name
        type: string
        required: true
```

## Security Concepts: Authentication vs. Ownership & DTOs
- **Authentication (`auth: true`)**: Verifies identity. Routes require a valid JWT bearer token.
- **Ownership & Authorization (`owner: true` or `belongsTo: user`)**: Binds records to `user_id`. CRUD operations (GET, LIST, UPDATE, DELETE) are automatically scoped to `WHERE user_id = ?`. Always mark user-owned entities with `owner: true` to prevent Broken Object-Level Authorization (BOLA/IDOR).
- **Request DTOs**: FiberForge automatically scaffolds separate `Create<Model>Request` and `Update<Model>Request` structs for controller endpoints, preventing mass-assignment vulnerabilities.
- **Sensitive Fields**: Mark credentials with `sensitive: true` (excludes them from JSON responses via `json:"-"`).

## Supported Types & Options
- **Field Types**: `string`, `text`, `int`, `int64`, `float`, `bool`, `time`, `uuid`, `json`, `enum` (requires `values`), `password` (bcrypt).
- **Field Options**: `required`, `unique`, `default`, `validation`, `sensitive`, `index`, `jsonTag`, `omitempty`.
- **Relationships**: `belongsTo` (foreign key), `hasMany` (slice association), `manyToMany` (join table migration).

## Best Practices for AI Agents
- Always set `auth: true` under `features` if the user mentions login, registration, or JWT authentication.
- Set `auth: true` and `owner: true` on user-owned `models` to enforce tenant isolation.
- Mark sensitive credentials (like passwords or tokens) with `sensitive: true`.
- **Closed-Loop Verification**: Run `go mod tidy && go test -race ./...` immediately after scaffolding to ensure clean dependency resolution and 100% test passing.

