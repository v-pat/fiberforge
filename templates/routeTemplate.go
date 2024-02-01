package templates

const RoutesTemplate = `
package routes

import (
    "github.com/gofiber/fiber/v2"
    "{{.AppName}}/controller"
)

// Setup sets up routes for the structs resource.
func Routes(app *fiber.App) {
	{{range  .StructNames}}
    // Define routes for {{.StructName}} resource
    {{.StructName}}_group := app.Group("/{{.Endpoint}}")

    // Create a {{.StructName}}
    {{.StructName}}_group.Post("/", controller.Create{{.StructName}})

    // Get a {{.StructName}} by ID
    {{.StructName}}_group.Get("/:id", controller.Get{{.StructName}}ByID)

    // Update a {{.StructName}} by ID
    {{.StructName}}_group.Put("/:id", controller.Update{{.StructName}})

    // Delete a {{.StructName}} by ID
    {{.StructName}}_group.Delete("/:id", controller.Delete{{.StructName}}ByID)
	{{end}}
}
`
