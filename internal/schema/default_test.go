package schema

import (
	"testing"
)

func TestValidateDefault_PayloadsRejected(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	maliciousPayloads := []struct {
		name      string
		fieldType FieldType
		defVal    string
	}{
		{"Semicolon statement termination", TypeString, "test'; DROP TABLE users; --"},
		{"SQL line comment", TypeString, "test -- comment"},
		{"SQL block comment open", TypeString, "test /* comment"},
		{"SQL block comment close", TypeString, "test */ comment"},
		{"Single quote injection", TypeString, "' OR 1=1 --"},
		{"Double quote injection", TypeString, "\" OR 1=1 --"},
		{"Backtick injection", TypeString, "` OR 1=1 --"},
		{"Newline injection", TypeString, "foo\nDROP TABLE users"},
		{"Carriage return injection", TypeString, "foo\rDROP TABLE users"},
		{"Invalid boolean default", TypeBool, "true; DROP TABLE users"},
		{"Invalid int default", TypeInt, "123; DROP TABLE users"},
		{"Invalid float default", TypeFloat, "1.23; DROP TABLE users"},
		{"Invalid time default", TypeTime, "NOW(); DROP TABLE users"},
	}

	for _, tc := range maliciousPayloads {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				AppName:   "myapp",
				Framework: "fiber",
				Database:  "postgres",
				Models: []Model{
					{
						Name:     "item",
						Endpoint: "items",
						Fields: []Field{
							{
								Name:    "title",
								Type:    tc.fieldType,
								Default: strPtr(tc.defVal),
							},
						},
					},
				},
			}

			err := Validate(cfg)
			if err == nil {
				t.Fatalf("expected error for default payload %q in type %s, got nil", tc.defVal, tc.fieldType)
			}
		})
	}
}

func TestValidateDefault_ValidDefaults(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	validCases := []struct {
		fieldType FieldType
		defVal    string
		values    []string
	}{
		{TypeBool, "true", nil},
		{TypeBool, "false", nil},
		{TypeInt, "123", nil},
		{TypeInt, "-5", nil},
		{TypeFloat, "3.14", nil},
		{TypeTime, "CURRENT_TIMESTAMP", nil},
		{TypeTime, "NOW()", nil},
		{TypeString, "active", nil},
		{TypeEnum, "published", []string{"draft", "published"}},
	}

	for _, tc := range validCases {
		t.Run(string(tc.fieldType)+"_"+tc.defVal, func(t *testing.T) {
			cfg := &Config{
				AppName:   "myapp",
				Framework: "fiber",
				Database:  "postgres",
				Models: []Model{
					{
						Name:     "item",
						Endpoint: "items",
						Fields: []Field{
							{
								Name:    "status",
								Type:    tc.fieldType,
								Default: strPtr(tc.defVal),
								Values:  tc.values,
							},
						},
					},
				},
			}

			if err := Validate(cfg); err != nil {
				t.Fatalf("expected valid default %q to pass, got err: %v", tc.defVal, err)
			}
		})
	}
}
