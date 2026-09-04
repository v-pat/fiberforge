package schema

// Config describes a generated project. It is the single input contract shared
// by the CLI scaffolder and the MCP server. It can be expressed in YAML or JSON.
type Config struct {
	AppName   string         `yaml:"appName" json:"appName"`
	Framework string         `yaml:"framework" json:"framework"`
	Database  string         `yaml:"database" json:"database"`
	Port      int            `yaml:"port" json:"port"`
	Features  Features       `yaml:"features" json:"features"`
	Models    []Model        `yaml:"models" json:"models"`
	OutputDir string         `yaml:"outputDir" json:"outputDir,omitempty"`
	Env       map[string]any `yaml:"env" json:"env,omitempty"`
}

// Features are opt-in capabilities generated alongside the CRUD scaffold.
type Features struct {
	Auth       bool `yaml:"auth" json:"auth"`
	Docker     bool `yaml:"docker" json:"docker"`
	Migrations bool `yaml:"migrations" json:"migrations"`
	Swagger    bool `yaml:"swagger" json:"swagger"`
	RateLimit  bool `yaml:"rateLimit" json:"rateLimit"`
	CORS       bool `yaml:"cors" json:"cors"`
	Logging    bool `yaml:"logging" json:"logging"`
	Testing    bool `yaml:"testing" json:"testing"`
	CI         bool `yaml:"ci" json:"ci"`
}

// RelationshipKind is the type of a model relationship.
type RelationshipKind string

const (
	BelongsTo  RelationshipKind = "belongsTo"
	HasMany    RelationshipKind = "hasMany"
	ManyToMany RelationshipKind = "manyToMany"
)

// Relationship describes a link between two models.
type Relationship struct {
	Type    RelationshipKind `yaml:"type" json:"type"`
	Model   string           `yaml:"model" json:"model"`
	Through string           `yaml:"through,omitempty" json:"through,omitempty"`
}

// FieldType is the set of supported logical field types.
type FieldType string

const (
	TypeString   FieldType = "string"
	TypeText     FieldType = "text"
	TypeInt      FieldType = "int"
	TypeInt64    FieldType = "int64"
	TypeFloat    FieldType = "float"
	TypeBool     FieldType = "bool"
	TypeTime     FieldType = "time"
	TypeUUID     FieldType = "uuid"
	TypeJSON     FieldType = "json"
	TypeEnum     FieldType = "enum"
	TypePassword FieldType = "password"
)

// Field describes a single column/field on a model.
type Field struct {
	Name       string    `yaml:"name" json:"name"`
	Type       FieldType `yaml:"type" json:"type"`
	Required   bool      `yaml:"required" json:"required"`
	Unique     bool      `yaml:"unique" json:"unique"`
	Default    *string   `yaml:"default,omitempty" json:"default,omitempty"`
	Validation string    `yaml:"validation,omitempty" json:"validation,omitempty"`
	Sensitive  bool      `yaml:"sensitive" json:"sensitive"`
	Index      bool      `yaml:"index" json:"index"`
	JSONTag    string    `yaml:"jsonTag,omitempty" json:"jsonTag,omitempty"`
	OmitEmpty  bool      `yaml:"omitempty" json:"omitempty"`
	Values     []string  `yaml:"values,omitempty" json:"values,omitempty"` // for enum
}

// Model describes a resource to scaffold CRUD for.
type Model struct {
	Name          string         `yaml:"name" json:"name"`
	Endpoint      string         `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Fields        []Field        `yaml:"fields" json:"fields"`
	Relationships []Relationship `yaml:"relationships,omitempty" json:"relationships,omitempty"`
	AuthProtected bool           `yaml:"auth" json:"auth"`
	TableName     string         `yaml:"tableName,omitempty" json:"tableName,omitempty"`
}
