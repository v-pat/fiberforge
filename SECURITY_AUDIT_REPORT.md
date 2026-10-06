# FiberForge Comprehensive Security Audit Report

**Date:** 2026-10-04  
**Target:** FiberForge (`github.com/v-pat/fiberforge`)  
**Auditor:** DeepMind Antigravity Advanced Agentic Security Team  
**Scope:** Full repository codebase, CLI tool (`cmd/`), MCP server (`internal/mcp/`), schema validation engine (`internal/schema/`), code generation pipeline (`internal/engine/`), template system (`internal/templates/`), module recipes (`internal/modules/`), npm wrapper (`npm/fiberforge/`), CI/CD workflows (`.github/workflows/`), and generated application security architecture.

---

## Executive Summary

| Metric | Result |
|---|---|
| **Overall Risk Level** | **CRITICAL** |
| **Critical Findings** | **5** |
| **High Findings** | **5** |
| **Medium Findings** | **6** |
| **Low Findings** | **3** |
| **Total Findings** | **19** |

FiberForge is designed as a developer-facing tool and Model Context Protocol (MCP) server that transforms YAML/JSON schemas into production-ready Go Fiber REST APIs. Because FiberForge is intended to be invoked by automated AI coding agents and CLI users with untrusted or externally influenced schemas, its parsing, validation, and generation boundaries must be strictly defended.

The audit revealed that **FiberForge contains multiple critical security vulnerabilities allowing Arbitrary File Overwrite / Path Traversal outside the workspace, Direct Remote Code Execution (RCE) / Arbitrary Go Code Injection into generated projects, SQL Injection in database migration files, and complete Authentication Bypass via hardcoded secrets in generated applications.**

Furthermore, the npm package distribution path relies on unverified binary downloads with no cryptographic hash verification, and the repository contains checked-in, pre-compiled Mach-O binary executables.

---

## Critical Findings

### [CRIT-01] Path Traversal and Arbitrary File Overwrite Outside Workspace via `appName` (CLI & MCP Bypass)

**Severity:** Critical  
**Confidence:** High  
**Location:** [`internal/engine/engine.go:24-29`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/engine.go#L24-L29), [`internal/mcp/server.go:184-205`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/mcp/server.go#L184-L205), [`cmd/scaffold.go:51-54`](file:///Users/vaibhavpathak/Documents/fiberforge/cmd/scaffold.go#L51-L54)  
**Category:** CWE-22: Improper Limitation of a Pathname to a Restricted Directory ('Path Traversal')

#### Attack Scenario
An attacker crafts a schema where `appName` is a relative path containing directory traversal sequences (e.g. `appName: "../../evil"` or `appName: "../../../.ssh"`). 
1. When executed via CLI (`fiberforge scaffold schema.yaml`), no path validation is performed.
2. When executed via the MCP server (`tools/call` -> `generate_project`), the server performs a workspace containment check **only if `outputDir` is explicitly passed in the JSON request**:
   ```go
   if cfg.OutputDir != "" {
       absOut, err := filepath.Abs(cfg.OutputDir)
       // ... checks if absOut is within cwd ...
   }
   ```
   If the caller omits `outputDir`, `cfg.OutputDir` remains empty (`""`). The containment check is bypassed completely. `engine.New(cfg)` then falls back to `dir = "./" + cfg.AppName`, resolving to the traversal path.
3. `engine.Generate()` calls `os.MkdirAll` and `os.WriteFile` on files like `main.go`, `go.mod`, `.env.example`, overwriting arbitrary files outside the developer's workspace root.

#### Root Cause
1. `cfg.AppName` is never validated with an identifier regular expression or path sanitizer in [`internal/schema/parse.go`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/schema/parse.go#L63-L65). Only non-emptiness is checked.
2. The MCP server's workspace confinement guard checks `cfg.OutputDir` instead of the resolved effective target directory `eng.Dir()`.
3. The CLI [`cmd/scaffold.go`](file:///Users/vaibhavpathak/Documents/fiberforge/cmd/scaffold.go) contains zero path containment checks.

#### Impact
Arbitrary file write and directory creation anywhere on the developer or AI agent host system where the executing process has write permissions, potentially overwriting shell profiles (`~/.bashrc`, `~/.zshrc`), authorized keys, or critical source files.

#### Proof
```yaml
# /tmp/fiberforge-security-audit/schema_traversal.yaml
appName: "../../fiberforge-mcp-escape"
database: postgres
models:
  - name: user
    endpoint: users
    fields:
      - name: email
        type: string
        required: true
      - name: password
        type: password
        required: true
```
Submitting via MCP JSON-RPC without `outputDir`:
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"generate_project","arguments":{"schema":"appName: \"../../fiberforge-mcp-escape\"\ndatabase: postgres\nmodels:\n  - name: user\n    endpoint: users\n    fields:\n      - name: email\n        type: string\n        required: true\n      - name: password\n        type: password\n        required: true\n"}}}
```
**Observed Execution Result:** The MCP server generates the complete project under `./../../fiberforge-mcp-escape`, completely outside the repository root.

#### Why Existing Validation Does Not Stop It
[`internal/schema/parse.go:106`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/schema/parse.go#L106) declares `identRe := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")`, but only validates `m.Name` and `f.Name`. `cfg.AppName` is never checked against `identRe`.

#### Recommended Fix
1. Validate `cfg.AppName` in `schema.Validate`:
   ```go
   appNameRe := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
   if !appNameRe.MatchString(cfg.AppName) {
       return fmt.Errorf("appName %q contains invalid characters (must match %s)", cfg.AppName, appNameRe.String())
   }
   ```
2. In both CLI and MCP, canonicalize and verify the final target path using `filepath.Abs` and `filepath.EvalSymlinks`, asserting that it resides strictly within the intended working directory:
   ```go
   targetDir := eng.Dir()
   absTarget, err := filepath.Abs(targetDir)
   if err != nil || !strings.HasPrefix(absTarget, cwd + string(filepath.Separator)) {
       return fmt.Errorf("refusing to write to %q outside workspace root %q", targetDir, cwd)
   }
   ```

#### Regression Test
Create a test in `internal/mcp/server_test.go` and `cmd/scaffold_test.go` asserting that `appName: "../../test"` returns an error and does not touch the filesystem.

---

### [CRIT-02] Arbitrary Go Code Injection / Remote Code Execution via Model `tableName`

**Severity:** Critical  
**Confidence:** High  
**Location:** [`internal/templates/model.go:24-26`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/model.go#L24-L26), [`internal/templates/model.go:46-48`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/model.go#L46-L48), [`internal/schema/parse.go:105-144`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/schema/parse.go#L105-L144)  
**Category:** CWE-94: Improper Control of Generation of Code ('Code Injection')

#### Attack Scenario
An attacker supplies a schema defining a model with a crafted `tableName` attribute:
```yaml
models:
  - name: User
    endpoint: users
    tableName: "users\" }\n\nfunc init() { panic(\"PWNED_TABLE_NAME\") }\n\nfunc (User) Dummy() string { return \""
    fields:
      - name: email
        type: string
        required: true
      - name: password
        type: password
        required: true
```
FiberForge scaffolds the project into `model/user.go`. When the developer or CI runs `go build`, `go test`, or starts the backend, the injected `init()` function executes arbitrary code immediately.

#### Root Cause
In [`internal/templates/model.go`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/model.go#L24-L26):
```go
{{if .HasTableName}}
// TableName returns the explicit table name.
func ({{.Name}}) TableName() string {
	return "{{.TableName}}"
}
{{end}}
```
`m.TableName` is placed directly inside double quotes in Go code. Because `m.TableName` is never validated or escaped, quotes and newlines escape the string literal and insert top-level Go statements.

#### Impact
Remote Code Execution (RCE) on any developer workstation, build server, or production environment running or testing the generated code.

#### Proof
Generated `model/user.go`:
```go
// TableName returns the explicit table name.
func (User) TableName() string {
	return "users"
}

func init() { panic("PWNED_TABLE_NAME") }

func (User) Dummy() string {
	return ""
}
```
`gofmt` successfully formats this file without error, and `go build ./...` compiles and executes the `init()` backdoor.

#### Why Existing Validation Does Not Stop It
`internal/schema/parse.go` does not inspect `m.TableName`.

#### Recommended Fix
1. Validate `m.TableName` against an identifier regular expression:
   ```go
   if m.TableName != "" && !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(m.TableName) {
       return fmt.Errorf("model %q tableName %q is not a valid identifier", m.Name, m.TableName)
   }
   ```
2. When formatting into Go templates, use `%q` (`strconv.Quote`) rather than `"%s"`.

#### Regression Test
Assert in `internal/schema/parse_test.go` that schemas containing `tableName: "foo\" \n bar"` fail validation.

---

### [CRIT-03] Arbitrary Go Code Injection via AST Route Registration in `add model` and Incremental Updates

**Severity:** Critical  
**Confidence:** High  
**Location:** [`internal/engine/ast.go:51-68`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/ast.go#L51-L68), [`cmd/add.go:40-53`](file:///Users/vaibhavpathak/Documents/fiberforge/cmd/add.go#L40-L53), [`internal/mcp/server.go:411-466`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/mcp/server.go#L411-L466)  
**Category:** CWE-94: Improper Control of Generation of Code ('Code Injection')

#### Attack Scenario
An attacker or an AI agent interacting with the MCP tool `add_model` or CLI `fiberforge add model` passes a crafted endpoint string:
```bash
fiberforge add model EvilItem --dir ./myproject --endpoint 'items"); panic("PWNED_AST"); _ = api.Group("dummy'
```
FiberForge uses `go/parser` to parse this snippet and appends the AST statement directly into the developer's existing `routes/routes.go`.

#### Root Cause
In [`internal/engine/ast.go:51-68`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/ast.go#L51-L68):
```go
	// Construct route group statements Go code snippet
	snippet := fmt.Sprintf(`package dummy

func dummy() {
	%s := api.Group("/%s")
`, varName, path)
```
`path` (which comes from `model.Endpoint`) is concatenated directly into the Go source snippet passed to `parser.ParseFile`. The parser parses the injected code as valid statements inside `dummy()`, and `ast.go` appends all parsed statements to `Routes()`:
```go
	dummyFunc := dummyNode.Decls[0].(*ast.FuncDecl)
	newStmts := dummyFunc.Body.List
	routesFunc.Body.List = append(routesFunc.Body.List, newStmts...)
```

#### Impact
Arbitrary code injection into existing Go code files during incremental model additions.

#### Proof
Inspecting `routes/routes.go` after running the reproduction command:
```go
func Routes(app *fiber.App) {
	api := app.Group("/api")
	_ = api
	evilItemGroup := api.Group("/items")
	panic("PWNED_AST")
	_ = api.Group("dummy")
	evilItemGroup.Post("/", controller.CreateEvilItem)
    // ...
}
```
The injected `panic("PWNED_AST")` is now an active statement in the route registration function.

#### Why Existing Validation Does Not Stop It
`model.Endpoint` is only checked for non-emptiness (`if m.Endpoint == ""`). No syntax, character, or quote check is applied.

#### Recommended Fix
1. Validate `model.Endpoint` against an endpoint regex (e.g. `^[a-zA-Z0-9/_-]+$` with no quotes, semicolons, or newlines).
2. Build AST nodes programmatically using `ast.CallExpr` and `ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("/" + path)}` instead of parsing raw string templates.

#### Regression Test
Add a test in `internal/engine/ast_test.go` attempting to inject Go statements through `model.Endpoint` and verify that an error is returned.

---

### [CRIT-04] Arbitrary Go Code Injection via Struct Field `Validation` and `JSONTag` Attributes

**Severity:** Critical  
**Confidence:** High  
**Location:** [`internal/engine/data.go:144-163`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/data.go#L144-L163), [`internal/schema/parse.go:121-131`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/schema/parse.go#L121-L131)  
**Category:** CWE-94: Improper Control of Generation of Code ('Code Injection')

#### Attack Scenario
An attacker specifies a schema field with a `validation` or `jsonTag` attribute that contains a backtick (`` ` ``) and literal newlines:
```yaml
models:
  - name: user
    endpoint: users
    fields:
      - name: email
        type: string
        validation: |-
          email"`
          }
          func init() { panic("PWNED_STRUCT_TAG") }
          type Dummy struct {
            X string `
        required: true
      - name: password
        type: password
        required: true
```
FiberForge constructs the struct tags by string concatenation with backticks:
```go
return "`" + gorm + jsonTag + valTag + "`"
```
The backtick closes the Go struct tag, and the trailing lines break out of the struct declaration into package-level declarations.

#### Root Cause
Struct tag assembly in `internal/engine/data.go:buildStructTag` wraps user input inside backticks without sanitizing or rejecting backticks (`\x60`) and newlines in `f.Validation` and `f.JSONTag`.

#### Impact
Remote Code Execution via arbitrary Go package injection in `model/*.go`.

#### Proof
Generated `model/user.go`:
```go
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Email     string         `gorm:"not null" json:"email" validate:"required,email"`
}

func init() { panic("PWNED_STRUCT_TAG") }

type Dummy struct {
	X        string `"`
	Password string `gorm:"not null" json:"password" validate:"required"`
}
```
`gofmt` successfully formats this, and the injected `init()` function runs on package initialization.

#### Why Existing Validation Does Not Stop It
`validateField(f Field)` in [`internal/schema/parse.go:158-171`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/schema/parse.go#L158-L171) only validates `f.Type`. It does not inspect `f.Validation` or `f.JSONTag`.

#### Recommended Fix
In `validateField`, validate:
```go
if strings.ContainsAny(f.Validation, "`\r\n\"") {
    return fmt.Errorf("field validation cannot contain backticks, quotes, or newlines")
}
if strings.ContainsAny(f.JSONTag, "`\r\n\"") {
    return fmt.Errorf("field jsonTag cannot contain backticks, quotes, or newlines")
}
```

#### Regression Test
Add test cases in `internal/schema/parse_test.go` ensuring schema parsing rejects fields with backticks in validation and struct tags.

---

### [CRIT-05] SQL Injection into Migration DDL Files via `tableName` and Field `Default`

**Severity:** Critical  
**Confidence:** High  
**Location:** [`internal/engine/data.go:183-219`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/data.go#L183-L219), [`internal/engine/data.go:221-234`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/data.go#L221-L234)  
**Category:** CWE-89: Improper Neutralization of Special Elements used in an SQL Command ('SQL Injection')

#### Attack Scenario
An attacker provides a schema with:
```yaml
models:
  - name: user
    endpoint: users
    tableName: "users; DROP SCHEMA public CASCADE; --"
    fields:
      - name: email
        type: string
        default: "'test'); MALICIOUS SQL STATEMENT; --"
```
When migrations are generated, FiberForge formats the raw strings directly into `.up.sql` and `.down.sql`.

#### Root Cause
In [`internal/engine/data.go:199-201`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/engine/data.go#L199-L201):
```go
up = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n\tid %s PRIMARY KEY%s\n);\n",
    table, idType(driver), colStr)
down = fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", table)
```
and line 231:
```go
if f.Default != nil {
    parts = append(parts, "DEFAULT "+*f.Default)
}
```
Identifiers and default expressions are interpolated directly into SQL statements without escaping, quoting, or validation.

#### Impact
Execution of destructive arbitrary SQL queries when `make migrate` or `golang-migrate` is executed against the database.

#### Proof
Generated `migrations/000001_user.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS users; DROP SCHEMA public CASCADE; -- (
	id BIGSERIAL PRIMARY KEY,
		email VARCHAR(255) NOT NULL DEFAULT 'test'); MALICIOUS SQL STATEMENT; --,
		password VARCHAR(255) NOT NULL
);
```
Generated `migrations/000001_user.down.sql`:
```sql
DROP TABLE IF EXISTS users; DROP SCHEMA public CASCADE; --;
```

#### Why Existing Validation Does Not Stop It
Neither `m.TableName` nor `f.Default` is validated in `internal/schema/parse.go`.

#### Recommended Fix
1. Enforce strict identifier rules on `tableName` (`^[a-zA-Z_][a-zA-Z0-9_]*$`).
2. Quote SQL table names with driver-appropriate quotes (e.g. `"` for PostgreSQL, `` ` `` for MySQL).
3. Validate `f.Default` against permitted literal types (numeric, boolean, or single-quoted string literals with SQL escaping).

#### Regression Test
Add unit tests in `internal/engine/migration_test.go` checking that quotes and semicolons in `tableName` and `Default` are rejected or properly sanitized.

---

## High Findings

### [HIGH-01] Hardcoded JWT Signing Secret Fallback in MongoDB Generated Applications

**Severity:** High  
**Confidence:** High  
**Location:** [`internal/templates/auth.go:429-434`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/auth.go#L429-L434)  
**Category:** CWE-798: Use of Hard-coded Credentials / CWE-321: Cryptographic Weakness

#### Attack Scenario
A developer scaffolds a MongoDB project with JWT authentication (`features.auth: true`, `database: mongodb`). In development, staging, or production where `JWT_SECRET` is unset, the application starts normally without error. An external attacker creates a JWT signed with `"change-me-in-production"` using HS256, sending it in the `Authorization: Bearer <token>` header. The API accepts the token and authenticates the attacker as any user.

#### Root Cause
In `internal/templates/auth.go`:
```go
// Secret returns the JWT signing secret from the environment.
func Secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("change-me-in-production")
}
```
While the SQL implementation was updated in commit `5756f8a` to call `log.Fatal`, the MongoDB template was overlooked and retains the fallback hardcoded secret.

#### Impact
Complete authentication bypass on all protected endpoints for MongoDB deployments.

#### Proof
Starting a generated MongoDB app without setting `JWT_SECRET`: `Secret()` returns `[]byte("change-me-in-production")`. Any JWT token signed with `"change-me-in-production"` and claims `{"sub": "<any-object-id>"}` passes `auth.ValidateToken()`.

#### Why Existing Validation Does Not Stop It
This is a flaw in the generated template code.

#### Recommended Fix
Align `MongoJWTTemplate` with `JWTTemplate`:
```go
func Secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	log.Fatal("JWT_SECRET environment variable is not set")
	return nil
}
```

#### Regression Test
Verify in `internal/engine/matrix_test.go` that the generated `auth/jwt.go` for MongoDB does not contain `"change-me-in-production"`.

---

### [HIGH-02] Broken Object-Level Authorization (BOLA/IDOR) Across All Generated Entity Endpoints

**Severity:** High  
**Confidence:** High  
**Location:** [`internal/templates/controller.go:40-81`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/controller.go#L40-L81), [`internal/templates/routes.go:39-46`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/routes.go#L39-L46)  
**Category:** CWE-639: Authorization Bypass Through User-Controlled Key / OWASP API1:2023

#### Attack Scenario
A multi-user application is generated with `features.auth: true` and models marked `auth: true`. User A logs in and obtains a valid JWT. User A can send `GET /api/items/1`, `PUT /api/items/1`, or `DELETE /api/items/1` for records owned by User B. The controller processes the request and executes the operation.

#### Root Cause
In [`internal/templates/controller.go`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/controller.go):
```go
func Get{{.Name}}ByID(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 64)
    // ...
    m, err := service.Get{{.Name}}ByID(uint(id))
```
The controllers never inspect `c.Locals("userId")`. FiberForge equates *authentication* (valid JWT) with *authorization* (ownership/access rights).

#### Impact
Complete horizontal privilege escalation: any authenticated user can view, edit, or delete data belonging to any other user.

#### Proof
Deploying any generated project with `auth: true` on models: the generated service queries `DB.First(&m, id)` and `DB.Delete(&model{}, id)` without any user ID condition.

#### Recommended Fix
1. Where relationships associate a model with `User`, include tenant/owner filtering in the generated service queries:
   `databases.DB.Where("id = ? AND user_id = ?", id, currentUserID).First(&m)`
2. Document clearly in generated READMEs and API documentation that the generated CRUD handlers are basic prototypes requiring application-specific authorization policies.

---

### [HIGH-03] Mass Assignment Vulnerability in Generated Update Endpoints

**Severity:** High  
**Confidence:** High  
**Location:** [`internal/templates/controller.go:58-69`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/controller.go#L58-L69), [`internal/templates/service.go:40-48`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/service.go#L40-L48), [`internal/templates/service.go:112-123`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/service.go#L112-L123)  
**Category:** CWE-915: Improperly Controlled Modification of Dynamically-Determined Object Attributes

#### Attack Scenario
An API endpoint handles `PUT /api/posts/:id`. A client submits JSON containing protected fields, such as `{"id": 999, "userId": 42, "role": "admin"}`. GORM updates all non-zero fields directly on the model:
```go
databases.DB.Model(&m).Updates(patch)
```
In MongoDB, `mgm.Coll(patch).Update(patch)` replaces the document entirely with the supplied struct.

#### Root Cause
No Data Transfer Objects (DTOs) or field whitelists are generated for create/update operations. The incoming request is unmarshaled directly into the database model.

#### Impact
Unauthorized modification of sensitive model attributes, foreign keys, or ownership markers.

#### Recommended Fix
Generate dedicated request DTOs (e.g. `Create{{.Name}}Request` and `Update{{.Name}}Request`) containing only user-editable fields, rather than binding directly to GORM/MGM database models.

---

### [HIGH-04] NPM Distribution Security: Unverified Binary Download Without Checksum or Hash Verification

**Severity:** High  
**Confidence:** High  
**Location:** [`npm/fiberforge/install.js:32-84`](file:///Users/vaibhavpathak/Documents/fiberforge/npm/fiberforge/install.js#L32-L84)  
**Category:** CWE-353: Missing Support for Integrity Check, CWE-494: Download of Code Without Integrity Check

#### Attack Scenario
A developer installs FiberForge using `npm install -g fiberforge-cli` or runs it via `npx -y fiberforge-cli`. 
1. The `postinstall` script runs [`npm/fiberforge/install.js`](file:///Users/vaibhavpathak/Documents/fiberforge/npm/fiberforge/install.js).
2. It fetches `https://github.com/v-pat/fiberforge/releases/download/v1.0.0/...` over HTTPS.
3. The archive is piped directly to disk and extracted via `tar -xzf` / `unzip`.
4. At no point is a SHA-256 checksum or GPG signature verified against an embedded manifest.
5. If GitHub release assets are compromised or modified, arbitrary executable code is downloaded and executed with the user's privileges.
6. Additionally, [`npm/fiberforge/package.json`](file:///Users/vaibhavpathak/Documents/fiberforge/npm/fiberforge/package.json#L3) is version `1.0.2`, while `install.js` line 7 hardcodes `const version = '1.0.0'`, downloading obsolete binaries.

#### Root Cause
`install.js` does not fetch or verify `checksums.txt` (which GoReleaser produces in `.goreleaser.yaml`).

#### Impact
Remote code execution during package installation via supply-chain tampering.

#### Recommended Fix
1. Embed the expected SHA-256 hashes for each platform release inside `install.js` or download and verify `checksums.txt` against a pinned public key before extracting.
2. Synchronize version numbers between `package.json` and `install.js`.

---

### [HIGH-05] Pre-compiled Mach-O Binary Executables Committed Directly to Git Repository

**Severity:** High  
**Confidence:** High  
**Location:** `fiberforge` (10,111,522 bytes), `npm/fiberforge/bin/fiberforge` (6,913,842 bytes)  
**Category:** CWE-506: Embedded Malicious Code / Supply Chain Hygiene

#### Attack Scenario
Developers or automated environments clone the Git repository and execute `./fiberforge`. The committed binary cannot be audited via text-based git diffs, cannot be verified to match the committed source code, and bypasses local compilation controls.

#### Root Cause
Binaries were manually added to git tracking (`git log` shows commits `ddfcdf4`, `87845e8`, `27a3904`). `.gitignore` only ignores `.env`, `bin/`, `*.exe`, and `coverage.out`, failing to ignore the binary named `fiberforge`.

#### Impact
Supply chain risk, audit evasion, and repo bloat.

#### Recommended Fix
1. Remove `fiberforge` and `npm/fiberforge/bin/fiberforge` from git tracking (`git rm --cached`).
2. Add `/fiberforge` and `npm/fiberforge/bin/` to `.gitignore`.
3. Distribute binaries exclusively via GitHub Releases and signed checksums produced reproducibly by GitHub Actions.

---

## Medium Findings

### [MED-01] Missing Password Complexity and Email Format Validation on Authentication Endpoints

**Severity:** Medium  
**Confidence:** High  
**Location:** [`internal/templates/auth.go:255-258`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/auth.go#L255-L258)  
**Category:** CWE-521: Weak Password Requirements

#### Attack Scenario
An attacker registers an account on `/api/auth/register` with an empty password (`""`) or single-character password and invalid email address. The application hashes the empty string with bcrypt and registers the account.

#### Root Cause
In [`internal/templates/auth.go:255-258`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/auth.go#L255-L258):
```go
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
```
No validation struct tags (`validate:"required,email"`, `validate:"required,min=8"`) are present on `credentials`. Although `ValidateStruct(&req)` is invoked, `validator.Validate` has no rules to check.

#### Impact
Creation of accounts with empty or weak passwords, leading to trivial account compromise and credential stuffing vulnerability.

#### Recommended Fix
Add validator tags to the `credentials` struct:
```go
type credentials struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}
```

---

### [MED-02] Permissive Wildcard CORS Configuration Generated by Default

**Severity:** Medium  
**Confidence:** High  
**Location:** [`internal/templates/main.go:55-58`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/main.go#L55-L58)  
**Category:** CWE-942: Permissive Cross-Origin Resource Sharing Policy

#### Attack Scenario
When `features.cors: true` is configured, FiberForge generates:
```go
app.Use(cors.New(cors.Config{
    AllowOrigins: "*",
    AllowHeaders: "Origin, Content-Type, Accept, Authorization",
}))
```
Any malicious third-party site visited by an authenticated user can make cross-origin requests to the API.

#### Impact
Excessive exposure of APIs to cross-origin abuse.

#### Recommended Fix
Provide an environment variable configuration (e.g. `CORS_ALLOWED_ORIGINS`) with a sensible default like `http://localhost:3000` rather than hardcoding `*`.

---

### [MED-03] Generated Docker Containers Execute as Unprivileged Root

**Severity:** Medium  
**Confidence:** High  
**Location:** [`internal/templates/support.go:4-19`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/support.go#L4-L19)  
**Category:** CWE-250: Execution with Unnecessary Privileges

#### Attack Scenario
A vulnerability in the Go runtime, Go Fiber, or an uploaded file processing handler allows remote code execution. Because the container process runs as UID 0 (`root`), the attacker has full root access within the container namespace.

#### Root Cause
The generated Dockerfile runtime stage is:
```dockerfile
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /bin/{{.AppName}} .
EXPOSE {{.Port}}
ENTRYPOINT ["/app/{{.AppName}}"]
```
It does not create or switch to a non-root user.

#### Impact
Container compromise with full root capabilities.

#### Recommended Fix
Add a non-root user in the Dockerfile template:
```dockerfile
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
```

---

### [MED-04] Hardcoded Development Database Credentials & Secrets in Compose and Config

**Severity:** Medium  
**Confidence:** High  
**Location:** [`internal/templates/support.go:29-37`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/support.go#L29-L37), [`internal/templates/main.go:126-153`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/main.go#L126-L153)  
**Category:** CWE-798: Use of Hard-coded Credentials

#### Attack Scenario
In `docker-compose.yml`, database credentials default to:
`DB_USER: postgres`, `DB_PASSWORD: password`, `JWT_SECRET: change-me-in-production`.
In `config/config.go`, if environment variables are not found at startup, `defaultValue(k)` automatically populates them with these identical defaults. If developers deploy the container without setting environment variables, production instances run with default passwords.

#### Root Cause
`config.Load()` automatically calls `os.Setenv(k, defaultValue(k))` instead of failing startup if production credentials are missing.

#### Recommended Fix
Distinguish development from production environments (`APP_ENV=production`) and refuse to start if secrets and database passwords match default credentials.

---

### [MED-05] Missing Rate Limiting on Authentication Endpoints When `rateLimit` is Disabled

**Severity:** Medium  
**Confidence:** High  
**Location:** [`internal/templates/main.go:59-64`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/main.go#L59-L64)  
**Category:** CWE-307: Improper Restriction of Excessive Authentication Attempts

#### Attack Scenario
A developer generates an application with `features.auth: true` but omits `features.rateLimit: true`. The generated `/api/auth/login` endpoint has zero rate limiting, allowing unlimited brute-force password guessing. Even when `rateLimit: true` is on, the limit is global (60/min) rather than strict on auth routes (e.g. 5/min).

#### Recommended Fix
Always attach dedicated rate limiting to `/api/auth/login` and `/api/auth/register` regardless of the global `rateLimit` flag.

---

### [MED-06] Internal Error and Schema Leakage in HTTP 500 Responses

**Severity:** Medium  
**Confidence:** High  
**Location:** [`internal/templates/controller.go:25,34,47,66,78`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/controller.go#L25), [`internal/templates/routes.go:24`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/templates/routes.go#L24)  
**Category:** CWE-209: Generation of Error Message Containing Sensitive Information

#### Attack Scenario
An attacker sends a payload causing a database constraint violation or connection error. The controller catches the error and executes:
```go
return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
```
The exact GORM / PostgreSQL driver error (including table names, column names, constraints, or connection strings) is reflected in the HTTP response body.

#### Recommended Fix
Log `err.Error()` on the server via `slog.Error` and return an opaque message like `"An unexpected error occurred"` to clients.

---

## Low Findings

### [LOW-01] MCP Process Denial of Service via Oversized Input Lines

**Severity:** Low  
**Confidence:** High  
**Location:** [`internal/mcp/server.go:61-76`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/mcp/server.go#L61-L76)  
**Category:** CWE-400: Uncontrolled Resource Consumption

#### Attack Scenario
An AI agent or caller sends a JSON-RPC message line exceeding 4MB to `fiberforge serve`. `bufio.NewScanner` buffer overflows, `sc.Scan()` returns `false`, and `sc.Err()` returns `bufio.ErrTooLong`. The server terminates immediately.

#### Recommended Fix
Use a streaming `json.Decoder` reading from `s.in` rather than line-based `bufio.Scanner`, allowing arbitrary valid JSON-RPC payloads without arbitrary buffer limits.

---

### [LOW-02] Path Traversal in MCP `add_model` and `apply_module` Target Directories

**Severity:** Low  
**Confidence:** High  
**Location:** [`internal/mcp/server.go:411-466`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/mcp/server.go#L411-L466), [`internal/mcp/server.go:487-536`](file:///Users/vaibhavpathak/Documents/fiberforge/internal/mcp/server.go#L487-L536)  
**Category:** CWE-22: Path Traversal

#### Attack Scenario
While `generate_project` attempted to implement a workspace root guard for `outputDir`, the `add_model` and `apply_module` MCP tools accept `targetDir` without any workspace confinement checks, allowing writes to arbitrary directories.

#### Recommended Fix
Apply the same canonical path validation to `targetDir` in `add_model` and `apply_module`.

---

### [LOW-03] GitHub Actions Workflow Default Broad Permissions

**Severity:** Low  
**Confidence:** High  
**Location:** [`.github/workflows/ci.yml:1-72`](file:///Users/vaibhavpathak/Documents/fiberforge/.github/workflows/ci.yml#L1-L72)  
**Category:** CWE-276: Incorrect Default Permissions

#### Attack Scenario
`.github/workflows/ci.yml` does not declare a top-level `permissions:` block. By default, workflows inherit broad write permissions if the repository default is permissive.

#### Recommended Fix
Add `permissions: contents: read` to `.github/workflows/ci.yml`.

---

## Attack Surface Summary

| Surface | Risk | Notes |
|---|---|---|
| **CLI (`cmd/`)** | **CRITICAL** | Zero path containment check; arbitrary file creation outside workspace via `appName` and `--output-dir`. |
| **Filesystem (`internal/engine/`)** | **CRITICAL** | `os.WriteFile` and `os.MkdirAll` use unvalidated relative and absolute paths directly. |
| **MCP Server (`internal/mcp/`)** | **CRITICAL** | Workspace confinement guard is bypassed when `outputDir` is omitted; DoS crash on lines > 4MB. |
| **Templates & Go Generation** | **CRITICAL** | Unescaped string interpolations in `TableName`, `Endpoint`, `Validation`, and `JSONTag` enable arbitrary Go code execution (`init()` backdoors). |
| **SQL Migrations** | **CRITICAL** | Unescaped DDL interpolation of `tableName` and `default` leads to SQL injection. |
| **Generated Go Architecture** | **HIGH** | Complete BOLA/IDOR on CRUD endpoints, Mass Assignment on update endpoints, hardcoded JWT secrets for Mongo. |
| **Docker Configuration** | **MEDIUM** | Generated Dockerfile runs as `root`; compose file includes default passwords and secrets. |
| **GitHub Actions** | **LOW** | CI workflow lacks restrictive top-level permissions. |
| **NPM Wrapper (`npm/`)** | **HIGH** | Unchecked HTTP downloads without hash verification in `install.js`; version mismatch with package.json. |
| **Supply Chain & Hygiene** | **HIGH** | Committed 10MB Mach-O binary in repository root and 6.9MB binary in npm package. |

---

## Findings Matrix

| ID | Severity | Component | Exploitability | Impact | Confidence | Fix Priority |
|---|---|---|---|---|---|---|
| **CRIT-01** | Critical | CLI / MCP / Engine | High | High (Host File Overwrite) | High | P0 |
| **CRIT-02** | Critical | Template Engine (`TableName`) | High | High (RCE / Backdoor) | High | P0 |
| **CRIT-03** | Critical | AST Route Injector (`Endpoint`) | High | High (RCE / Backdoor) | High | P0 |
| **CRIT-04** | Critical | Template Engine (`Validation`) | High | High (RCE / Backdoor) | High | P0 |
| **CRIT-05** | Critical | Migrations (`TableName`, `Default`) | High | High (SQL Injection) | High | P0 |
| **HIGH-01** | High | Auth Template (MongoDB) | High | High (Auth Bypass) | High | P1 |
| **HIGH-02** | High | Generated Controllers | High | High (BOLA / IDOR) | High | P1 |
| **HIGH-03** | High | Generated Controllers/Services | High | Medium (Mass Assignment) | High | P1 |
| **HIGH-04** | High | NPM Wrapper (`install.js`) | Medium | High (Supply Chain RCE) | High | P1 |
| **HIGH-05** | High | Git Hygiene (Committed Binaries) | N/A | High (Supply Chain Trust) | High | P1 |
| **MED-01** | Medium | Auth Controller | High | Medium (Weak Passwords) | High | P2 |
| **MED-02** | Medium | Generated Middleware | High | Medium (Wildcard CORS) | High | P2 |
| **MED-03** | Medium | Generated Dockerfile | Medium | Medium (Root Privilege) | High | P2 |
| **MED-04** | Medium | Generated Config / Compose | Medium | Medium (Default Secrets) | High | P2 |
| **MED-05** | Medium | Generated Main / Auth | High | Medium (Credential Stuffing) | High | P2 |
| **MED-06** | Medium | Error Handling | High | Low (Info Disclosure) | High | P2 |
| **LOW-01** | Low | MCP Protocol (`server.go`) | Medium | Low (Server DoS) | High | P3 |
| **LOW-02** | Low | MCP Tools (`add_model`) | Medium | Low (Path Traversal) | High | P3 |
| **LOW-03** | Low | CI Workflows | Low | Low (Broad Permissions) | High | P3 |

---

## Security Strengths

Despite the findings above, FiberForge implements several sound design choices:
1. **Gofmt as a Syntax Gate:** All generated Go files pass through `go/format.Source()`. Syntax corruptions that do not form valid Go code are rejected before writing to disk.
2. **Safe Subprocess Execution:** The core FiberForge engine does not invoke arbitrary shell commands (`sh -c` or `bash`) with untrusted arguments; it generates pure code directly.
3. **Password Hashing:** Passwords in the authentication flow are hashed using `golang.org/x/crypto/bcrypt` with `bcrypt.DefaultCost` rather than unsalted or weak hashing algorithms.
4. **JWT Signing Verification:** Generated JWT validation explicitly verifies HMAC algorithm enforcement (`t.Method.(*jwt.SigningMethodHMAC)`), mitigating algorithm-confusion attacks.
5. **AST Route Updates:** Incremental route addition uses Go's standard `go/ast` parser to avoid destructive regular expression replacements in existing route files.

---

## Blind Spots & Residual Risk

1. **AI Agent Prompt Injection & Hallucination:** When AI agents interact with FiberForge tools, prompt injections from untrusted external documentation could lead an agent to generate schemas with malicious `tableName`, `endpoint`, or `validation` payloads. Schema validation must be strictly enforced server-side.
2. **Dynamic Database Extensions:** GORM AutoMigrate is generated by default. If developers run AutoMigrate against existing production databases with elevated permissions, unexpected schema alterations may occur.
3. **Local Testing Workflows:** The audit was conducted statically and with isolated non-destructive test harnesses in `/tmp/fiberforge-security-audit/`. Production multi-tenant database deployments were not actively attacked.
