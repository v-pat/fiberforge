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
	"strings"
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

	fiberCfg := fiber.Config{
		AppName: "{{.AppName}}",
	}
	if tp := os.Getenv("TRUSTED_PROXIES"); strings.TrimSpace(tp) != "" {
		var proxies []string
		for _, p := range strings.Split(tp, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				proxies = append(proxies, trimmed)
			}
		}
		if len(proxies) > 0 {
			fiberCfg.EnableTrustedProxyCheck = true
			fiberCfg.TrustedProxies = proxies
			fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor
		}
	}

	app := fiber.New(fiberCfg)

	app.Use(requestid.New())
	{{if .Logging}}
	app.Use(logger.New())
	{{end}}{{if .Recover}}
	app.Use(recover.New())
	{{end}}{{if .Helmet}}
	app.Use(helmet.New())
	{{end}}{{if .CORS}}
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000, http://localhost:8080"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
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
DB_PASSWORD=your_database_password
DB_NAME={{.AppName}}
{{if eq .DbType "mongodb"}}DB_URI=mongodb://localhost:{{.DBPort}}
{{end}}PORT={{.Port}}
{{if .Auth}}JWT_SECRET=your_jwt_secret_min_16_chars
{{end}}{{if .CORS}}CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
{{end}}# Reverse proxy / trusted proxies (comma-separated IP or CIDR ranges, e.g. 10.0.0.0/8,127.0.0.1).
# Leave empty unless running behind a trusted reverse proxy (NGINX, Cloudflare, AWS ALB).
# NEVER set to unrestricted ranges (e.g. 0.0.0.0/0) as this allows client IP header spoofing.
# TRUSTED_PROXIES=
{{range $k, $v := .Env}}{{$k}}={{$v}}
{{end}}`

// ConfigTemplate renders the config loader package.
const ConfigTemplate = `// config/config.go
package config

import (
	"fmt"
	"os"
	"strings"
)

// Load verifies required environment variables. In development, sensible
// defaults are applied. In production, startup fails if secrets/credentials
// are missing or set to default/insecure values.
func Load() error {
	isProd := strings.EqualFold(os.Getenv("ENV"), "production") ||
		strings.EqualFold(os.Getenv("APP_ENV"), "production")

{{if eq .DbType "mongodb"}}	required := []string{"DB_URI", "DB_NAME", "PORT"}
{{else}}	required := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "PORT"}
{{end}}
{{if .Auth}}	required = append(required, "JWT_SECRET")
{{end}}

	for _, k := range required {
		val := os.Getenv(k)
		if val == "" {
			if isProd {
				return fmt.Errorf("missing required environment variable %q in production", k)
			}
			os.Setenv(k, defaultValue(k))
			val = os.Getenv(k)
		}

		if isProd {
			if k == "DB_PASSWORD" && (val == "password" || val == "postgres" || val == "root") {
				return fmt.Errorf("insecure default password for %s is not permitted in production", k)
			}
			if k == "JWT_SECRET" && (val == "change-me-in-production" || val == "dev-jwt-secret-do-not-use-in-production" || len(val) < 16) {
				return fmt.Errorf("insecure or weak %s is not permitted in production (minimum 16 characters required)", k)
			}
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
		return "password"
	case "DB_NAME":
		return "{{.AppName}}"
	case "DB_URI":
		return "mongodb://localhost:{{.DBPort}}"
	case "PORT":
		return "{{.Port}}"
	case "JWT_SECRET":
		return "dev-jwt-secret-do-not-use-in-production"
	default:
		return ""
	}
}
`
