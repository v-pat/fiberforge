package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/v-pat/fiberforge/internal/schema"
)

// modelData builds the per-model data for model templates.
func (e *Engine) modelData(m schema.Model) modelModel {
	var fields []fieldModel
	hasUUID := false
	for _, f := range m.Fields {
		gt := goType(f)
		if e.isMongo() {
			gt = goTypeMongo(f)
		}
		if f.Type == schema.TypeUUID {
			hasUUID = true
		}
		jsonName := f.JSONTag
		if jsonName == "" {
			jsonName = f.Name
		}
		fm := fieldModel{
			GoName:    schema.Pascal(f.Name),
			GoType:    gt,
			JSONName:  jsonName,
			MongoName: f.Name,
		}
		if !e.isMongo() {
			fm.StructTag = buildStructTag(f, jsonName)
		}
		fields = append(fields, fm)
	}

	rels := e.relationshipData(m)
	hasFK := false
	// belongsTo relationships add an indexed FK column next to the association.
	for _, r := range rels {
		if r.Kind != string(schema.BelongsTo) {
			continue
		}
		hasFK = true
		var fk fieldModel
		if e.isMongo() {
			fk = fieldModel{
				GoName:    r.RefIDName,
				GoType:    "primitive.ObjectID",
				MongoName: r.RefVar + "Id",
				JSONName:  r.RefVar + "Id",
			}
		} else {
			fkCol := schema.Snake(r.GoName) + "_id"
			fk = fieldModel{
				GoName:    r.RefIDName,
				GoType:    "uint",
				JSONName:  r.RefVar + "Id",
				StructTag: "`gorm:\"column:" + fkCol + ";index\" json:\"" + r.RefVar + "Id\"`",
			}
		}
		fields = append(fields, fk)
	}

	return modelModel{
		Name:            schema.Pascal(m.Name),
		Fields:          fields,
		WritableFields:  e.writableFields(m),
		HasUUID:         hasUUID,
		HasTableName:    m.TableName != "",
		TableName:       m.TableName,
		QuotedTableName: strconv.Quote(m.TableName),
		HasFK:           hasFK,
		Relationships:   rels,
	}
}

func isWritableField(f schema.Field) bool {
	lower := strings.ToLower(f.Name)
	if lower == "id" || lower == "createdat" || lower == "created_at" ||
		lower == "updatedat" || lower == "updated_at" ||
		lower == "deletedat" || lower == "deleted_at" ||
		lower == "userid" || lower == "user_id" {
		return false
	}
	return true
}

// writableFields returns client-writable fields (excluding protected IDs, timestamps, FKs).
func (e *Engine) writableFields(m schema.Model) []fieldModel {
	var out []fieldModel
	for _, f := range m.Fields {
		if !isWritableField(f) {
			continue
		}
		gt := goType(f)
		if e.isMongo() {
			gt = goTypeMongo(f)
		}
		jsonName := f.JSONTag
		if jsonName == "" {
			jsonName = f.Name
		}
		fm := fieldModel{
			GoName:    schema.Pascal(f.Name),
			GoType:    gt,
			JSONName:  jsonName,
			MongoName: f.Name,
		}
		if !e.isMongo() {
			fm.StructTag = buildStructTag(f, jsonName)
		}
		out = append(out, fm)
	}
	return out
}

// relationshipData maps declared relationships to renderable association fields.
// belongsTo renders a single object + FK; hasMany and manyToMany render slices.
func (e *Engine) relationshipData(m schema.Model) []relationshipModel {
	var out []relationshipModel
	for _, r := range m.Relationships {
		target := schema.Pascal(r.Model)
		targetLower := schema.Camel(r.Model)
		pluralLower := schema.Lower(schema.Plural(target))
		pluralPascal := schema.Pascal(schema.Plural(target))

		rm := relationshipModel{Kind: string(r.Type), GoName: target}
		switch r.Type {
		case schema.BelongsTo:
			rm.RefIDName = target + "ID"
			rm.RefVar = targetLower
			rm.FieldName = target
			if e.isMongo() {
				rm.AssocType = "*" + target
				rm.Tag = "`bson:\"" + targetLower + ",omitempty\" json:\"" + targetLower + ",omitempty\"`"
			} else {
				rm.AssocType = target
				rm.Tag = "`gorm:\"constraint:OnUpdate:CASCADE,OnDelete:SET NULL\" json:\"" + targetLower + "\"`"
			}
		case schema.HasMany:
			rm.RefVar = pluralLower
			rm.FieldName = pluralPascal
			if e.isMongo() {
				rm.AssocType = "[]*" + target
				rm.Tag = "`bson:\"" + pluralLower + ",omitempty\" json:\"" + pluralLower + ",omitempty\"`"
			} else {
				rm.AssocType = "[]" + target
				rm.Tag = "`gorm:\"foreignKey:" + schema.Pascal(m.Name) + "ID\" json:\"" + pluralLower + "\"`"
			}
		case schema.ManyToMany:
			rm.RefVar = pluralLower
			rm.FieldName = pluralPascal
			if e.isMongo() {
				rm.AssocType = "[]*" + target
				rm.Tag = "`bson:\"" + pluralLower + ",omitempty\" json:\"" + pluralLower + ",omitempty\"`"
			} else {
				selfPlural := schema.Plural(m.Name)
				rm.AssocType = "[]" + target
				rm.Tag = "`gorm:\"many2many:" + schema.Lower(selfPlural) + "_" + schema.Lower(schema.Plural(target)) + ";\" json:\"" + pluralLower + "\"`"
			}
		default:
			continue
		}
		out = append(out, rm)
	}
	return out
}

// buildStructTag assembles a single gorm+json struct tag for a SQL field.
func buildStructTag(f schema.Field, jsonName string) string {
	cleanJSON := sanitizeTagValue(jsonName)
	var parts []string
	if f.Unique {
		parts = append(parts, "uniqueIndex")
	}
	if f.Index {
		parts = append(parts, "index")
	}
	if f.Required {
		parts = append(parts, "not null")
	}
	gorm := ""
	if len(parts) > 0 {
		gorm = `gorm:"` + joinTags(parts) + `" `
	}
	jsonTag := `json:"` + cleanJSON + `"`
	if f.Sensitive {
		jsonTag = `json:"-"`
	} else if f.OmitEmpty {
		jsonTag = `json:"` + cleanJSON + `,omitempty"`
	}

	var valRules []string
	if f.Required {
		valRules = append(valRules, "required")
	}
	if f.Validation != "" {
		valRules = append(valRules, sanitizeTagValue(f.Validation))
	}
	valTag := ""
	if len(valRules) > 0 {
		valTag = ` validate:"` + strings.Join(valRules, ",") + `"`
	}

	return "`" + gorm + jsonTag + valTag + "`"
}

func sanitizeTagValue(s string) string {
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// modelPascalNames returns the Pascal-cased model names for AutoMigrate.
func (e *Engine) modelPascalNames() []string {
	var out []string
	for _, m := range e.cfg.Models {
		out = append(out, schema.Pascal(m.Name))
	}
	return out
}

// dsnFormat returns the fmt.Sprintf format string for the SQL DSN builder.
func dsnFormat(driver string) string {
	if driver == "postgres" {
		return "postgres://%s:%s@%s:%s/%s?sslmode=disable"
	}
	return "%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local"
}

// quoteIdentifier wraps an identifier in dialect-appropriate quotes.
func quoteIdentifier(ident, driver string) string {
	if driver == "postgres" {
		return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
	}
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// formatDefault returns a safe SQL representation of the field's default value.
func formatDefault(f schema.Field, driver string) string {
	if f.Default == nil {
		return ""
	}
	raw := strings.TrimSpace(*f.Default)
	switch f.Type {
	case schema.TypeBool:
		if strings.EqualFold(raw, "true") || raw == "1" {
			if driver == "mysql" {
				return "1"
			}
			return "TRUE"
		}
		if strings.EqualFold(raw, "false") || raw == "0" {
			if driver == "mysql" {
				return "0"
			}
			return "FALSE"
		}
		return raw
	case schema.TypeInt, schema.TypeInt64, schema.TypeFloat:
		return raw
	case schema.TypeTime:
		return "CURRENT_TIMESTAMP"
	case schema.TypeString, schema.TypeText, schema.TypeEnum, schema.TypeUUID:
		escaped := strings.ReplaceAll(raw, "'", "''")
		return "'" + escaped + "'"
	default:
		return raw
	}
}

// buildMigration generates up/down SQL for a model on a given driver.
func buildMigration(m schema.Model, driver string) (up, down string) {
	table := m.TableName
	if table == "" {
		table = schema.Plural(m.Name)
	}
	quotedTable := quoteIdentifier(table, driver)
	var cols []string
	for _, f := range m.Fields {
		cols = append(cols, columnDDL(f, driver))
	}
	for _, r := range m.Relationships {
		if r.Type == schema.BelongsTo {
			fkCol := quoteIdentifier(schema.Snake(r.Model)+"_id", driver)
			cols = append(cols, "\t"+fkCol+" BIGINT")
		}
	}
	colStr := ",\n\t" + strings.Join(cols, ",\n\t")
	up = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n\t%s %s PRIMARY KEY%s\n);\n",
		quotedTable, quoteIdentifier("id", driver), idType(driver), colStr)
	down = fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", quotedTable)

	for _, r := range m.Relationships {
		if r.Type == schema.ManyToMany {
			selfTable := m.TableName
			if selfTable == "" {
				selfTable = schema.Plural(m.Name)
			}
			targetTable := schema.Plural(r.Model)
			joinTable := schema.Lower(selfTable) + "_" + schema.Lower(targetTable)
			quotedJoinTable := quoteIdentifier(joinTable, driver)
			selfFK := quoteIdentifier(schema.Camel(m.Name)+"Id", driver)
			targetFK := quoteIdentifier(schema.Camel(r.Model)+"Id", driver)
			up += fmt.Sprintf("\nCREATE TABLE IF NOT EXISTS %s (\n\t%s BIGINT NOT NULL,\n\t%s BIGINT NOT NULL,\n\tPRIMARY KEY (%s, %s)\n);\n",
				quotedJoinTable, selfFK, targetFK, selfFK, targetFK)
			down += fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", quotedJoinTable)
		}
	}
	return up, down
}

func columnDDL(f schema.Field, driver string) string {
	colType := columnType(f, driver)
	parts := []string{"\t" + quoteIdentifier(f.Name, driver) + " " + colType}
	if f.Required {
		parts = append(parts, "NOT NULL")
	}
	if f.Unique {
		parts = append(parts, "UNIQUE")
	}
	if f.Default != nil {
		defVal := formatDefault(f, driver)
		if defVal != "" {
			parts = append(parts, "DEFAULT "+defVal)
		}
	}
	return strings.Join(parts, " ")
}

func columnType(f schema.Field, driver string) string {
	switch f.Type {
	case schema.TypeString, schema.TypeUUID, schema.TypePassword:
		return "VARCHAR(255)"
	case schema.TypeText:
		return "TEXT"
	case schema.TypeInt:
		return intType(driver)
	case schema.TypeInt64:
		return bigintType(driver)
	case schema.TypeFloat:
		return "DOUBLE PRECISION"
	case schema.TypeBool:
		return "BOOLEAN"
	case schema.TypeTime:
		return "TIMESTAMP"
	case schema.TypeJSON:
		return "JSON"
	case schema.TypeEnum:
		return "VARCHAR(255)"
	default:
		return "VARCHAR(255)"
	}
}

func idType(driver string) string {
	if driver == "postgres" {
		return "BIGSERIAL"
	}
	return "BIGINT AUTO_INCREMENT"
}

func intType(driver string) string {
	if driver == "mysql" {
		return "INT"
	}
	return "INTEGER"
}

func bigintType(driver string) string {
	return "BIGINT"
}
