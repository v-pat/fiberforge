package schema

import (
	"testing"
)

func TestValidate_TableNameSecurity(t *testing.T) {
	maliciousTableNames := []string{
		"users\" }\n\nfunc init() { panic(\"PWNED\") }\n\nfunc (User) Dummy() string { return \"",
		"users\"; DROP TABLE users; --",
		"users`",
		"users' OR '1'='1",
		"users/*comment*/",
		"users\nnewline",
		"users\rreturn",
		"users;whoami",
		"users-hyphen",
		"123users",
		"users$evil",
	}

	for _, name := range maliciousTableNames {
		cfg := &Config{
			AppName:   "testapp",
			Database:  "postgres",
			Framework: "fiber",
			Models: []Model{
				{
					Name:      "User",
					Endpoint:  "users",
					TableName: name,
					Fields: []Field{
						{Name: "Email", Type: TypeString},
						{Name: "Password", Type: TypePassword},
					},
				},
			},
		}

		err := Validate(cfg)
		if err == nil {
			t.Errorf("expected malicious tableName %q to fail validation, but it passed", name)
		}
	}

	validTableNames := []string{
		"users",
		"app_users",
		"_internal_data",
		"UsersTable",
	}

	for _, name := range validTableNames {
		cfg := &Config{
			AppName:   "testapp",
			Database:  "postgres",
			Framework: "fiber",
			Models: []Model{
				{
					Name:      "User",
					Endpoint:  "users",
					TableName: name,
					Fields: []Field{
						{Name: "Email", Type: TypeString},
						{Name: "Password", Type: TypePassword},
					},
				},
			},
		}

		err := Validate(cfg)
		if err != nil {
			t.Errorf("expected valid tableName %q to pass validation, got error: %v", name, err)
		}
	}
}
