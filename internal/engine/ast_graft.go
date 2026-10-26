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
)

// RouteHook specifies an HTTP route registration to inject via AST grafting.
type RouteHook struct {
	Group   string
	Path    string
	Method  string
	Handler string
	Auth    bool
}

// GraftImports parses a target Go source file and adds the given import packages idempotently.
func GraftImports(filePath string, imports []string) error {
	if len(imports) == 0 {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filePath, err)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", filePath, err)
	}

	existingImports := make(map[string]bool)
	for _, imp := range node.Imports {
		pathVal, _ := strconv.Unquote(imp.Path.Value)
		existingImports[pathVal] = true
	}

	var toAdd []string
	for _, imp := range imports {
		cleanImp := strings.Trim(imp, `"' `)
		if cleanImp != "" && !existingImports[cleanImp] {
			toAdd = append(toAdd, cleanImp)
		}
	}

	if len(toAdd) == 0 {
		return nil // All imports already present
	}

	// Locate existing import declaration or create a new one
	var importDecl *ast.GenDecl
	for _, decl := range node.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			importDecl = gen
			break
		}
	}

	if importDecl != nil {
		for _, pkg := range toAdd {
			importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(pkg),
				},
			})
		}
	} else {
		// Create new import declaration
		var specs []ast.Spec
		for _, pkg := range toAdd {
			specs = append(specs, &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(pkg),
				},
			})
		}
		newDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: specs,
		}
		node.Decls = append([]ast.Decl{newDecl}, node.Decls...)
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("formatting AST for %s: %w", filePath, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		formatted = buf.Bytes()
	}

	return os.WriteFile(filePath, formatted, 0o644)
}

// GraftRouteHook parses routes/routes.go and registers a route hook idempotently.
// If AST traversal cannot find func Routes, it falls back to comment markers.
func GraftRouteHook(routesFilePath string, hook RouteHook) error {
	data, err := os.ReadFile(routesFilePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", routesFilePath, err)
	}

	// Quick idempotency check: if handler and path are already present in file, skip
	cleanPath := "/" + strings.TrimPrefix(hook.Path, "/")
	if strings.Contains(string(data), strconv.Quote(cleanPath)) && strings.Contains(string(data), hook.Handler) {
		return nil
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, routesFilePath, data, parser.ParseComments)
	if err != nil {
		return fallbackGraftComment(routesFilePath, string(data), hook)
	}

	var routesFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "Routes" {
			routesFunc = fn
			break
		}
	}

	if routesFunc == nil {
		return fallbackGraftComment(routesFilePath, string(data), hook)
	}

	// Build AST statement for the route call
	method := strings.Title(strings.ToLower(hook.Method))
	receiverIdent := "api"
	if hook.Group != "" && hook.Group != "/api" {
		receiverIdent = "app"
	}

	var handlerExpr ast.Expr
	if strings.Contains(hook.Handler, ".") {
		parts := strings.Split(hook.Handler, ".")
		handlerExpr = &ast.SelectorExpr{
			X:   ast.NewIdent(parts[0]),
			Sel: ast.NewIdent(parts[1]),
		}
	} else {
		handlerExpr = ast.NewIdent(hook.Handler)
	}

	var callArgs []ast.Expr
	callArgs = append(callArgs, &ast.BasicLit{
		Kind:  token.STRING,
		Value: strconv.Quote(cleanPath),
	})

	if hook.Auth {
		authCall := &ast.CallExpr{
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
		}
		callArgs = append(callArgs, authCall)
	}

	callArgs = append(callArgs, handlerExpr)

	stmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(receiverIdent),
				Sel: ast.NewIdent(method),
			},
			Args: callArgs,
		},
	}

	routesFunc.Body.List = append(routesFunc.Body.List, stmt)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fallbackGraftComment(routesFilePath, string(data), hook)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		formatted = buf.Bytes()
	}

	return os.WriteFile(routesFilePath, formatted, 0o644)
}

func fallbackGraftComment(filePath, content string, hook RouteHook) error {
	cleanPath := "/" + strings.TrimPrefix(hook.Path, "/")
	method := strings.Title(strings.ToLower(hook.Method))
	stmt := fmt.Sprintf("\tapi.%s(%q, %s)", method, cleanPath, hook.Handler)
	if hook.Auth {
		stmt = fmt.Sprintf("\tapi.%s(%q, middleware.JWT(auth.Secret()), %s)", method, cleanPath, hook.Handler)
	}

	markerStart := "// fiberforge:blueprint:routes:start"
	markerEnd := "// fiberforge:blueprint:routes:end"

	if strings.Contains(content, markerStart) && strings.Contains(content, markerEnd) {
		replacement := markerStart + "\n" + stmt + "\n"
		newContent := strings.Replace(content, markerStart+"\n", replacement, 1)
		return os.WriteFile(filePath, []byte(newContent), 0o644)
	}

	// Fallback to appending before the last closing brace
	lastBrace := strings.LastIndex(content, "}")
	if lastBrace != -1 {
		newContent := content[:lastBrace] + "\n" + stmt + "\n" + content[lastBrace:]
		return os.WriteFile(filePath, []byte(newContent), 0o644)
	}

	return fmt.Errorf("unable to graft route into %s: no Routes func or closing brace found", filePath)
}

// GraftMiddleware appends a middleware registration to routes/routes.go idempotently.
func GraftMiddleware(routesFilePath string, middlewareExpr string) error {
	data, err := os.ReadFile(routesFilePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", routesFilePath, err)
	}

	if strings.Contains(string(data), middlewareExpr) {
		return nil
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, routesFilePath, data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", routesFilePath, err)
	}

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

	// Parse expression
	exprAst, err := parser.ParseExpr(middlewareExpr)
	if err != nil {
		return fmt.Errorf("invalid middleware expression %q: %w", middlewareExpr, err)
	}

	stmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("app"),
				Sel: ast.NewIdent("Use"),
			},
			Args: []ast.Expr{exprAst},
		},
	}

	// Prepend middleware to top of Routes func
	routesFunc.Body.List = append([]ast.Stmt{stmt}, routesFunc.Body.List...)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("formatting AST: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		formatted = buf.Bytes()
	}

	return os.WriteFile(routesFilePath, formatted, 0o644)
}

// GraftInitCall appends an initialization call to func main() in main.go idempotently.
func GraftInitCall(mainFilePath string, initExpr string) error {
	data, err := os.ReadFile(mainFilePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", mainFilePath, err)
	}

	if strings.Contains(string(data), initExpr) {
		return nil
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, mainFilePath, data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", mainFilePath, err)
	}

	var mainFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			mainFunc = fn
			break
		}
	}

	if mainFunc == nil {
		return fmt.Errorf("func main not found in %s", mainFilePath)
	}

	exprAst, err := parser.ParseExpr(initExpr)
	if err != nil {
		return fmt.Errorf("invalid init expression %q: %w", initExpr, err)
	}

	stmt := &ast.ExprStmt{X: exprAst}
	// Append statement into main()
	mainFunc.Body.List = append(mainFunc.Body.List, stmt)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return fmt.Errorf("formatting AST: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		formatted = buf.Bytes()
	}

	return os.WriteFile(mainFilePath, formatted, 0o644)
}
