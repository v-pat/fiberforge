package templates

// Define a model template for generating the struct code in the model file.
const ModelTemplate = `package model



import {{if eq .DbType "mongodb"}}"github.com/kamva/mgm/v3"{{else}}"gorm.io/gorm"{{end}}

// {{.StructName}} represents the {{.StructName}} struct.
type {{.StructName}} struct {
	{{if eq .DbType "mongodb"}}mgm.DefaultModel ` + "`bson:\",inline\"`" + `{{else}}gorm.Model{{end}}
{{range .Fields}}
	{{.TitlecasedName}} {{.Type}} ` + "`json:\"{{.Name}}\"`" +
	`{{end}}
}`
