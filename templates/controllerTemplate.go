package templates

const ControllerTemplate = `
package controller

import (
	"strconv"
	"github.com/gofiber/fiber/v2"
	"{{.AppName}}/model"
	"{{.AppName}}/service"
)

// Create{{.StructName}} creates a new {{.StructName}}.
func Create{{.StructName}}(ctx *fiber.Ctx) error {
	var {{.StructName}} model.{{.StructNameTitleCase}}
	if err := ctx.BodyParser(&{{.StructName}}); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	err := service.Create{{.StructName}}({{.StructName}})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create {{.StructName}}"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "{{.StructName}} created."})
}

// GetAll{{.StructNameTitleCase}}s retrieves all rows in {{.StructName}} table.
func GetAll{{.StructName}}(ctx *fiber.Ctx) error {

	{{.StructName}}s,err := service.GetAll{{.StructNameTitleCase}}s()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create {{.StructName}}"})
	}

	return ctx.Status(fiber.StatusOK).JSON({{.StructName}}s)
}

// Get{{.StructName}}ByID retrieves a {{.StructName}} by ID.
func Get{{.StructName}}ByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	{{if ne .DbType "mongodb"}}
	intId,err := strconv.Atoi(id)
	
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Id not valid"})
	}
	{{end}}
	{{.StructName}}, err := service.Get{{.StructName}}ByID({{if ne .DbType "mongodb"}}intId{{else}}id{{end}})
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "{{.StructName}} not found"})
	}

	return ctx.JSON({{.StructName}})
}

// Update{{.StructName}} updates an existing {{.StructName}} by ID.
func Update{{.StructName}}(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var updated{{.StructName}} model.{{.StructNameTitleCase}}
	if err := ctx.BodyParser(&updated{{.StructName}}); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	err := service.Update{{.StructName}}(updated{{.StructName}},id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update {{.StructName}}"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "{{.StructName}} updated"})
}

// Delete{{.StructName}}ByID deletes a {{.StructName}} by ID.
func Delete{{.StructName}}ByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	{{if ne .DbType "mongodb"}}
	intId,err := strconv.Atoi(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Id not valid"})
	}
	{{end}}
	err1 := service.Delete{{.StructName}}ByID({{if ne .DbType "mongodb"}}intId{{else}}id{{end}})
	if err1 != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete {{.StructName}}"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "{{.StructName}} deleted"})
}

`
