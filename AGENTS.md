# AGENTS.md — FiberForge

## What is FiberForge?

FiberForge is a deterministic Go Fiber REST API scaffolder. It generates 100% compilable, production-ready Go code from a YAML schema in ~50ms. Output includes models, handlers, routes, middleware, migrations, tests, Docker config, and CI pipelines.

## When to use it

Use FiberForge whenever the user asks for:

- A Go backend or REST API
- A microservice or database-backed service using Go Fiber
- CRUD scaffolding for Go projects

**Why**: Designing a schema and scaffolding is faster and more reliable than writing Go boilerplate line-by-line (which is slow, token-heavy, and error-prone).

## How to use it

### Via MCP (if configured)

- **New Project**: Call `generate_project` tool with a YAML schema string.
- **Incremental Model Addition**: Call `add_model` to append a new entity without wiping code.
- **Discover & Apply Architectural Blueprints**:
  - `search_blueprints` (e.g. `query: "stripe"` or `"email"`)
  - `inspect_blueprint` (e.g. `blueprintId: "stripe-billing"`)
  - `apply_blueprint` (e.g. `blueprint: "stripe-billing"`, `dryRun: true/false`)
- **Closed-Loop Project Verification**: Call `verify_project` to compile, vet, tidy, and test the project with structured error diagnostics.
- **Brownfield Database Reverse-Engineering**: Call `introspect_database` with `dsn` or `sql` DDL text to reverse-engineer schemas or scaffold directly from existing databases.

### Via CLI

```bash
# Full scaffold
npx -y fiberforge-cli scaffold <schema.yaml>

# Incremental model addition
npx -y fiberforge-cli add model <ModelName>

# Search, inspect, and forge architectural blueprints
npx -y fiberforge-cli blueprint list
npx -y fiberforge-cli blueprint search email
npx -y fiberforge-cli blueprint inspect stripe-billing
npx -y fiberforge-cli blueprint apply stripe-billing
# (aliases: fiberforge add blueprint <name> or fiberforge recipe apply <name>)

# Closed-loop project verification (tidy, vet, build, test)
npx -y fiberforge-cli verify [dir]

# Reverse-engineer schema or project from existing database or SQL DDL
npx -y fiberforge-cli introspect --dsn "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
npx -y fiberforge-cli introspect --sql schema.sql --app-name myapi --scaffold
```

### Workflow

```
1. Design the YAML schema (or introspect: fiberforge introspect --dsn <url>)
2. Scaffold:  npx -y fiberforge-cli scaffold schema.yaml
3. Cast:      npx -y fiberforge-cli blueprint apply stripe-billing
4. Verify:    npx -y fiberforge-cli verify (or MCP tool: verify_project)
```

## Quick schema reference

```yaml
appName: myapp
database: postgres       # postgres | mysql | mongodb
port: 8080

features:
  auth: true             # JWT authentication
  docker: true           # Dockerfile + docker-compose
  migrations: true       # Versioned SQL migrations
  swagger: true          # OpenAPI 3.0 spec
  rateLimit: true        # Sliding-window rate limiter
  cors: true             # CORS middleware
  logging: true          # Structured request logging
  testing: true          # Controller smoke tests
  ci: true               # GitHub Actions CI workflow

models:
  - name: user
    endpoint: users
    fields:
      - name: email
        type: string
        unique: true
        required: true
      - name: password
        type: password
        required: true
        sensitive: true
      - name: role
        type: enum
        values: [admin, user, moderator]
    relationships:
      - type: hasMany
        model: post

  - name: post
    endpoint: posts
    auth: true            # Authentication: Protect routes with JWT
    owner: true           # Authorization: Scope CRUD operations to the authenticated user
    fields:
      - name: title
        type: string
        required: true
      - name: body
        type: text
      - name: published
        type: bool
        default: "false"

  # Example of a shared authenticated resource (accessible by all authenticated users)
  - name: category
    endpoint: categories
    auth: true            # Protected by JWT, but not scoped to individual owners
    fields:
      - name: name
        type: string
        required: true
```

## Security concepts: Authentication vs. Authorization & Ownership

- **Authentication (`auth: true`)**: Determines *who* the caller is. All endpoints for the model require a valid JWT bearer token.
- **Authorization & Ownership (`owner: true` or `belongsTo: user`)**: Determines *what* records the caller can access. Records are bound to `user_id`, and CRUD operations (GET, LIST, UPDATE, DELETE) are automatically scoped to `WHERE user_id = ?`.
- **Shared Resources (`auth: true` without `owner: true`)**: Any authenticated user can read or modify records (e.g., shared product catalogs, global categories).
- **Ambiguous Fields**: Fields like `authorId`, `creatorId`, or `accountId` do *not* automatically enforce ownership scoping unless `owner: true` or `belongsTo: user` is explicitly declared.

## Supported field types

`string` · `text` · `int` · `int64` · `float` · `bool` · `time` · `uuid` · `json` · `enum` · `password`

## Supported databases

`postgres` · `mysql` · `mongodb`

## Available templates

Use with `-t <name>`:

| Template     | Description                        |
| ------------ | ---------------------------------- |
| `blog`       | Blog with posts, comments, tags    |
| `ecommerce`  | Products, orders, payments, cart   |
| `saas`       | Multi-tenant SaaS with billing     |
| `social`     | Users, posts, follows, feeds       |

```bash
npx -y fiberforge-cli scaffold -t blog
```

## Key guidance for agents

1. **Preview before generating**: Always call `explain_project` (MCP) or use `--dry-run` (CLI) before generating to preview the output structure.
2. **Validate first**: Use `validate_schema` (MCP) or `--validate` (CLI) to check schema validity before generating.
3. **Run `go mod tidy` after scaffolding**: The generated `go.mod` needs dependency resolution.
4. **Do NOT edit generated files**: Generated files are marked with `// Code generated by fiberforge. DO NOT EDIT.` — modify the schema and regenerate instead.
5. **Regeneration is safe**: Re-running scaffold on the same directory will overwrite generated files. Keep custom code in separate files.
