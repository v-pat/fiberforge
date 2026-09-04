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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := service.Create{{.Name}}(&m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

// List{{.Name}}s handles GET /{{.Endpoint}}.
func List{{.Name}}s(c *fiber.Ctx) error {
	items, err := service.List{{.Name}}s()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

// Get{{.Name}}ByID handles GET /{{.Endpoint}}/:id.
func Get{{.Name}}ByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	m, err := service.Get{{.Name}}ByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(m)
}

// Update{{.Name}} handles PUT /{{.Endpoint}}/:id.
func Update{{.Name}}(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var patch model.{{.Name}}
	if err := c.BodyParser(&patch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := service.Update{{.Name}}(uint(id), &patch); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "{{.Name}} updated"})
}

// Delete{{.Name}}ByID handles DELETE /{{.Endpoint}}/:id.
func Delete{{.Name}}ByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := service.Delete{{.Name}}ByID(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "{{.Name}} deleted"})
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := service.Create{{.Name}}(&m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

// List{{.Name}}s handles GET /{{.Endpoint}}.
func List{{.Name}}s(c *fiber.Ctx) error {
	items, err := service.List{{.Name}}s()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

// Get{{.Name}}ByID handles GET /{{.Endpoint}}/:id.
func Get{{.Name}}ByID(c *fiber.Ctx) error {
	m, err := service.Get{{.Name}}ByID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(m)
}

// Update{{.Name}} handles PUT /{{.Endpoint}}/:id.
func Update{{.Name}}(c *fiber.Ctx) error {
	var patch model.{{.Name}}
	if err := c.BodyParser(&patch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := service.Update{{.Name}}(c.Params("id"), &patch); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "{{.Name}} updated"})
}

// Delete{{.Name}}ByID handles DELETE /{{.Endpoint}}/:id.
func Delete{{.Name}}ByID(c *fiber.Ctx) error {
	if err := service.Delete{{.Name}}ByID(c.Params("id")); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "{{.Name}} deleted"})
}
`
