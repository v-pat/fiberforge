package generators

import (
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func GenerateServiceFile(fileName string, structDef model.StructDefinition, structCode, database string, appName string) error {

	serviceFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer serviceFile.Close()

	// Parse the service template
	tmpl, err := template.New("service").Parse(tmpl.ServiceTemplate)
	if err != nil {
		return err
	}

	// Define data for the template
	data := struct {
		AppName             string
		StructName          string
		StructNameTitlecase string
		Database            string
		DBName              string
		StructCode          string
	}{
		AppName:             appName,
		StructName:          structDef.StructName,
		StructNameTitlecase: cases.Title(language.English).String(structDef.StructName),
		Database:            database,
		StructCode:          structCode,
		DBName:              cases.Title(language.English).String(appName),
	}

	// Execute the template and write to the service file
	if err := tmpl.Execute(serviceFile, data); err != nil {
		return err
	}

	return nil
}
