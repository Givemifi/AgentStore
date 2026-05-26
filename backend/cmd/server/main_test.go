package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMainRegistersTenantMiddlewareForChatRoutes(t *testing.T) {
	file := parseMainGo(t)
	if !chatAPIUsesMiddleware(file, "tenantMiddleware", "RequireTenant") {
		t.Fatal("expected chatAPI to register tenantMiddleware.RequireTenant in main.go")
	}
}

func TestMainRegistersActiveBillingMiddlewareForChatRoutes(t *testing.T) {
	file := parseMainGo(t)
	if !chatAPIUsesMiddleware(file, "middleware", "RequireActiveBilling") {
		t.Fatal("expected chatAPI to register middleware.RequireActiveBilling in main.go")
	}
}

func parseMainGo(t *testing.T) *ast.File {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}

	mainPath := filepath.Join(filepath.Dir(thisFile), "main.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, mainPath, nil, 0)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}
	return file
}

func chatAPIUsesMiddleware(file *ast.File, receiverName, selectorName string) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Use" {
			return true
		}

		receiver, ok := sel.X.(*ast.Ident)
		if !ok || receiver.Name != "chatAPI" {
			return true
		}

		if len(call.Args) != 1 {
			return true
		}

		middlewareSelector := extractSelector(call.Args[0])
		if middlewareSelector == nil {
			return true
		}

		middlewareReceiver, ok := middlewareSelector.X.(*ast.Ident)
		if !ok {
			return true
		}

		if middlewareReceiver.Name == receiverName && middlewareSelector.Sel.Name == selectorName {
			found = true
			return false
		}

		return true
	})
	return found
}

func extractSelector(expr ast.Expr) *ast.SelectorExpr {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		return v
	case *ast.CallExpr:
		selector, _ := v.Fun.(*ast.SelectorExpr)
		return selector
	default:
		return nil
	}
}
