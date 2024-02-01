package generators

import (
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"
)

func CreateConfigJsonFile(appName string) error {

	//Get database config
	dbDetails := model.DbConfigDetails{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "root",
		Database: appName,
	}

	// Create a new template
	tmpl, err := template.New("envConfig").Parse(tmpl.EnvConfigTemplate)
	if err != nil {
		return err
	}

	// Create or open the file
	file, err := os.Create("generated/config.json")
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
