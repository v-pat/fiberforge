package templates

// ControllerTemplate renders the Fiber HTTP handlers for a SQL model.
const ControllerTemplate = `package controller

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/model"
	"{{.AppName}}/service"
)

// Create{{.Name}} handles POST /{{.Endpoint}}.
func Create{{.Name}}(c *fiber.Ctx) error {
	var input model.Create{{.Name}}Input
	if err := c.BodyParser(&input); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&input); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	var m model.{{.Name}}
{{range .WritableFields}}	m.{{.GoName}} = input.{{.GoName}}
{{end}}
{{if .IsOwned}}
	if uid, ok := c.Locals("userId").(uint); ok {
		m.UserID = uid
	} else {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
{{end}}
	if err := service.Create{{.Name}}(&m); err != nil {
		slog.Error("failed to create {{.Name}}", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to create {{.Name}}", nil)
	}
	return SendSuccess(c, fiber.StatusCreated, m)
}

// List{{.Name}}s handles GET /{{.Endpoint}}.
func List{{.Name}}s(c *fiber.Ctx) error {
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	items, err := service.List{{.Name}}s(userID)
{{else}}
	items, err := service.List{{.Name}}s()
{{end}}
	if err != nil {
		slog.Error("failed to list {{.Name}}s", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to list {{.Name}}s", nil)
	}
	return SendSuccess(c, fiber.StatusOK, items)
}

// Get{{.Name}}ByID handles GET /{{.Endpoint}}/:id.
func Get{{.Name}}ByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id", nil)
	}
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	m, err := service.Get{{.Name}}ByID(uint(id), userID)
{{else}}
	m, err := service.Get{{.Name}}ByID(uint(id))
{{end}}
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "NOT_FOUND", "{{.Name}} not found", nil)
	}
	return SendSuccess(c, fiber.StatusOK, m)
}

// Update{{.Name}} handles PUT /{{.Endpoint}}/:id.
func Update{{.Name}}(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id", nil)
	}
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
{{end}}
	var patch model.Update{{.Name}}Input
	if err := c.BodyParser(&patch); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&patch); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
{{if .IsOwned}}
	if err := service.Update{{.Name}}(uint(id), userID, &patch); err != nil {
{{else}}
	if err := service.Update{{.Name}}(uint(id), &patch); err != nil {
{{end}}
		if err.Error() == "{{.Name}} not found" {
			return SendError(c, fiber.StatusNotFound, "NOT_FOUND", "{{.Name}} not found", nil)
		}
		slog.Error("failed to update {{.Name}}", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to update {{.Name}}", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} updated"})
}

// Delete{{.Name}}ByID handles DELETE /{{.Endpoint}}/:id.
func Delete{{.Name}}ByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id", nil)
	}
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(uint)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	if err := service.Delete{{.Name}}ByID(uint(id), userID); err != nil {
{{else}}
	if err := service.Delete{{.Name}}ByID(uint(id)); err != nil {
{{end}}
		if err.Error() == "{{.Name}} not found" {
			return SendError(c, fiber.StatusNotFound, "NOT_FOUND", "{{.Name}} not found", nil)
		}
		slog.Error("failed to delete {{.Name}}", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete {{.Name}}", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} deleted"})
}
`

// MongoControllerTemplate renders Fiber handlers for a Mongo model.
const MongoControllerTemplate = `package controller

import (
	"log/slog"

{{if .IsOwned}}	"go.mongodb.org/mongo-driver/bson/primitive"
{{end}}	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/model"
	"{{.AppName}}/service"
)

// Create{{.Name}} handles POST /{{.Endpoint}}.
func Create{{.Name}}(c *fiber.Ctx) error {
	var input model.Create{{.Name}}Input
	if err := c.BodyParser(&input); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&input); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	var m model.{{.Name}}
{{range .WritableFields}}	m.{{.GoName}} = input.{{.GoName}}
{{end}}
{{if .IsOwned}}
	if uid, ok := c.Locals("userId").(string); ok {
		if oid, err := primitive.ObjectIDFromHex(uid); err == nil {
			m.UserID = oid
		}
	} else {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
{{end}}
	if err := service.Create{{.Name}}(&m); err != nil {
		slog.Error("failed to create {{.Name}}", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to create {{.Name}}", nil)
	}
	return SendSuccess(c, fiber.StatusCreated, m)
}

// List{{.Name}}s handles GET /{{.Endpoint}}.
func List{{.Name}}s(c *fiber.Ctx) error {
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	items, err := service.List{{.Name}}s(userID)
{{else}}
	items, err := service.List{{.Name}}s()
{{end}}
	if err != nil {
		slog.Error("failed to list {{.Name}}s", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to list {{.Name}}s", nil)
	}
	return SendSuccess(c, fiber.StatusOK, items)
}

// Get{{.Name}}ByID handles GET /{{.Endpoint}}/:id.
func Get{{.Name}}ByID(c *fiber.Ctx) error {
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	m, err := service.Get{{.Name}}ByID(c.Params("id"), userID)
{{else}}
	m, err := service.Get{{.Name}}ByID(c.Params("id"))
{{end}}
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "NOT_FOUND", "{{.Name}} not found", nil)
	}
	return SendSuccess(c, fiber.StatusOK, m)
}

// Update{{.Name}} handles PUT /{{.Endpoint}}/:id.
func Update{{.Name}}(c *fiber.Ctx) error {
	var patch model.Update{{.Name}}Input
	if err := c.BodyParser(&patch); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&patch); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	if err := service.Update{{.Name}}(c.Params("id"), userID, &patch); err != nil {
{{else}}
	if err := service.Update{{.Name}}(c.Params("id"), &patch); err != nil {
{{end}}
		if err.Error() == "{{.Name}} not found" {
			return SendError(c, fiber.StatusNotFound, "NOT_FOUND", "{{.Name}} not found", nil)
		}
		slog.Error("failed to update {{.Name}}", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to update {{.Name}}", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} updated"})
}

// Delete{{.Name}}ByID handles DELETE /{{.Endpoint}}/:id.
func Delete{{.Name}}ByID(c *fiber.Ctx) error {
{{if .IsOwned}}
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	if err := service.Delete{{.Name}}ByID(c.Params("id"), userID); err != nil {
{{else}}
	if err := service.Delete{{.Name}}ByID(c.Params("id")); err != nil {
{{end}}
		if err.Error() == "{{.Name}} not found" {
			return SendError(c, fiber.StatusNotFound, "NOT_FOUND", "{{.Name}} not found", nil)
		}
		slog.Error("failed to delete {{.Name}}", "error", err)
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete {{.Name}}", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} deleted"})
}
`
