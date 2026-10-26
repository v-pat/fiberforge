package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/v-pat/fiberforge/internal/schema"
)

func introspectPostgres(ctx context.Context, db *sql.DB, schemaName string) ([]schema.Model, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	// 1. Query user tables
	tableQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		  AND table_type = 'BASE TABLE'
		  AND table_name NOT IN ('schema_migrations', 'migrations')
		ORDER BY table_name;`

	rows, err := db.QueryContext(ctx, tableQuery, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to query postgres tables: %w", err)
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
		model, err := introspectPostgresTable(ctx, db, schemaName, tableName)
		if err != nil {
			return nil, err
		}
		models = append(models, *model)
	}

	return models, nil
}

func introspectPostgresTable(ctx context.Context, db *sql.DB, schemaName, tableName string) (*schema.Model, error) {
	// Query columns
	colQuery := `
		SELECT 
			column_name, 
			data_type, 
			is_nullable, 
			column_default, 
			character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position;`

	colRows, err := db.QueryContext(ctx, colQuery, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for table %s: %w", tableName, err)
	}
	defer colRows.Close()

	// Query unique columns
	uniqueCols := make(map[string]bool)
	uniqueQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = $1 
		  AND tc.table_name = $2
		  AND tc.constraint_type = 'UNIQUE';`

	if uRows, err := db.QueryContext(ctx, uniqueQuery, schemaName, tableName); err == nil {
		for uRows.Next() {
			var col string
			if err := uRows.Scan(&col); err == nil {
				uniqueCols[col] = true
			}
		}
		uRows.Close()
	}

	// Query foreign keys
	fkQuery := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
		  ON ccu.constraint_name = tc.constraint_name
		  AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = $2;`

	var rels []schema.Relationship
	fkColumns := make(map[string]string)
	if fkRows, err := db.QueryContext(ctx, fkQuery, schemaName, tableName); err == nil {
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
		var colName, dataType, isNullable string
		var colDefault sql.NullString
		var charMaxLen sql.NullInt64

		if err := colRows.Scan(&colName, &dataType, &isNullable, &colDefault, &charMaxLen); err != nil {
			return nil, err
		}

		// Skip primary ID or standard GORM metadata timestamps from explicit field list
		if isStandardGORMColumn(colName) {
			continue
		}

		// If this column is a foreign key (e.g. user_id), GORM will manage it via BelongsTo relationship
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
		unique := uniqueCols[colName]

		f := schema.Field{
			Name:     fieldName,
			Type:     fieldType,
			Required: required,
			Unique:   unique,
		}

		if colDefault.Valid && !strings.HasPrefix(colDefault.String, "nextval") {
			defVal := cleanSQLDefault(colDefault.String)
			if defVal != "" {
				f.Default = &defVal
			}
		}

		// Check sensitive field heuristics
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

func cleanSQLDefault(def string) string {
	d := strings.TrimSpace(def)
	// PostgreSQL defaults like 'foo'::character varying
	if idx := strings.Index(d, "::"); idx != -1 {
		d = d[:idx]
	}
	d = strings.Trim(d, "'")
	return d
}
