package templates

// Define a model template for generating the struct code in the model file.
const ModelTemplate = `package model

import "gorm.io/gorm"

// {{.StructName}} represents the {{.StructName}} struct.
type {{.StructName}} struct {
	gorm.Model
{{range .Fields}}
	{{.TitlecasedName}} {{.Type}} ` + "`json:\"{{.Name}}\"`" +
	`{{end}}
}`
