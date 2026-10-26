package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/v-pat/fiberforge/internal/schema"
)

func introspectMySQL(ctx context.Context, db *sql.DB) ([]schema.Model, error) {
	// Query user tables in current database
	tableQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE() 
		  AND table_type = 'BASE TABLE'
		  AND table_name NOT IN ('schema_migrations', 'migrations')
		ORDER BY table_name;`

	rows, err := db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query mysql tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	models := make([]schema.Model, 0, len(tableNames))
	for _, tableName := range tableNames {
		model, err := introspectMySQLTable(ctx, db, tableName)
		if err != nil {
			return nil, err
		}
		models = append(models, *model)
	}

	return models, nil
}

func introspectMySQLTable(ctx context.Context, db *sql.DB, tableName string) (*schema.Model, error) {
	colQuery := `
		SELECT 
			column_name, 
			data_type, 
			is_nullable, 
			column_default, 
			column_key,
			character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position;`

	colRows, err := db.QueryContext(ctx, colQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for table %s: %w", tableName, err)
	}
	defer colRows.Close()

	// Query foreign keys
	fkQuery := `
		SELECT
			column_name,
			referenced_table_name
		FROM information_schema.key_column_usage
		WHERE table_schema = DATABASE()
		  AND table_name = ?
		  AND referenced_table_name IS NOT NULL;`

	var rels []schema.Relationship
	fkColumns := make(map[string]string)
	if fkRows, err := db.QueryContext(ctx, fkQuery, tableName); err == nil {
		for fkRows.Next() {
			var col, foreignTable string
			if err := fkRows.Scan(&col, &foreignTable); err == nil {
				fkColumns[col] = foreignTable
				rels = append(rels, schema.Relationship{
					Type:  schema.BelongsTo,
					Model: TableToModelName(foreignTable),
				})
			}
		}
		fkRows.Close()
	}

	modelName := TableToModelName(tableName)
	var fields []schema.Field

	for colRows.Next() {
		var colName, dataType, isNullable, columnKey string
		var colDefault sql.NullString
		var charMaxLen sql.NullInt64

		if err := colRows.Scan(&colName, &dataType, &isNullable, &colDefault, &columnKey, &charMaxLen); err != nil {
			return nil, err
		}

		if isStandardGORMColumn(colName) {
			continue
		}

		if _, isFK := fkColumns[colName]; isFK {
			continue
		}

		var maxLenPtr *int
		if charMaxLen.Valid {
			ml := int(charMaxLen.Int64)
			maxLenPtr = &ml
		}

		fieldType := mapSQLType(dataType, maxLenPtr)
		fieldName := ColumnToFieldName(colName)
		required := (isNullable == "NO")
		unique := (columnKey == "UNI")

		f := schema.Field{
			Name:     fieldName,
			Type:     fieldType,
			Required: required,
			Unique:   unique,
		}

		if colDefault.Valid {
			defVal := strings.Trim(colDefault.String, "'")
			if defVal != "" && !strings.EqualFold(defVal, "null") {
				f.Default = &defVal
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

	if err := colRows.Err(); err != nil {
		return nil, err
	}

	endpoint := strings.ToLower(tableName)
	endpoint = strings.ReplaceAll(endpoint, "_", "-")

	return &schema.Model{
		Name:          modelName,
		Endpoint:      endpoint,
		Fields:        fields,
		Relationships: rels,
	}, nil
}
