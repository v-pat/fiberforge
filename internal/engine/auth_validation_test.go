package engine

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/schema"
)

func TestAuthValidation_GeneratedTags(t *testing.T) {
	databases := []string{"postgres", "mongodb"}
	for _, db := range databases {
		t.Run(db, func(t *testing.T) {
			cfg := &schema.Config{
				AppName:   "authvaltest",
				Framework: "fiber",
				Database:  db,
				Features:  schema.Features{Auth: true},
				Models: []schema.Model{
					{
						Name:     "user",
						Endpoint: "users",
						Fields: []schema.Field{
							{Name: "email", Type: schema.TypeString, Required: true},
							{Name: "password", Type: schema.TypePassword, Required: true},
						},
					},
				},
			}

			outDir := t.TempDir()
			cfg.OutputDir = outDir

			if _, err := New(cfg).Generate(); err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			ctrlPath := filepath.Join(outDir, "controller", "auth_controller.go")
			content, err := os.ReadFile(ctrlPath)
			if err != nil {
				t.Fatalf("failed to read auth_controller.go: %v", err)
			}

			ctrlStr := string(content)

			// Invariant: credentials struct must have validation tags for email and password
			if !strings.Contains(ctrlStr, `validate:"required,email,max=255"`) {
				t.Fatalf("credentials struct must enforce email validation tags in %s, got:\n%s", db, ctrlStr)
			}
			if !strings.Contains(ctrlStr, `validate:"required,min=8,max=72"`) {
				t.Fatalf("credentials struct must enforce password validation tags (min 8, max 72) in %s, got:\n%s", db, ctrlStr)
			}

			// Invariant: Login and Register must not leak whether an account exists
			if strings.Contains(ctrlStr, "user not found") || strings.Contains(ctrlStr, "incorrect password") {
				t.Fatalf("controller must not leak whether account exists or password was incorrect")
			}
		})
	}
}

func TestAuthValidation_CredentialsValidationLogic(t *testing.T) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	validateCredentials := func(email, password string) error {
		if strings.TrimSpace(email) == "" {
			return &validationError{"email is required"}
		}
		if len(email) > 255 || !emailRegex.MatchString(email) {
			return &validationError{"email is invalid"}
		}
		if len(password) < 8 {
			return &validationError{"password must be at least 8 characters"}
		}
		if len(password) > 72 {
			return &validationError{"password cannot exceed 72 characters"}
		}
		return nil
	}

	testCases := []struct {
		name      string
		email     string
		password  string
		expectErr bool
	}{
		{"Empty email", "", "secret123", true},
		{"Malformed email", "not-an-email", "secret123", true},
		{"Empty password", "user@example.com", "", true},
		{"Too short password (<8)", "user@example.com", "short", true},
		{"Excessively long password (>72)", "user@example.com", strings.Repeat("A", 73), true},
		{"Valid credentials", "user@example.com", "secret123", false},
		{"Valid boundary password 8 chars", "user@example.com", "12345678", false},
		{"Valid boundary password 72 chars", "user@example.com", strings.Repeat("B", 72), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCredentials(tc.email, tc.password)
			if tc.expectErr && err == nil {
				t.Fatalf("expected validation error for case %q, but got nil", tc.name)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected case %q to pass, but got error: %v", tc.name, err)
			}
		})
	}
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}
