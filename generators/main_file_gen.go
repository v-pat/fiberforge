package generators

import (
	"os"
	"text/template"

	"github.com/v-pat/fiberforge/model"
	tmpl "github.com/v-pat/fiberforge/templates"
)

func CreateMainFile(structDefs []model.StructDefinition, database string, appName string) error {
	mainFilePath := appName + "/main.go"

	// Open the main.go file for appending
	mainFile, err := os.OpenFile(mainFilePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer mainFile.Close()

	// Parse the routes template
	tmpl, err := template.New("main").Parse(tmpl.MainTemplate)
	if err != nil {
		return err
	}

	// Define data for the template
	type structname struct {
		StructName string
	}

	type dataType struct {
		StructNames []structname
		AppName     string
	}
	data := dataType{}
	names := []structname{}

	for _, structDef := range structDefs {
		names = append(names, structname{StructName: structDef.StructName})
	}

	data.StructNames = names
	data.AppName = appName

	// Execute the template and write to the routes file
	if err := tmpl.Execute(mainFile, data); err != nil {
		return err
	}

	return nil
}
