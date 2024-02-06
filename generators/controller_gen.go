package generators

import (
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func GenerateControllerFile(fileName string, structDef model.StructDefinition, appName string, database string) error {
	controllerFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer controllerFile.Close()

	// Parse the controller template
	tmpl, err := template.New("controller").Parse(tmpl.ControllerTemplate)
	if err != nil {
		return err
	}

	// Define data for the template
	data := struct {
		StructName          string
		AppName             string
		StructNameTitleCase string
		DbType              string
	}{
		StructName:          structDef.StructName,
		AppName:             appName,
		StructNameTitleCase: cases.Title(language.English).String(structDef.StructName),
		DbType:              database,
	}

	// Execute the template and write to the controller file
	if err := tmpl.Execute(controllerFile, data); err != nil {
		return err
	}

	return nil
}
