package engine

import (
	"fmt"

	"github.com/v-pat/fiberforge/internal/schema"
	tmpl "github.com/v-pat/fiberforge/internal/templates"
)

// Local aliases to the templates package so pipeline.go stays concise.
var (
	mongoDBTemplate              = tmpl.MongoDBTemplate
	goModTemplate                = tmpl.GoModTemplate
	gitignoreTemplate            = tmpl.GitignoreTemplate
	modelTemplate                = tmpl.ModelTemplate
	mongoModelTemplate           = tmpl.MongoModelTemplate
	serviceTemplate              = tmpl.ServiceTemplate
	mongoServiceTemplate         = tmpl.MongoServiceTemplate
	controllerTemplate           = tmpl.ControllerTemplate
	controllerUtilsTemplate      = tmpl.ControllerUtilsTemplate
	mongoControllerTemplate      = tmpl.MongoControllerTemplate
	routesTemplate               = tmpl.RoutesTemplate
	mainTemplate                 = tmpl.MainTemplate
	configTemplate               = tmpl.ConfigTemplate
	swaggerJSONTemplate          = tmpl.SwaggerJSONTemplate
	controllerTestTemplate       = tmpl.ControllerTestTemplate
	authTestTemplate             = tmpl.AuthTestTemplate
	authPasswordTemplate         = tmpl.AuthTemplate
	authJWTTemplate              = tmpl.JWTTemplate
	mongoAuthJWTTemplate         = tmpl.MongoJWTTemplate
	authStoreTemplate            = tmpl.AuthServiceTemplate
	mongoAuthStoreTemplate       = tmpl.MongoAuthServiceTemplate
	middlewareTemplate           = tmpl.MiddlewareTemplate
	mongoMiddlewareTemplate      = tmpl.MongoMiddlewareTemplate
	authControllerTemplate       = tmpl.AuthControllerTemplate
	mongoAuthControllerTemplate  = tmpl.MongoAuthControllerTemplate
	authUserServiceTemplate      = tmpl.AuthUserServiceTemplate
	mongoAuthUserServiceTemplate = tmpl.MongoAuthUserServiceTemplate
)

// renderReadme renders the project README from the config.
func (e *Engine) renderReadme() (string, error) {
	var models []map[string]string
	for _, m := range e.cfg.Models {
		if e.cfg.Features.Auth && schema.Lower(m.Name) == "user" {
			continue
		}
		models = append(models, map[string]string{
			"Endpoint": m.Endpoint,
		})
	}
	return e.render("readme", tmpl.READMETemplate, map[string]any{
		"AppName": e.AppModule(),
		"Port":    fmt.Sprintf("%d", e.cfg.Port),
		"DbName":  dbDisplayName(e.cfg.Database),
		"Auth":    e.cfg.Features.Auth,
		"Models":  models,
	})
}

// renderEnv renders .env.example from the config.
func (e *Engine) renderEnv() (string, error) {
	dbPort, dbUser := "5432", "postgres"
	switch e.cfg.Database {
	case "mysql":
		dbPort, dbUser = "3306", "root"
	case "mongodb":
		dbPort, dbUser = "27017", "root"
	}
	return e.render("env", tmpl.EnvTemplate, map[string]any{
		"AppName":    e.AppModule(),
		"DBPort":     dbPort,
		"DBUser":     dbUser,
		"DBPassword": "password",
		"DbType":     e.cfg.Database,
		"Port":       fmt.Sprintf("%d", e.cfg.Port),
		"Auth":       e.cfg.Features.Auth,
		"Env":        e.cfg.Env,
	})
}

// sqlDBTemplate renders the SQL database connection file.
func sqlDBTemplate(data any, e *Engine) string {
	src := tmpl.SqlDBTemplate
	out, err := e.render("sqldb", src, data)
	if err != nil {
		return "// failed to render sql db template: " + err.Error()
	}
	return out
}

// dockerfileContent renders the Dockerfile.
func dockerfileContent(e *Engine) string {
	out, err := e.render("dockerfile", tmpl.DockerfileTemplate, map[string]any{
		"AppName": schema.Lower(e.AppModule()),
		"Port":    fmt.Sprintf("%d", e.cfg.Port),
	})
	if err != nil {
		return "// dockerfile render failed: " + err.Error()
	}
	return out
}

// dockerComposeContent renders docker-compose.yml.
func dockerComposeContent(e *Engine) string {
	dbPort := "5432"
	switch e.cfg.Database {
	case "mysql":
		dbPort = "3306"
	case "mongodb":
		dbPort = "27017"
	}
	out, err := e.render("compose", tmpl.DockerComposeTemplate, map[string]any{
		"AppName":    schema.Lower(e.AppModule()),
		"DbType":     e.cfg.Database,
		"DbHost":     "db",
		"DbPort":     dbPort,
		"DbUser":     dbUser(e.cfg.Database),
		"DbPassword": "password",
		"Port":       fmt.Sprintf("%d", e.cfg.Port),
		"Auth":       e.cfg.Features.Auth,
	})
	if err != nil {
		return "# docker-compose render failed: " + err.Error()
	}
	return out
}

// ciContent renders the GitHub Actions workflow.
func ciContent(e *Engine) string {
	out, err := e.render("ci", tmpl.CITemplate, map[string]any{
		"AppName": schema.Lower(e.AppModule()),
		"Docker":  e.cfg.Features.Docker,
	})
	if err != nil {
		return "# ci render failed: " + err.Error()
	}
	return out
}

// makefileContent renders the Makefile.
func makefileContent(e *Engine) string {
	out, err := e.render("makefile", tmpl.MakefileTemplate, map[string]any{
		"AppName":    schema.Lower(e.AppModule()),
		"Migrations": e.cfg.Features.Migrations,
		"Docker":     e.cfg.Features.Docker,
	})
	if err != nil {
		return "# makefile render failed: " + err.Error()
	}
	return out
}

func dbPort(db string) string {
	switch db {
	case "mysql":
		return "3306"
	case "mongodb":
		return "27017"
	default:
		return "5432"
	}
}

func dbUser(db string) string {
	if db == "mysql" || db == "mongodb" {
		return "root"
	}
	return "postgres"
}

func dbPassword(db string) string {
	if db == "mysql" || db == "mongodb" {
		return "secret"
	}
	return "postgres"
}

func dbDisplayName(db string) string {
	switch db {
	case "postgres":
		return "PostgreSQL"
	case "mysql":
		return "MySQL"
	case "mongodb":
		return "MongoDB"
	}
	return db
}
