package templates

// ControllerTemplate renders the Fiber HTTP handlers for a SQL model.
const ControllerTemplate = `package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/model"
	"{{.AppName}}/service"
)

// Create{{.Name}} handles POST /{{.Endpoint}}.
func Create{{.Name}}(c *fiber.Ctx) error {
	var m model.{{.Name}}
	if err := c.BodyParser(&m); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&m); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	if err := service.Create{{.Name}}(&m); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusCreated, m)
}

// List{{.Name}}s handles GET /{{.Endpoint}}.
func List{{.Name}}s(c *fiber.Ctx) error {
	items, err := service.List{{.Name}}s()
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, items)
}

// Get{{.Name}}ByID handles GET /{{.Endpoint}}/:id.
func Get{{.Name}}ByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id", nil)
	}
	m, err := service.Get{{.Name}}ByID(uint(id))
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, m)
}

// Update{{.Name}} handles PUT /{{.Endpoint}}/:id.
func Update{{.Name}}(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id", nil)
	}
	var patch model.{{.Name}}
	if err := c.BodyParser(&patch); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&patch); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	if err := service.Update{{.Name}}(uint(id), &patch); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} updated"})
}

// Delete{{.Name}}ByID handles DELETE /{{.Endpoint}}/:id.
func Delete{{.Name}}ByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id", nil)
	}
	if err := service.Delete{{.Name}}ByID(uint(id)); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} deleted"})
}
`

// MongoControllerTemplate renders Fiber handlers for a Mongo model.
const MongoControllerTemplate = `package controller

import (
	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/model"
	"{{.AppName}}/service"
)

// Create{{.Name}} handles POST /{{.Endpoint}}.
func Create{{.Name}}(c *fiber.Ctx) error {
	var m model.{{.Name}}
	if err := c.BodyParser(&m); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&m); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	if err := service.Create{{.Name}}(&m); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusCreated, m)
}

// List{{.Name}}s handles GET /{{.Endpoint}}.
func List{{.Name}}s(c *fiber.Ctx) error {
	items, err := service.List{{.Name}}s()
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, items)
}

// Get{{.Name}}ByID handles GET /{{.Endpoint}}/:id.
func Get{{.Name}}ByID(c *fiber.Ctx) error {
	m, err := service.Get{{.Name}}ByID(c.Params("id"))
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, m)
}

// Update{{.Name}} handles PUT /{{.Endpoint}}/:id.
func Update{{.Name}}(c *fiber.Ctx) error {
	var patch model.{{.Name}}
	if err := c.BodyParser(&patch); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&patch); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	if err := service.Update{{.Name}}(c.Params("id"), &patch); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} updated"})
}

// Delete{{.Name}}ByID handles DELETE /{{.Endpoint}}/:id.
func Delete{{.Name}}ByID(c *fiber.Ctx) error {
	if err := service.Delete{{.Name}}ByID(c.Params("id")); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"message": "{{.Name}} deleted"})
}
`
