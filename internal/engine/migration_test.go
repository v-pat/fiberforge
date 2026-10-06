package engine

import (
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestBuildMigration(t *testing.T) {
	m := schema.Model{
		Name: "Post",
		Fields: []schema.Field{
			{Name: "title", Type: schema.TypeString, Required: true},
			{Name: "views", Type: schema.TypeInt},
		},
		Relationships: []schema.Relationship{
			{Type: schema.BelongsTo, Model: "User"},
			{Type: schema.ManyToMany, Model: "Tag"},
		},
	}

	upPg, downPg := buildMigration(m, "postgres")
	if !strings.Contains(upPg, `CREATE TABLE IF NOT EXISTS "posts"`) {
		t.Errorf("expected table creation, got:\n%s", upPg)
	}
	if !strings.Contains(upPg, `"user_id" BIGINT`) {
		t.Errorf("expected FK column user_id, got:\n%s", upPg)
	}
	if !strings.Contains(upPg, `CREATE TABLE IF NOT EXISTS "posts_tags"`) {
		t.Errorf("expected join table posts_tags, got:\n%s", upPg)
	}
	if !strings.Contains(downPg, `DROP TABLE IF EXISTS "posts";`) {
		t.Errorf("expected drop table, got:\n%s", downPg)
	}

	upMy, _ := buildMigration(m, "mysql")
	if !strings.Contains(upMy, "BIGINT AUTO_INCREMENT PRIMARY KEY") {
		t.Errorf("expected MySQL primary key syntax, got:\n%s", upMy)
	}
	if !strings.Contains(upMy, "CREATE TABLE IF NOT EXISTS `posts`") {
		t.Errorf("expected MySQL table creation with backticks, got:\n%s", upMy)
	}
}

func TestBuildMigration_SafeDefaultFormatting(t *testing.T) {
	defFalse := "false"
	defVal := "usd"
	defNum := "42"
	m := schema.Model{
		Name: "Item",
		Fields: []schema.Field{
			{Name: "active", Type: schema.TypeBool, Default: &defFalse},
			{Name: "currency", Type: schema.TypeString, Default: &defVal},
			{Name: "count", Type: schema.TypeInt, Default: &defNum},
		},
	}

	upPg, _ := buildMigration(m, "postgres")
	if !strings.Contains(upPg, `DEFAULT FALSE`) {
		t.Errorf("expected DEFAULT FALSE in postgres, got:\n%s", upPg)
	}
	if !strings.Contains(upPg, `DEFAULT 'usd'`) {
		t.Errorf("expected quoted string DEFAULT 'usd' in postgres, got:\n%s", upPg)
	}
	if !strings.Contains(upPg, `DEFAULT 42`) {
		t.Errorf("expected numeric DEFAULT 42 in postgres, got:\n%s", upPg)
	}

	upMy, _ := buildMigration(m, "mysql")
	if !strings.Contains(upMy, `DEFAULT 0`) {
		t.Errorf("expected DEFAULT 0 in mysql, got:\n%s", upMy)
	}
}

func TestBuildMigration_QuotedIdentifiers(t *testing.T) {
	m := schema.Model{
		Name:      "user_account",
		TableName: "custom_users",
		Fields: []schema.Field{
			{Name: "order", Type: schema.TypeInt}, // "order" is a reserved SQL keyword
		},
	}

	upPg, downPg := buildMigration(m, "postgres")
	if !strings.Contains(upPg, `CREATE TABLE IF NOT EXISTS "custom_users"`) {
		t.Errorf("expected quoted table name in postgres: %s", upPg)
	}
	if !strings.Contains(upPg, `"order" INTEGER`) {
		t.Errorf("expected quoted reserved keyword column in postgres: %s", upPg)
	}
	if !strings.Contains(downPg, `DROP TABLE IF EXISTS "custom_users";`) {
		t.Errorf("expected quoted table name in down migration: %s", downPg)
	}

	upMy, downMy := buildMigration(m, "mysql")
	if !strings.Contains(upMy, "CREATE TABLE IF NOT EXISTS `custom_users`") {
		t.Errorf("expected quoted table name in mysql: %s", upMy)
	}
	if !strings.Contains(upMy, "`order` INT") {
		t.Errorf("expected quoted reserved keyword column in mysql: %s", upMy)
	}
	if !strings.Contains(downMy, "DROP TABLE IF EXISTS `custom_users`;") {
		t.Errorf("expected quoted table name in down migration: %s", downMy)
	}
}
