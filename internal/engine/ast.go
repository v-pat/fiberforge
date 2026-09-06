package engine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"

	"github.com/v-pat/fiberforge/internal/schema"
)

// RegisterRouteInAST parses an existing routes/routes.go file using Go AST and appends
// route group registration for a new model or feature module without touching existing code.
func RegisterRouteInAST(routesFilePath string, model schema.Model, authEnabled bool) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, routesFilePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", routesFilePath, err)
	}

	// Find func Routes(app *fiber.App)
	var routesFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "Routes" {
			routesFunc = fn
			break
		}
	}
	if routesFunc == nil {
		return fmt.Errorf("func Routes not found in %s", routesFilePath)
	}

	modelName := schema.Pascal(model.Name)
	varName := schema.Camel(model.Name) + "Group"
	path := model.Endpoint

	// Idempotency check: if varName is already declared in func Routes AST, return early
	for _, stmt := range routesFunc.Body.List {
		if assign, ok := stmt.(*ast.AssignStmt); ok {
			for _, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == varName {
					return nil
				}
			}
		}
	}

	// Construct route group statements Go code snippet
	snippet := fmt.Sprintf(`package dummy

func dummy() {
	%s := api.Group("/%s")
`, varName, path)

	if authEnabled && model.AuthProtected {
		snippet += fmt.Sprintf("\t%s = %s.Use(middleware.JWT(auth.Secret()))\n", varName, varName)
	}

	snippet += fmt.Sprintf(`	%s.Post("/", controller.Create%s)
	%s.Get("/", controller.List%ss)
	%s.Get("/:id", controller.Get%sByID)
	%s.Put("/:id", controller.Update%s)
	%s.Delete("/:id", controller.Delete%sByID)
}`, varName, modelName, varName, modelName, varName, modelName, varName, modelName, varName, modelName)

	dummyFset := token.NewFileSet()
	dummyNode, err := parser.ParseFile(dummyFset, "", snippet, 0)
	if err != nil {
		return fmt.Errorf("failed to parse route snippet: %w", err)
	}

	// Extract statements from dummy func
	dummyFunc := dummyNode.Decls[0].(*ast.FuncDecl)
	newStmts := dummyFunc.Body.List

	// Append new statements to Routes() body
	routesFunc.Body.List = append(routesFunc.Body.List, newStmts...)

	// Format updated AST back to source file
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("failed to format AST for %s: %w", routesFilePath, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		formatted = buf.Bytes()
	}

	return os.WriteFile(routesFilePath, formatted, 0o644)
}
