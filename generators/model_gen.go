package generators

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	tmpl "github.com/v-pat/fiberforge/templates"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type modelType struct {
	StructName string
	Fields     []field
	DbType     string
}

// Field represents the structure of a field in a Go struct.
type field struct {
	Name           string
	TitlecasedName string
	Type           string
}

func inferGoType(value interface{}) string {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return "int64"
	case float32, float64:
		return "float64"
	case string:
		return "string"
	case bool:
		return "bool"
	case []interface{}:
		// If it's an array, infer the element type
		if len(value.([]interface{})) > 0 {
			return "[]" + inferGoType(value.([]interface{})[0])
		}
	}

	// Default to interface{} for unsupported types
	return "interface{}"
}

func GenerateStructFromJSON(jsonData map[string]interface{}, structName string, database string) (string, error) {
	// Initialize the struct code
	structCode := fmt.Sprintf("type %s struct {\n", structName)

	var structVar modelType

	structVar.StructName = cases.Title(language.English).String(structName)

	// Iterate through JSON fields and generate struct fields
	for fieldName, fieldValue := range jsonData {
		if strings.ToLower(fieldName) != "id" {
			fieldType := inferGoType(fieldValue)
			structField := fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, fieldName)
			structVar.Fields = append(structVar.Fields, field{
				Name:           fieldName,
				Type:           fieldType,
				TitlecasedName: cases.Title(language.English).String(fieldName),
			})
			structCode += structField
		}
	}

	structVar.DbType = database

	// Define the file path for the model file
	filePath := fmt.Sprintf(AppName+"/model/%s.go", strings.ToLower(structName))

	// Create the model file
	modelFile, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer modelFile.Close()

	// Parse the model template
	tmpl, err := template.New("model").Parse(tmpl.ModelTemplate)
	if err != nil {
		return "", err
	}

	// Execute the template and write the struct code to the model file
	if err := tmpl.Execute(modelFile, structVar); err != nil {
		return "", err
	}

	// Close the struct definition
	structCode += "}\n"

	return structCode, nil
}
