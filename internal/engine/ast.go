package engine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"

	"github.com/v-pat/fiberforge/internal/schema"
)

// RegisterRouteInAST parses an existing routes/routes.go file using Go AST and appends
// route group registration for a new model or feature module without touching existing code.
// Route statements are constructed programmatically via AST nodes, eliminating any code injection risk.
func RegisterRouteInAST(routesFilePath string, model schema.Model, authEnabled bool) error {
	if err := schema.ValidateEndpoint(model.Endpoint); err != nil {
		return fmt.Errorf("invalid route endpoint: %w", err)
	}

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

	// Programmatically construct AST statements for the route group
	newStmts := buildRouteGroupAST(varName, modelName, path, authEnabled && model.AuthProtected)

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

// buildRouteGroupAST constructs AST statements for a route group structurally.
func buildRouteGroupAST(varName, modelName, path string, isProtected bool) []ast.Stmt {
	cleanPath := "/" + strings.TrimPrefix(path, "/")
	var stmts []ast.Stmt

	// <varName> := api.Group("<cleanPath>")
	assignGroup := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(varName)},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("api"),
					Sel: ast.NewIdent("Group"),
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(cleanPath),
					},
				},
			},
		},
	}
	stmts = append(stmts, assignGroup)

	// <varName> = <varName>.Use(middleware.JWT(auth.Secret()))
	if isProtected {
		assignAuth := &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(varName)},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent(varName),
						Sel: ast.NewIdent("Use"),
					},
					Args: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("middleware"),
								Sel: ast.NewIdent("JWT"),
							},
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   ast.NewIdent("auth"),
										Sel: ast.NewIdent("Secret"),
									},
								},
							},
						},
					},
				},
			},
		}
		stmts = append(stmts, assignAuth)
	}

	makeMethodCall := func(method, route, handler string) *ast.ExprStmt {
		return &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent(varName),
					Sel: ast.NewIdent(method),
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(route),
					},
					&ast.SelectorExpr{
						X:   ast.NewIdent("controller"),
						Sel: ast.NewIdent(handler),
					},
				},
			},
		}
	}

	stmts = append(stmts,
		makeMethodCall("Post", "/", "Create"+modelName),
		makeMethodCall("Get", "/", "List"+modelName+"s"),
		makeMethodCall("Get", "/:id", "Get"+modelName+"ByID"),
		makeMethodCall("Put", "/:id", "Update"+modelName),
		makeMethodCall("Delete", "/:id", "Delete"+modelName+"ByID"),
	)

	return stmts
}
