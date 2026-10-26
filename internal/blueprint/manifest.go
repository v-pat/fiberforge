package blueprint

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/v-pat/fiberforge/internal/schema"
)

// AppliedBlueprintRecord tracks an applied blueprint inside fiberforge.yaml.
type AppliedBlueprintRecord struct {
	ID        string   `yaml:"id" json:"id"`
	Version   string   `yaml:"version" json:"version"`
	AppliedAt string   `yaml:"appliedAt" json:"appliedAt"`
	Files     []string `yaml:"files,omitempty" json:"files,omitempty"`
	Models    []string `yaml:"models,omitempty" json:"models,omitempty"`
}

// ProjectConfig represents the minimal fiberforge.yaml structure for tracking blueprints.
type ProjectConfig struct {
	AppName    string                   `yaml:"appName,omitempty"`
	Database   string                   `yaml:"database,omitempty"`
	Blueprints []AppliedBlueprintRecord `yaml:"blueprints,omitempty"`
}

func getManifestPath(projectDir string) (string, error) {
	safeDir, err := schema.ResolveSafePath(projectDir, "")
	if err != nil {
		return "", err
	}
	path := filepath.Join(safeDir, "fiberforge.yaml")
	if _, err := os.Stat(path); err != nil {
		// Fallback to fiberforge.yml
		ymlPath := filepath.Join(safeDir, "fiberforge.yml")
		if _, err := os.Stat(ymlPath); err == nil {
			return ymlPath, nil
		}
	}
	return path, nil
}

// RecordAppliedBlueprint writes or updates a blueprint record in fiberforge.yaml.
func RecordAppliedBlueprint(projectDir string, rec AppliedBlueprintRecord) error {
	manifestPath, err := getManifestPath(projectDir)
	if err != nil {
		return err
	}

	if rec.AppliedAt == "" {
		rec.AppliedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var root yaml.Node
	data, err := os.ReadFile(manifestPath)
	if err == nil && len(data) > 0 {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("parsing %s: %w", manifestPath, err)
		}
	} else {
		// Create empty mapping node
		root.Kind = yaml.DocumentNode
		mapping := &yaml.Node{Kind: yaml.MappingNode}
		root.Content = []*yaml.Node{mapping}
	}

	// Read existing blueprints to update or append
	var cfg struct {
		Blueprints []AppliedBlueprintRecord `yaml:"blueprints"`
	}
	_ = yaml.Unmarshal(data, &cfg)

	found := false
	for i, existing := range cfg.Blueprints {
		if existing.ID == rec.ID {
			cfg.Blueprints[i] = rec
			found = true
			break
		}
	}
	if !found {
		cfg.Blueprints = append(cfg.Blueprints, rec)
	}

	// Update raw yaml data with the updated blueprints key
	var genericMap map[string]any
	if len(data) > 0 {
		_ = yaml.Unmarshal(data, &genericMap)
	}
	if genericMap == nil {
		genericMap = make(map[string]any)
	}
	genericMap["blueprints"] = cfg.Blueprints

	out, err := yaml.Marshal(genericMap)
	if err != nil {
		return fmt.Errorf("marshaling fiberforge.yaml: %w", err)
	}

	return os.WriteFile(manifestPath, out, 0o644)
}

// IsBlueprintApplied checks if a blueprint is recorded as installed in the project.
func IsBlueprintApplied(projectDir string, blueprintID string) (bool, *AppliedBlueprintRecord, error) {
	records, err := ListAppliedBlueprints(projectDir)
	if err != nil {
		return false, nil, err
	}
	for _, rec := range records {
		if rec.ID == blueprintID {
			return true, &rec, nil
		}
	}
	return false, nil, nil
}

// ListAppliedBlueprints returns all blueprints recorded in fiberforge.yaml.
func ListAppliedBlueprints(projectDir string) ([]AppliedBlueprintRecord, error) {
	manifestPath, err := getManifestPath(projectDir)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
	}

	var cfg struct {
		Blueprints []AppliedBlueprintRecord `yaml:"blueprints"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", manifestPath, err)
	}

	return cfg.Blueprints, nil
}
