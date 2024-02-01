package generators

import (
	"encoding/json"
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"

	"github.com/spf13/viper"
)

func CreateConfigJsonFile(appName string) error {

	//Get database config
	dbDetails := model.DbConfigDetails{}

	data, err := json.Marshal(viper.Get("sql_database"))
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &dbDetails)
	if err != nil {
		return err
	}

	dbDetails.Database = appName

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
