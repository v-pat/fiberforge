package blueprint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/v-pat/fiberforge/internal/schema"
)

var (
	idRegex    = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*$`)
	envRegex   = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	identRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// ParseManifest decodes blueprint manifest bytes (YAML or JSON) and validates it.
func ParseManifest(data []byte) (*Blueprint, error) {
	var bp Blueprint
	if err := yaml.Unmarshal(data, &bp); err != nil {
		if jsonErr := json.Unmarshal(data, &bp); jsonErr != nil {
			return nil, fmt.Errorf("invalid blueprint manifest: YAML (%v), JSON (%v)", err, jsonErr)
		}
	}

	applyDefaults(&bp)

	if err := Validate(&bp); err != nil {
		return nil, fmt.Errorf("invalid blueprint %q: %w", bp.ID, err)
	}

	return &bp, nil
}

// ParseBundle loads a complete blueprint directory bundle containing blueprint.yaml.
func ParseBundle(bundleDir string) (*Blueprint, error) {
	manifestPath := filepath.Join(bundleDir, "blueprint.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// Fallback to recipe.yaml for backwards compatibility
		fallbackPath := filepath.Join(bundleDir, "recipe.yaml")
		if _, err := os.Stat(fallbackPath); err == nil {
			manifestPath = fallbackPath
		} else {
			return nil, fmt.Errorf("manifest blueprint.yaml not found in %s", bundleDir)
		}
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest from %s: %w", manifestPath, err)
	}

	bp, err := ParseManifest(data)
	if err != nil {
		return nil, err
	}

	// Verify all mapped source files exist in the bundle
	for i, f := range bp.Files {
		srcPath := filepath.Join(bundleDir, f.Source)
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("blueprint %q references missing file %q: %w", bp.ID, f.Source, err)
		}
		bp.Files[i].Content = string(content)
	}

	return bp, nil
}

func applyDefaults(bp *Blueprint) {
	if bp.Target.Language == "" {
		bp.Target.Language = "go"
	}
	if bp.Target.Framework == "" {
		bp.Target.Framework = "fiber"
	}
	for i := range bp.Models {
		m := &bp.Models[i]
		if m.Endpoint == "" && m.Name != "" {
			m.Endpoint = strings.ToLower(schema.Plural(m.Name))
		}
	}
}

// Validate verifies that the blueprint manifest is syntactically and semantically sound.
func Validate(bp *Blueprint) error {
	if bp.ID == "" {
		return fmt.Errorf("id is required")
	}
	if !idRegex.MatchString(bp.ID) {
		return fmt.Errorf("id %q must be lowercase alphanumeric with hyphens (e.g. 'stripe-billing')", bp.ID)
	}
	if bp.Version == "" {
		return fmt.Errorf("version is required")
	}
	if bp.Title == "" {
		return fmt.Errorf("title is required")
	}
	if bp.Description == "" {
		return fmt.Errorf("description is required")
	}
	if bp.Category == "" {
		bp.Category = "General"
	}

	// Validate Env vars
	for _, env := range bp.Env {
		if env.Key == "" {
			return fmt.Errorf("env var key is required")
		}
		if !envRegex.MatchString(env.Key) {
			return fmt.Errorf("env var %q must be uppercase alphanumeric with underscores", env.Key)
		}
	}

	// Validate File mappings and protect against path traversal
	for _, f := range bp.Files {
		if f.Target == "" {
			return fmt.Errorf("file target path is required")
		}
		cleanTarget := filepath.Clean(f.Target)
		if filepath.IsAbs(cleanTarget) || strings.HasPrefix(cleanTarget, "..") || strings.Contains(cleanTarget, "/../") {
			return fmt.Errorf("file target %q contains illegal path traversal", f.Target)
		}
	}

	// Validate Models
	for _, m := range bp.Models {
		if m.Name == "" {
			return fmt.Errorf("model name is required")
		}
		if !identRegex.MatchString(m.Name) {
			return fmt.Errorf("model name %q must be a valid identifier", m.Name)
		}
		if err := schema.ValidateEndpoint(m.Endpoint); err != nil {
			return fmt.Errorf("model %q: %w", m.Name, err)
		}
		for _, f := range m.Fields {
			if f.Name == "" {
				return fmt.Errorf("model %q field name is required", m.Name)
			}
			if !identRegex.MatchString(f.Name) {
				return fmt.Errorf("model %q field %q must be a valid identifier", m.Name, f.Name)
			}
		}
	}

	// Validate Routes hooks
	for _, r := range bp.Hooks.Routes {
		if r.Path == "" {
			return fmt.Errorf("route hook path is required")
		}
		if r.Handler == "" {
			return fmt.Errorf("route hook handler is required")
		}
		if r.Method == "" {
			return fmt.Errorf("route hook method is required")
		}
		method := strings.ToUpper(r.Method)
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "ALL", "USE":
		default:
			return fmt.Errorf("invalid route hook HTTP method %q", r.Method)
		}
	}

	return nil
}
