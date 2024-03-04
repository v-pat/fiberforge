package generators

import (
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func UpdateRoutesFile(structDefs []model.StructDefinition, database string, appName string) error {
	routesFilePath := AppName + "/routes/routes.go"

	// Open the routes.go file for appending
	routesFile, err := os.OpenFile(routesFilePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer routesFile.Close()

	// Parse the routes template
	tmpl, err := template.New("routes").Parse(tmpl.RoutesTemplate)
	if err != nil {
		return err
	}

	// Define data for the template
	type structname struct {
		StructName string
		Endpoint   string
	}

	type dataType struct {
		StructNames []structname
		AppName     string
	}
	data := dataType{}
	names := []structname{}

	for _, structDef := range structDefs {
		names = append(names, structname{StructName: structDef.StructName, Endpoint: cases.Lower(language.English).String(structDef.StructName)})
	}

	data.StructNames = names
	data.AppName = appName

	// Execute the template and write to the routes file
	if err := tmpl.Execute(routesFile, data); err != nil {
		return err
	}

	return nil
}
