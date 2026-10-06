package engine

import (
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestBuildStructTag_Sanitization(t *testing.T) {
	tests := []struct {
		name       string
		field      schema.Field
		jsonName   string
		disallowed []string
	}{
		{
			name: "backtick breakout",
			field: schema.Field{
				Name:       "Test",
				Type:       schema.TypeString,
				Validation: "email`\n\tfunc init() {}",
			},
			jsonName:   "test`",
			disallowed: []string{"`", "\n", "\r", "\""},
		},
		{
			name: "normal field",
			field: schema.Field{
				Name:       "Email",
				Type:       schema.TypeString,
				Required:   true,
				Validation: "email",
			},
			jsonName: "email",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := buildStructTag(tc.field, tc.jsonName)
			// Tag must start and end with backtick
			if !strings.HasPrefix(tag, "`") || !strings.HasSuffix(tag, "`") {
				t.Fatalf("struct tag is not backtick-wrapped: %s", tag)
			}
			inner := tag[1 : len(tag)-1]
			// Inner must not contain any backticks
			if strings.Contains(inner, "`") {
				t.Fatalf("inner tag contains unescaped backtick: %s", tag)
			}
			// Inner must not contain newlines
			if strings.Contains(inner, "\n") || strings.Contains(inner, "\r") {
				t.Fatalf("inner tag contains newlines: %s", tag)
			}
		})
	}
}
