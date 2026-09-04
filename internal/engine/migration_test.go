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
	if !strings.Contains(upPg, "CREATE TABLE IF NOT EXISTS posts") {
		t.Errorf("expected table creation, got:\n%s", upPg)
	}
	if !strings.Contains(upPg, "userId BIGINT") {
		t.Errorf("expected FK column userId, got:\n%s", upPg)
	}
	if !strings.Contains(upPg, "CREATE TABLE IF NOT EXISTS posts_tags") {
		t.Errorf("expected join table posts_tags, got:\n%s", upPg)
	}
	if !strings.Contains(downPg, "DROP TABLE IF EXISTS posts;") {
		t.Errorf("expected drop table, got:\n%s", downPg)
	}

	upMy, _ := buildMigration(m, "mysql")
	if !strings.Contains(upMy, "BIGINT AUTO_INCREMENT PRIMARY KEY") {
		t.Errorf("expected MySQL primary key syntax, got:\n%s", upMy)
	}
}
