package introspect

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/v-pat/fiberforge/internal/schema"
)

var (
	createTableRe = regexp.MustCompile(`(?is)CREATE\s+TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+["` + "`" + `]?([a-zA-Z0-9_]+)["` + "`" + `]?\s*\((.*?)\);`)
	fkConstraintRe = regexp.MustCompile(`(?is)FOREIGN\s+KEY\s*\(\s*["` + "`" + `]?([a-zA-Z0-9_]+)["` + "`" + `]?\s*\)\s*REFERENCES\s+["` + "`" + `]?([a-zA-Z0-9_]+)["` + "`" + `]?`)
)

// IntrospectSQL parses raw SQL DDL (e.g. CREATE TABLE statements) into a FiberForge Config.
func IntrospectSQL(ddlText string, appName string, database string) (*schema.Config, error) {
	if appName == "" {
		appName = "scaffolded-api"
	}
	if database == "" {
		database = "postgres"
	}

	matches := createTableRe.FindAllStringSubmatch(ddlText, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no CREATE TABLE statements found in SQL input")
	}

	models := make([]schema.Model, 0, len(matches))

	for _, m := range matches {
		rawTableName := m[1]
		body := m[2]

		lowerTable := strings.ToLower(rawTableName)
		if lowerTable == "schema_migrations" || lowerTable == "migrations" {
			continue
		}

		model, err := parseTableBody(rawTableName, body)
		if err != nil {
			return nil, err
		}
		models = append(models, *model)
	}

	cfg := &schema.Config{
		AppName:  appName,
		Database: database,
		Port:     8080,
		Features: schema.Features{
			Auth:       detectHasAuth(models),
			Docker:     true,
			Migrations: true,
			Swagger:    true,
			RateLimit:  true,
			CORS:       true,
			Logging:    true,
			Testing:    true,
			CI:         true,
		},
		Models: models,
	}

	return cfg, nil
}

func parseTableBody(tableName, body string) (*schema.Model, error) {
	// First extract table-level foreign keys
	fkMatches := fkConstraintRe.FindAllStringSubmatch(body, -1)
	fkColumns := make(map[string]string)
	var rels []schema.Relationship

	for _, fk := range fkMatches {
		col := strings.ToLower(fk[1])
		foreignTable := fk[2]
		fkColumns[col] = foreignTable
		rels = append(rels, schema.Relationship{
			Type:  schema.BelongsTo,
			Model: TableToModelName(foreignTable),
		})
	}

	// Split column definitions by comma, taking into account nested parens
	colDefs := splitSQLDefinitions(body)
	var fields []schema.Field

	for _, def := range colDefs {
		line := strings.TrimSpace(def)
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "PRIMARY KEY") ||
			strings.HasPrefix(upper, "FOREIGN KEY") ||
			strings.HasPrefix(upper, "CONSTRAINT") ||
			strings.HasPrefix(upper, "UNIQUE") ||
			strings.HasPrefix(upper, "KEY ") ||
			strings.HasPrefix(upper, "INDEX ") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		rawColName := strings.Trim(parts[0], "`\"[]")
		rawType := parts[1]

		if isStandardGORMColumn(rawColName) {
			continue
		}

		if _, isFK := fkColumns[strings.ToLower(rawColName)]; isFK {
			continue
		}

		fieldType := mapSQLType(rawType, nil)
		fieldName := ColumnToFieldName(rawColName)
		required := strings.Contains(upper, "NOT NULL")
		unique := strings.Contains(upper, "UNIQUE")

		f := schema.Field{
			Name:     fieldName,
			Type:     fieldType,
			Required: required,
			Unique:   unique,
		}

		// Check for inline default
		if idx := strings.Index(upper, "DEFAULT "); idx != -1 {
			rem := line[idx+len("DEFAULT "):]
			defTokens := strings.Fields(rem)
			if len(defTokens) > 0 {
				defVal := strings.Trim(defTokens[0], "'\";,")
				if defVal != "" && !strings.EqualFold(defVal, "NULL") && !strings.HasPrefix(defVal, "nextval") {
					f.Default = &defVal
				}
			}
		}

		lowerF := strings.ToLower(fieldName)
		if strings.Contains(lowerF, "password") || strings.Contains(lowerF, "token") || strings.Contains(lowerF, "secret") {
			if strings.Contains(lowerF, "password") {
				f.Type = schema.TypePassword
			}
			f.Sensitive = true
		}

		fields = append(fields, f)
	}

	modelName := TableToModelName(tableName)
	endpoint := strings.ToLower(tableName)
	endpoint = strings.ReplaceAll(endpoint, "_", "-")

	return &schema.Model{
		Name:          modelName,
		Endpoint:      endpoint,
		Fields:        fields,
		Relationships: rels,
	}, nil
}

func splitSQLDefinitions(body string) []string {
	var defs []string
	var current strings.Builder
	parenDepth := 0

	for _, ch := range body {
		switch ch {
		case '(':
			parenDepth++
			current.WriteRune(ch)
		case ')':
			parenDepth--
			current.WriteRune(ch)
		case ',':
			if parenDepth == 0 {
				defs = append(defs, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		defs = append(defs, current.String())
	}
	return defs
}
