package schema

import (
	"testing"
)

func TestValidate_StructTagBreakoutRejected(t *testing.T) {
	maliciousFields := []Field{
		{
			Name:       "Email",
			Type:       TypeString,
			Validation: "email`\n\tfunc init() {}\n\ttype X struct {`",
		},
		{
			Name:       "Email",
			Type:       TypeString,
			Validation: "email`",
		},
		{
			Name:       "Email",
			Type:       TypeString,
			Validation: "email\nrequired",
		},
		{
			Name:       "Email",
			Type:       TypeString,
			Validation: "email\"quote",
		},
		{
			Name:    "Email",
			Type:    TypeString,
			JSONTag: "email`",
		},
		{
			Name:    "Email",
			Type:    TypeString,
			JSONTag: "email\nbreak",
		},
		{
			Name:    "Email",
			Type:    TypeString,
			JSONTag: "email\"tag",
		},
	}

	for _, f := range maliciousFields {
		cfg := &Config{
			AppName:   "testapp",
			Database:  "postgres",
			Framework: "fiber",
			Models: []Model{
				{
					Name:     "User",
					Endpoint: "users",
					Fields: []Field{
						f,
						{Name: "Password", Type: TypePassword},
					},
				},
			},
		}

		err := Validate(cfg)
		if err == nil {
			t.Errorf("expected malicious field (Validation: %q, JSONTag: %q) to fail validation, but it passed", f.Validation, f.JSONTag)
		}
	}

	validFields := []Field{
		{Name: "Email", Type: TypeString, Validation: "email,required", JSONTag: "email"},
		{Name: "Age", Type: TypeInt, Validation: "gte=0,lte=130", JSONTag: "user_age"},
		{Name: "Code", Type: TypeString, Validation: "min=3,max=10", JSONTag: "item-code"},
	}

	for _, f := range validFields {
		cfg := &Config{
			AppName:   "testapp",
			Database:  "postgres",
			Framework: "fiber",
			Models: []Model{
				{
					Name:     "User",
					Endpoint: "users",
					Fields: []Field{
						f,
						{Name: "Password", Type: TypePassword},
					},
				},
			},
		}

		err := Validate(cfg)
		if err != nil {
			t.Errorf("expected valid field tags to pass validation, got error: %v", err)
		}
	}
}
