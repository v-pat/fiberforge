package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads a config file from disk. Both YAML and JSON are supported and
// detected by file extension (JSON is also valid YAML, so either path works).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes config bytes that may be YAML or JSON.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Fall back to strict JSON decoding if YAML fails.
		var jsonErr error
		if jsonErr = json.Unmarshal(data, &cfg); jsonErr != nil {
			return nil, fmt.Errorf("invalid YAML (%v) and invalid JSON (%v)", err, jsonErr)
		}
	}
	if err := ApplyDefaults(&cfg); err != nil {
		return nil, err
	}
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ApplyDefaults fills in sensible defaults for fields the user omitted.
func ApplyDefaults(cfg *Config) error {
	if cfg.Framework == "" {
		cfg.Framework = "fiber"
	}
	if cfg.Database == "" {
		return fmt.Errorf("database is required (postgres, mysql, or mongodb)")
	}
	cfg.Database = strings.ToLower(cfg.Database)
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.Owner {
			m.AuthProtected = true
			hasUserRel := false
			for _, r := range m.Relationships {
				if r.Type == BelongsTo && strings.EqualFold(r.Model, "user") {
					hasUserRel = true
					break
				}
			}
			hasUserIDField := false
			for _, f := range m.Fields {
				if strings.EqualFold(f.Name, "userid") || strings.EqualFold(f.Name, "user_id") {
					hasUserIDField = true
					break
				}
			}
			if !hasUserRel && !hasUserIDField {
				m.Relationships = append(m.Relationships, Relationship{
					Type:  BelongsTo,
					Model: "user",
				})
			}
		}
	}
	return nil
}

// Validate checks that the config is semantically consistent.
func Validate(cfg *Config) error {
	if cfg.AppName == "" {
		return fmt.Errorf("appName is required")
	}
	if err := ValidateAppName(cfg.AppName); err != nil {
		return err
	}
	if cfg.Framework != "fiber" {
		return fmt.Errorf("unsupported framework %q (only fiber is supported)", cfg.Framework)
	}
	switch cfg.Database {
	case "postgres", "mysql", "mongodb":
	default:
		return fmt.Errorf("unsupported database %q (postgres, mysql, mongodb)", cfg.Database)
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("port %d out of range", cfg.Port)
	}

	// Auth needs a User model with email + password fields. When the user did
	// not define one, ensureUserModel adds it later; when they did, require the
	// fields the auth module expects so the generated code compiles.
	if cfg.Features.Auth {
		for _, m := range cfg.Models {
			if !strings.EqualFold(m.Name, "user") {
				continue
			}
			hasEmail, hasPassword := false, false
			for _, f := range m.Fields {
				switch {
				case strings.EqualFold(f.Name, "email"):
					if f.Type == "string" {
						hasEmail = true
					}
				case strings.EqualFold(f.Name, "password"):
					if f.Type == "string" || f.Type == "password" {
						hasPassword = true
					}
				}
			}
			if !hasEmail || !hasPassword {
				return fmt.Errorf("the user model needs email (string) and password (string/password) fields when auth is enabled")
			}
		}
	}

	nameSeen := map[string]bool{}
	identRe := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
	for _, m := range cfg.Models {
		if m.Name == "" {
			return fmt.Errorf("every model needs a name")
		}
		if !identRe.MatchString(m.Name) {
			return fmt.Errorf("model name %q is not a valid identifier", m.Name)
		}
		if nameSeen[strings.ToLower(m.Name)] {
			return fmt.Errorf("duplicate model name %q", m.Name)
		}
		nameSeen[strings.ToLower(m.Name)] = true
		if m.TableName != "" && !identRe.MatchString(m.TableName) {
			return fmt.Errorf("model %q tableName %q is not a valid identifier", m.Name, m.TableName)
		}
		if m.Owner && !cfg.Features.Auth {
			return fmt.Errorf("model %q has owner: true but features.auth is not enabled", m.Name)
		}
		if err := ValidateEndpoint(m.Endpoint); err != nil {
			return fmt.Errorf("model %q: %w", m.Name, err)
		}
		for _, f := range m.Fields {
			if f.Name == "" {
				return fmt.Errorf("model %q has a field with no name", m.Name)
			}
			if !identRe.MatchString(f.Name) {
				return fmt.Errorf("model %q field %q is not a valid identifier", m.Name, f.Name)
			}
			if err := validateField(f); err != nil {
				return fmt.Errorf("model %q field %q: %w", m.Name, f.Name, err)
			}
		}
		for _, r := range m.Relationships {
			switch r.Type {
			case BelongsTo, HasMany, ManyToMany:
			default:
				return fmt.Errorf("model %q has invalid relationship type %q", m.Name, r.Type)
			}
			if !nameSeen[strings.ToLower(r.Model)] && !isReferenced(r.Model, cfg) {
				return fmt.Errorf("model %q relationship references unknown model %q", m.Name, r.Model)
			}
		}
	}
	return nil
}

func isReferenced(name string, cfg *Config) bool {
	if cfg.Features.Auth && strings.EqualFold(name, "user") {
		return true
	}
	for _, m := range cfg.Models {
		if strings.EqualFold(m.Name, name) {
			return true
		}
	}
	return false
}

func validateField(f Field) error {
	if err := validateFieldTags(f); err != nil {
		return err
	}
	if err := validateDefault(f); err != nil {
		return err
	}
	switch f.Type {
	case TypeString, TypeText, TypeInt, TypeInt64, TypeFloat, TypeBool,
		TypeTime, TypeUUID, TypeJSON, TypePassword:
		return nil
	case TypeEnum:
		if len(f.Values) == 0 {
			return fmt.Errorf("enum field needs values")
		}
		return nil
	default:
		return fmt.Errorf("unsupported field type %q", f.Type)
	}
}

var jsonTagRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
var validationRe = regexp.MustCompile(`^[a-zA-Z0-9_=,.-]+$`)

func validateFieldTags(f Field) error {
	if f.JSONTag != "" {
		if !jsonTagRe.MatchString(f.JSONTag) || strings.Contains(f.JSONTag, "..") {
			return fmt.Errorf("invalid jsonTag %q: must contain only alphanumeric characters, underscores, hyphens, or dots", f.JSONTag)
		}
	}
	if f.Validation != "" {
		if !validationRe.MatchString(f.Validation) || strings.ContainsAny(f.Validation, "`\r\n\"") {
			return fmt.Errorf("invalid validation rules %q: contains illegal characters", f.Validation)
		}
	}
	return nil
}

var endpointRe = regexp.MustCompile(`^[a-zA-Z0-9/_-]+$`)

// ValidateEndpoint verifies that an endpoint path contains only valid URL segment characters without path traversal.
func ValidateEndpoint(ep string) error {
	if ep == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}
	if !endpointRe.MatchString(ep) || strings.Contains(ep, "..") {
		return fmt.Errorf("endpoint %q contains invalid characters (must match %s without '..')", ep, endpointRe.String())
	}
	return nil
}

var (
	intDefaultRe    = regexp.MustCompile(`^[+-]?[0-9]+$`)
	floatDefaultRe  = regexp.MustCompile(`^[+-]?[0-9]+(\.[0-9]+)?$`)
	stringDefaultRe = regexp.MustCompile(`^[a-zA-Z0-9_ -]+$`)
)

func validateDefault(f Field) error {
	if f.Default == nil {
		return nil
	}
	val := strings.TrimSpace(*f.Default)
	if val == "" {
		return nil
	}
	// Reject SQL injection / comment / statement termination characters
	if strings.ContainsAny(val, ";'\"`\r\n\x00") || strings.Contains(val, "--") || strings.Contains(val, "/*") || strings.Contains(val, "*/") {
		return fmt.Errorf("invalid default value %q: contains illegal characters or SQL injection tokens", val)
	}

	switch f.Type {
	case TypeBool:
		switch strings.ToLower(val) {
		case "true", "false", "0", "1":
			return nil
		default:
			return fmt.Errorf("invalid boolean default %q (must be true, false, 0, or 1)", val)
		}
	case TypeInt, TypeInt64:
		if !intDefaultRe.MatchString(val) {
			return fmt.Errorf("invalid integer default %q", val)
		}
	case TypeFloat:
		if !floatDefaultRe.MatchString(val) {
			return fmt.Errorf("invalid float default %q", val)
		}
	case TypeTime:
		switch strings.ToUpper(val) {
		case "CURRENT_TIMESTAMP", "NOW()":
			return nil
		default:
			return fmt.Errorf("invalid time default %q (must be CURRENT_TIMESTAMP or NOW())", val)
		}
	case TypeEnum:
		if len(f.Values) > 0 {
			found := false
			for _, v := range f.Values {
				if v == val {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("default %q is not in enum values %v", val, f.Values)
			}
		}
		if !stringDefaultRe.MatchString(val) {
			return fmt.Errorf("invalid enum default %q: contains invalid characters", val)
		}
	case TypeString, TypeText, TypeUUID:
		if !stringDefaultRe.MatchString(val) {
			return fmt.Errorf("invalid string default %q: contains invalid characters", val)
		}
	case TypeJSON:
		if val != "{}" && val != "[]" {
			return fmt.Errorf("invalid JSON default %q (only '{}' or '[]' allowed)", val)
		}
	}
	return nil
}
