package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraftImports(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "routes.go")

	initialCode := `package routes

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(app *fiber.App) {}
`
	if err := os.WriteFile(filePath, []byte(initialCode), 0o644); err != nil {
		t.Fatal(err)
	}

	// Graft new package imports
	toAdd := []string{"myapp/pkg/mailer", "github.com/resend/resend-go/v2"}
	if err := GraftImports(filePath, toAdd); err != nil {
		t.Fatalf("GraftImports failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, `"myapp/pkg/mailer"`) {
		t.Errorf("expected myapp/pkg/mailer import, got:\n%s", strContent)
	}
	if !strings.Contains(strContent, `"github.com/resend/resend-go/v2"`) {
		t.Errorf("expected resend-go import, got:\n%s", strContent)
	}

	// Idempotency check: run again with same imports
	if err := GraftImports(filePath, toAdd); err != nil {
		t.Fatalf("second GraftImports call failed: %v", err)
	}

	contentSecond, _ := os.ReadFile(filePath)
	count := strings.Count(string(contentSecond), `"myapp/pkg/mailer"`)
	if count != 1 {
		t.Errorf("expected exactly 1 instance of import, got %d", count)
	}
}

func TestGraftRouteHook(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "routes.go")

	initialCode := `package routes

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(app *fiber.App) {
	api := app.Group("/api")
}
`
	if err := os.WriteFile(filePath, []byte(initialCode), 0o644); err != nil {
		t.Fatal(err)
	}

	hook := RouteHook{
		Path:    "/webhooks/stripe",
		Method:  "POST",
		Handler: "stripe.HandleWebhook",
		Auth:    false,
	}

	if err := GraftRouteHook(filePath, hook); err != nil {
		t.Fatalf("GraftRouteHook failed: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	strContent := string(content)
	if !strings.Contains(strContent, `api.Post("/webhooks/stripe", stripe.HandleWebhook)`) {
		t.Errorf("expected route registration in Routes(), got:\n%s", strContent)
	}

	// Idempotency: run again
	if err := GraftRouteHook(filePath, hook); err != nil {
		t.Fatalf("second GraftRouteHook failed: %v", err)
	}

	contentSecond, _ := os.ReadFile(filePath)
	if strings.Count(string(contentSecond), `stripe.HandleWebhook`) != 1 {
		t.Errorf("expected route to be registered exactly once, got:\n%s", string(contentSecond))
	}
}

func TestGraftRouteHook_WithAuth(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "routes.go")

	initialCode := `package routes

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(app *fiber.App) {
	api := app.Group("/api")
}
`
	if err := os.WriteFile(filePath, []byte(initialCode), 0o644); err != nil {
		t.Fatal(err)
	}

	hook := RouteHook{
		Path:    "/admin/metrics",
		Method:  "GET",
		Handler: "admin.GetMetrics",
		Auth:    true,
	}

	if err := GraftRouteHook(filePath, hook); err != nil {
		t.Fatalf("GraftRouteHook with auth failed: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	strContent := string(content)
	if !strings.Contains(strContent, `middleware.JWT(auth.Secret())`) {
		t.Errorf("expected auth middleware wrapped in route hook, got:\n%s", strContent)
	}
}

func TestGraftRouteHook_CommentFallback(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "routes.go")

	initialCode := `package routes
// fiberforge:blueprint:routes:start
// fiberforge:blueprint:routes:end
`
	if err := os.WriteFile(filePath, []byte(initialCode), 0o644); err != nil {
		t.Fatal(err)
	}

	hook := RouteHook{
		Path:    "/webhooks/resend",
		Method:  "POST",
		Handler: "mailer.Webhook",
	}

	if err := GraftRouteHook(filePath, hook); err != nil {
		t.Fatalf("fallback graft failed: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	strContent := string(content)
	if !strings.Contains(strContent, `api.Post("/webhooks/resend", mailer.Webhook)`) {
		t.Errorf("expected comment fallback injection, got:\n%s", strContent)
	}
}

func TestGraftMiddleware(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "routes.go")

	initialCode := `package routes

import "github.com/gofiber/fiber/v2"

func Routes(app *fiber.App) {
	api := app.Group("/api")
}
`
	if err := os.WriteFile(filePath, []byte(initialCode), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := GraftMiddleware(filePath, `sentryfiber.New(sentryfiber.Options{})`); err != nil {
		t.Fatalf("GraftMiddleware failed: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	strContent := string(content)
	if !strings.Contains(strContent, `app.Use(sentryfiber.New`) || !strings.Contains(strContent, `sentryfiber.Options{}`) {
		t.Errorf("expected app.Use(sentryfiber...) middleware, got:\n%s", strContent)
	}
}

func TestGraftInitCall(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "main.go")

	initialCode := `package main

func main() {
	println("server starting")
}
`
	if err := os.WriteFile(filePath, []byte(initialCode), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := GraftInitCall(filePath, `mailer.Init()`); err != nil {
		t.Fatalf("GraftInitCall failed: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	strContent := string(content)
	if !strings.Contains(strContent, `mailer.Init()`) {
		t.Errorf("expected mailer.Init() in main(), got:\n%s", strContent)
	}
}
