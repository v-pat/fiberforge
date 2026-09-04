package templates

// ControllerTestTemplate renders a smoke test file for a model's endpoints
// when the testing feature is enabled. Uses a test package to avoid import cycles.
const ControllerTestTemplate = `package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"{{.AppName}}/controller"
	"{{.AppName}}/databases"
)

// TestList{{.Name}}s tests the List{{.Name}}s handler.
func TestList{{.Name}}s(t *testing.T) {
	if err := databases.Ping(); err != nil {
		t.Skip("skipping test: database not connected")
	}
	app := fiber.New()
	app.Get("/api/{{.Endpoint}}", controller.List{{.Name}}s)
	req := httptest.NewRequest(http.MethodGet, "/api/{{.Endpoint}}", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestCreate{{.Name}} tests the Create{{.Name}} handler.
func TestCreate{{.Name}}(t *testing.T) {
	if err := databases.Ping(); err != nil {
		t.Skip("skipping test: database not connected")
	}
	app := fiber.New()
	app.Post("/api/{{.Endpoint}}", controller.Create{{.Name}})
	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/{{.Endpoint}}", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}
`

// AuthTestTemplate renders an auth smoke test when both auth and testing are on.
const AuthTestTemplate = `package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"{{.AppName}}/controller"
	"{{.AppName}}/databases"
)

// TestRegisterLogin tests the register and login handlers.
func TestRegisterLogin(t *testing.T) {
	if err := databases.Ping(); err != nil {
		t.Skip("skipping test: database not connected")
	}
	app := fiber.New()
	app.Post("/api/auth/register", controller.Register)
	app.Post("/api/auth/login", controller.Login)

	reg, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "secret123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reg))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}
`
