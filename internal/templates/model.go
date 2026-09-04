package templates

// ModelTemplate renders a GORM model file (postgres/mysql). Field struct tags
// are fully assembled (gorm + json) by the engine before rendering.
const ModelTemplate = `package model

import (
	"time"

	"gorm.io/gorm"
{{if .HasUUID}}	"github.com/google/uuid"
{{end}})
// {{.Name}} represents a {{.Name}} record.
type {{.Name}} struct {
	ID        uint           ` + "`gorm:\"primaryKey\" json:\"id\"`" + `
	CreatedAt time.Time      ` + "`json:\"createdAt\"`" + `
	UpdatedAt time.Time      ` + "`json:\"updatedAt\"`" + `
	DeletedAt gorm.DeletedAt ` + "`gorm:\"index\" json:\"-\"`" + `
{{range .Fields}}	{{.GoName}} {{.GoType}} {{.StructTag}}
{{end}}{{range .Relationships}}	{{.FieldName}} {{.AssocType}} {{.Tag}}
{{end}}}
{{if .HasTableName}}
// TableName returns the explicit table name.
func ({{.Name}}) TableName() string {
	return "{{.TableName}}"
}
{{end}}
`

// MongoModelTemplate renders a Mongo (mgm) model file with ObjectID.
const MongoModelTemplate = `package model

import (
	"github.com/kamva/mgm/v3"
{{if .HasUUID}}	"github.com/google/uuid"
{{end}}{{if .HasFK}}	"go.mongodb.org/mongo-driver/bson/primitive"
{{end}})
// {{.Name}} represents a {{.Name}} document.
type {{.Name}} struct {
	mgm.DefaultModel ` + "`bson:\",inline\"`" + `
{{range .Fields}}	{{.GoName}} {{.GoType}} ` + "`bson:\"{{.MongoName}}\" json:\"{{.JSONName}}\"`" + `
{{end}}{{range .Relationships}}	{{.FieldName}} {{.AssocType}} {{.Tag}}
{{end}}}
{{if .HasTableName}}
// CollectionName returns the explicit collection name.
func ({{.Name}}) CollectionName() string {
	return "{{.TableName}}"
}
{{end}}
`
