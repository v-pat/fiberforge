package generators

import (
	"log"
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// DatabaseConnectionParams represents the parameters required for generating database connection code.
type DatabaseConnectionParams struct {
	DatabaseDriver     string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	DBURLFormat        string
	DatabaseDriverName string
	StructNames        []model.StructDefinition
	AppName            string
}

// GenerateDatabaseConnectionCode generates code for connecting to a database (PostgreSQL or MySQL) and writes it to a file.
func generateDatabaseConnectionCode(params DatabaseConnectionParams, fileName string) error {
	// Create a new template
	tmpl, err := template.New("databaseConnection").Parse(tmpl.DatabaseConnectionTemplate)
	if err != nil {
		return err
	}

	// Create or open the file
	file, err := os.Create("generated/databases/" + fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Execute the template and write the generated code to the file
	if err := tmpl.Execute(file, params); err != nil {
		return err
	}

	return nil
}

func CreateDatabase(database string, dbName string, structDefs []model.StructDefinition, appName string) error {
	for i, structDef := range structDefs {
		structDefs[i].StructName = cases.Title(language.English).String(structDef.StructName)
	}

	params := DatabaseConnectionParams{
		DatabaseDriver:     "github.com/lib/pq",
		DBHost:             "localhost",
		DBPort:             "5432",
		DBName:             dbName,
		DBUser:             "myuser",
		DBPassword:         "mypassword",
		DBURLFormat:        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
		DatabaseDriverName: "postgres",
		StructNames:        structDefs,
		AppName:            appName,
	}

	if database == "postgres" {
		// Example usage: Generate code for PostgreSQL database connection and write to a file
		err := generateDatabaseConnectionCode(params, "postgres_connection.go")
		if err != nil {
			log.Println("Error:", err)
			return err
		}
	} else if database == "mysql" {
		// Example usage: Generate code for MySQL database connection and write to a file
		params.DatabaseDriver = "github.com/go-sql-driver/mysql"
		params.DBPort = "3306"
		params.DatabaseDriverName = "mysql"
		params.DBUser = "root"
		params.DBPassword = "root"
		params.DBURLFormat = "%s:%s@tcp(%s:%s)/%s"
		params.DBName = dbName
		params.StructNames = structDefs
		params.AppName = appName

		err := generateDatabaseConnectionCode(params, "mysql_connection.go")
		if err != nil {
			log.Println("Error:", err)
			return err
		}

	}
	return nil
}
