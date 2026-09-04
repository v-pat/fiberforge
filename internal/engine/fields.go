package engine

import (
	"github.com/v-pat/fiberforge/internal/schema"
)

// fieldModel is the per-field data passed to model templates.
type fieldModel struct {
	GoName    string // exported Go field name
	GoType    string // Go type
	JSONName  string // json tag
	MongoName string // mongo bson tag
	StructTag string // fully assembled struct tag (backtick-quoted)
	HasUUID   bool   // whether this field is a UUID type
}

// modelModel is the per-model data passed to model templates.
type modelModel struct {
	Name          string // Pascal model name
	Fields        []fieldModel
	HasUUID       bool
	HasTableName  bool   // tableName/collection declared in schema
	TableName     string // explicit table/collection name override
	HasFK         bool   // any belongsTo relationship (mongo needs primitive.ObjectID)
	Relationships []relationshipModel
}

// relationshipModel is a rendered belongsTo/hasMany/manyToMany relationship.
type relationshipModel struct {
	Kind      string // belongsTo | hasMany | manyToMany
	GoName    string // Pascal name of the target model
	RefIDName string // Pascal name of the FK field, e.g. UserID
	RefVar    string // camel name of the association's json key, e.g. user
	FieldName string // Go association field name, e.g. User or Posts
	AssocType string // Go type of the association field
	Tag       string // gorm or bson struct tag for the association field
}

// routeGroup is the per-model CRUD group passed to the routes template.
type routeGroup struct {
	Name string
	Path string
	Var  string
	Auth bool
}

// swaggerModel is passed to the swagger template.
type swaggerModel struct {
	Name     string
	Endpoint string
	Last     bool
}

// goType maps a logical FieldType to its Go representation for SQL databases.
func goType(f schema.Field) string {
	switch f.Type {
	case schema.TypeString, schema.TypeText, schema.TypePassword:
		return "string"
	case schema.TypeUUID:
		return "uuid.UUID"
	case schema.TypeInt:
		return "int"
	case schema.TypeInt64:
		return "int64"
	case schema.TypeFloat:
		return "float64"
	case schema.TypeBool:
		return "bool"
	case schema.TypeTime:
		return "time.Time"
	case schema.TypeJSON:
		return "map[string]interface{}"
	case schema.TypeEnum:
		return "string"
	default:
		return "string"
	}
}

// goTypeMongo maps a logical FieldType to Go for MongoDB.
func goTypeMongo(f schema.Field) string {
	switch f.Type {
	case schema.TypeInt, schema.TypeInt64:
		return "int64"
	case schema.TypeFloat:
		return "float64"
	default:
		return goType(f)
	}
}

func joinTags(tags []string) string {
	out := ""
	for i, t := range tags {
		if i > 0 {
			out += ";"
		}
		out += t
	}
	return out
}
