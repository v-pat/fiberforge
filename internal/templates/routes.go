package templates

// RoutesTemplate renders the central route registration. Each model gets a CRUD
// group. Auth-protected models are wrapped in the JWT middleware when auth is on.
const RoutesTemplate = `package routes

import (
{{if .AuthEnabled}}	"time"

	"github.com/gofiber/fiber/v2/middleware/limiter"
{{end}}	"github.com/gofiber/fiber/v2"

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
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "DOWN", "error": "service unavailable"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "UP"})
	})

	api := app.Group("/api")
	{{if .AuthEnabled}}
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "TOO_MANY_REQUESTS",
					"message": "too many authentication attempts, please try again later",
				},
			})
		},
	})
	// Auth endpoints protected by dedicated brute-force rate limiter.
	api.Post("/auth/register", authLimiter, controller.Register)
	api.Post("/auth/login", authLimiter, controller.Login)

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
