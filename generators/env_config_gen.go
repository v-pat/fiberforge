package generators

import (
	"os"
	"strings"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"
)

func CreateConfigJsonFile(appName string, database string) error {

	//Get database config
	var dbDetails model.DbConfigDetails
	if strings.ToLower(database) == "mysql" {
		dbDetails = model.DbConfigDetails{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "root",
			Database: appName,
			DbType:   strings.ToLower(database),
		}
	} else if strings.ToLower(database) == "postgres" {
		dbDetails = model.DbConfigDetails{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: appName,
			DbType:   strings.ToLower(database),
		}
	} else if strings.ToLower(database) == "mongodb" {
		dbDetails = model.DbConfigDetails{
			Host:     "localhost",
			Port:     27017,
			User:     "root",
			Password: "root",
			Database: appName,
			DbType:   strings.ToLower(database),
		}
	}

	// Create a new template
	tmpl, err := template.New("envConfig").Parse(tmpl.EnvConfigTemplate)
	if err != nil {
		return err
	}

	// Create or open the file
	file, err := os.Create(appName + "/config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	// Execute the template and write the generated code to the file
	if err := tmpl.Execute(file, dbDetails); err != nil {
		return err
	}

	return nil

}
