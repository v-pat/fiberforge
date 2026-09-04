package schema

import (
	"encoding/json"
	"fmt"
	"os"
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
	return nil
}

// Validate checks that the config is semantically consistent.
func Validate(cfg *Config) error {
	if cfg.AppName == "" {
		return fmt.Errorf("appName is required")
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
					hasEmail = true
				case strings.EqualFold(f.Name, "password"):
					hasPassword = true
				}
			}
			if !hasEmail || !hasPassword {
				return fmt.Errorf("the user model needs email and password fields when auth is enabled")
			}
		}
	}

	nameSeen := map[string]bool{}
	for _, m := range cfg.Models {
		if m.Name == "" {
			return fmt.Errorf("every model needs a name")
		}
		if nameSeen[strings.ToLower(m.Name)] {
			return fmt.Errorf("duplicate model name %q", m.Name)
		}
		nameSeen[strings.ToLower(m.Name)] = true
		if m.Endpoint == "" {
			return fmt.Errorf("model %q is missing an endpoint", m.Name)
		}
		for _, f := range m.Fields {
			if f.Name == "" {
				return fmt.Errorf("model %q has a field with no name", m.Name)
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
