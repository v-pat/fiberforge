package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"

	"github.com/v-pat/fiberforge/internal/schema"
)

// Options specifies configuration for reverse introspection.
type Options struct {
	AppName  string
	Database string // "postgres" | "mysql"
	DSN      string
	Schema   string // optional database schema (e.g. "public")
}

// IntrospectDatabase connects to a live database and reverse-engineers a FiberForge schema.
func IntrospectDatabase(ctx context.Context, opts Options) (*schema.Config, error) {
	if opts.AppName == "" {
		opts.AppName = "scaffolded-api"
	}
	dbType := strings.ToLower(opts.Database)
	if dbType == "" {
		if strings.HasPrefix(opts.DSN, "postgres://") || strings.HasPrefix(opts.DSN, "postgresql://") {
			dbType = "postgres"
		} else if strings.Contains(opts.DSN, "tcp(") || strings.HasPrefix(opts.DSN, "mysql://") {
			dbType = "mysql"
		} else {
			dbType = "postgres"
		}
	}

	var driverName string
	switch dbType {
	case "postgres":
		driverName = "postgres"
	case "mysql":
		driverName = "mysql"
		// Ensure parseTime=true is set for MySQL
		if !strings.Contains(opts.DSN, "parseTime=") {
			if strings.Contains(opts.DSN, "?") {
				opts.DSN += "&parseTime=true"
			} else {
				opts.DSN += "?parseTime=true"
			}
		}
	default:
		return nil, fmt.Errorf("unsupported database driver for introspection: %s (supported: postgres, mysql)", dbType)
	}

	db, err := sql.Open(driverName, opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	var models []schema.Model
	switch dbType {
	case "postgres":
		models, err = introspectPostgres(ctx, db, opts.Schema)
	case "mysql":
		models, err = introspectMySQL(ctx, db)
	}
	if err != nil {
		return nil, err
	}

	cfg := &schema.Config{
		AppName:  opts.AppName,
		Database: dbType,
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

// ToYAML serializes the schema configuration to clean YAML.
func ToYAML(cfg *schema.Config) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema to YAML: %w", err)
	}
	return string(data), nil
}

// TableToModelName converts a snake_case or plural table name to PascalCase singular.
func TableToModelName(tableName string) string {
	parts := strings.Split(tableName, "_")
	var sb strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		sb.WriteString(string(runes))
	}
	name := sb.String()
	return singularize(name)
}

// ColumnToFieldName converts column_name to camelCase.
func ColumnToFieldName(colName string) string {
	parts := strings.Split(colName, "_")
	var sb strings.Builder
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		if i == 0 {
			runes[0] = unicode.ToLower(runes[0])
		} else {
			runes[0] = unicode.ToUpper(runes[0])
		}
		sb.WriteString(string(runes))
	}
	return sb.String()
}

func singularize(s string) string {
	if strings.HasSuffix(s, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") && len(s) > 3 {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") && len(s) > 1 {
		return s[:len(s)-1]
	}
	return s
}

func mapSQLType(dataType string, charMaxLen *int) schema.FieldType {
	dt := strings.ToLower(dataType)
	switch {
	case strings.Contains(dt, "int8"), strings.Contains(dt, "bigint"), strings.Contains(dt, "bigserial"):
		return schema.TypeInt64
	case strings.Contains(dt, "int"), strings.Contains(dt, "serial"):
		return schema.TypeInt
	case strings.Contains(dt, "bool"):
		return schema.TypeBool
	case strings.Contains(dt, "float"), strings.Contains(dt, "double"), strings.Contains(dt, "numeric"), strings.Contains(dt, "decimal"), strings.Contains(dt, "real"):
		return schema.TypeFloat
	case strings.Contains(dt, "uuid"):
		return schema.TypeUUID
	case strings.Contains(dt, "json"):
		return schema.TypeJSON
	case strings.Contains(dt, "time"), strings.Contains(dt, "date"):
		return schema.TypeTime
	case strings.Contains(dt, "text"):
		return schema.TypeText
	case strings.Contains(dt, "char"):
		if charMaxLen != nil && *charMaxLen > 255 {
			return schema.TypeText
		}
		return schema.TypeString
	default:
		return schema.TypeString
	}
}

func isStandardGORMColumn(colName string) bool {
	lower := strings.ToLower(colName)
	switch lower {
	case "id", "created_at", "updated_at", "deleted_at":
		return true
	default:
		return false
	}
}

func detectHasAuth(models []schema.Model) bool {
	for _, m := range models {
		lower := strings.ToLower(m.Name)
		if lower == "user" || lower == "account" {
			for _, f := range m.Fields {
				if strings.ToLower(f.Name) == "password" || strings.ToLower(f.Name) == "passwordhash" {
					return true
				}
			}
		}
	}
	return false
}
