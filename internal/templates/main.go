package templates

// MainTemplate renders the application entry point. All middleware imports are
// conditional so the generated file compiles regardless of which features are
// enabled.
const MainTemplate = `package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	{{if .CORS}}	"github.com/gofiber/fiber/v2/middleware/cors"
	{{end}}{{if .Helmet}}	"github.com/gofiber/fiber/v2/middleware/helmet"
	{{end}}{{if .RateLimit}}	"github.com/gofiber/fiber/v2/middleware/limiter"
	{{end}}{{if .Logging}}	"github.com/gofiber/fiber/v2/middleware/logger"
	{{end}}	"github.com/gofiber/fiber/v2/middleware/requestid"
	{{if .Recover}}	"github.com/gofiber/fiber/v2/middleware/recover"
	{{end}}
	"{{.AppName}}/databases"
	"{{.AppName}}/routes"
	{{if .Config}}	configpkg "{{.AppName}}/config"
	{{end}}
)

func main() {
	loggerHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(loggerHandler))

	{{if .Config}}if err := configpkg.Load(); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	{{end}}if err := databases.Connect(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		AppName: "{{.AppName}}",
	})

	app.Use(requestid.New())
	{{if .Logging}}
	app.Use(logger.New())
	{{end}}{{if .Recover}}
	app.Use(recover.New())
	{{end}}{{if .Helmet}}
	app.Use(helmet.New())
	{{end}}{{if .CORS}}
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))
	{{end}}{{if .RateLimit}}
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: time.Minute,
	}))
	{{end}}
	routes.Routes(app)

	port := "{{.Port}}"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting server", "port", port)
		if err := app.Listen(":" + port); err != nil {
			slog.Error("server listener error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server gracefully...")

	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		slog.Error("forced shutdown", "error", err)
	}
	slog.Info("server shutdown complete")
}
`

// MainData is the data passed to MainTemplate.
type MainData struct {
	AppName   string
	Port      string
	Config    bool
	Logging   bool
	Recover   bool
	Helmet    bool
	CORS      bool
	RateLimit bool
}

// EnvTemplate renders .env.example.
const EnvTemplate = `# Copy this file to .env and fill in real values.
# The app reads these at startup.
DB_HOST=localhost
DB_PORT={{.DBPort}}
DB_USER={{.DBUser}}
DB_PASSWORD={{.DBPassword}}
DB_NAME={{.AppName}}
{{if eq .DbType "mongodb"}}DB_URI=mongodb://localhost:{{.DBPort}}
{{end}}PORT={{.Port}}
{{if .Auth}}JWT_SECRET=change-me-in-production
{{end}}{{range $k, $v := .Env}}{{$k}}={{$v}}
{{end}}`

// ConfigTemplate renders the config loader package.
const ConfigTemplate = `// config/config.go
package config

import "os"

// Load verifies required environment variables, applying dev defaults so
// startup fails fast with a clear message when something is missing.
func Load() error {
	required := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "PORT"}
	for _, k := range required {
		if os.Getenv(k) == "" {
			os.Setenv(k, defaultValue(k))
		}
	}
	return nil
}

func defaultValue(k string) string {
	switch k {
	case "DB_HOST":
		return "localhost"
	case "DB_PORT":
		return "{{.DBPort}}"
	case "DB_USER":
		return "{{.DBUser}}"
	case "DB_PASSWORD":
		return "{{.DBPassword}}"
	case "DB_NAME":
		return "{{.AppName}}"
	case "PORT":
		return "{{.Port}}"
	default:
		return ""
	}
}
`
