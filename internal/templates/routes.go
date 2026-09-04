package templates

// RoutesTemplate renders the central route registration. Each model gets a CRUD
// group. Auth-protected models are wrapped in the JWT middleware when auth is on.
const RoutesTemplate = `package routes

import (
	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/controller"
	"{{.AppName}}/databases"
	{{if .AuthEnabled}}"{{.AppName}}/middleware"
	"{{.AppName}}/auth"{{end}}
)

// Routes registers all API routes on the given app.
func Routes(app *fiber.App) {
	// Health check endpoints for probes / orchestrators.
	app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "UP"})
	})
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if err := databases.Ping(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "DOWN", "error": err.Error()})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "UP"})
	})

	api := app.Group("/api")
	{{if .AuthEnabled}}
	// Unprotected auth endpoints.
	api.Post("/auth/register", controller.Register)
	api.Post("/auth/login", controller.Login)

	authGroup := api.Group("", middleware.JWT(auth.Secret()))
	authGroup.Get("/auth/me", controller.Me)
	authGroup.Post("/auth/refresh", controller.Refresh)
	{{end}}
	{{range .Groups}}{{.Var}} := api.Group("/{{.Path}}")
	{{if .Auth}}{{.Var}} = {{.Var}}.Use(middleware.JWT(auth.Secret()))
	{{end}}{{.Var}}.Post("/", controller.Create{{.Name}})
	{{.Var}}.Get("/", controller.List{{.Name}}s)
	{{.Var}}.Get("/:id", controller.Get{{.Name}}ByID)
	{{.Var}}.Put("/:id", controller.Update{{.Name}})
	{{.Var}}.Delete("/:id", controller.Delete{{.Name}}ByID)
	{{end}}
}
`
