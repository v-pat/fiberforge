package blueprint

import "github.com/v-pat/fiberforge/internal/schema"

// TargetSpec defines the language and framework compatibility for a blueprint.
type TargetSpec struct {
	Language            string `yaml:"language" json:"language"`
	Framework           string `yaml:"framework" json:"framework"`
	MinRuntimeVersion   string `yaml:"minRuntimeVersion,omitempty" json:"minRuntimeVersion,omitempty"`
	MinFrameworkVersion string `yaml:"minFrameworkVersion,omitempty" json:"minFrameworkVersion,omitempty"`
}

// Dependency represents an external package dependency required by the blueprint.
type Dependency struct {
	Package string `yaml:"package" json:"package"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// EnvVar specifies an environment variable configuration entry.
type EnvVar struct {
	Key         string `yaml:"key" json:"key"`
	Required    bool   `yaml:"required" json:"required"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Description string `yaml:"description" json:"description"`
	Example     string `yaml:"example,omitempty" json:"example,omitempty"`
}

// FileMapping maps a source file in the blueprint bundle to a destination path in the target project.
type FileMapping struct {
	Source  string `yaml:"source" json:"source"`
	Target  string `yaml:"target" json:"target"`
	Content string `yaml:"-" json:"-"` // Used when loading from embedded or in-memory bundles
}

// RouteHook specifies an HTTP route registration to inject.
type RouteHook struct {
	Group   string `yaml:"group,omitempty" json:"group,omitempty"`
	Path    string `yaml:"path" json:"path"`
	Method  string `yaml:"method" json:"method"`
	Handler string `yaml:"handler" json:"handler"`
	Auth    bool   `yaml:"auth,omitempty" json:"auth,omitempty"`
}

// Hooks specifies dynamic code grafting points for routes, middleware, imports, and initialization.
type Hooks struct {
	Imports    []string    `yaml:"imports,omitempty" json:"imports,omitempty"`
	Init       []string    `yaml:"init,omitempty" json:"init,omitempty"`
	Middleware []string    `yaml:"middleware,omitempty" json:"middleware,omitempty"`
	Routes     []RouteHook `yaml:"routes,omitempty" json:"routes,omitempty"`
}

// Blueprint represents the complete definition of an architectural feature bundle.
type Blueprint struct {
	ID           string         `yaml:"id" json:"id"`
	Version      string         `yaml:"version" json:"version"`
	Title        string         `yaml:"title" json:"title"`
	Description  string         `yaml:"description" json:"description"`
	Category     string         `yaml:"category" json:"category"`
	Tags         []string       `yaml:"tags,omitempty" json:"tags,omitempty"`
	Author       string         `yaml:"author,omitempty" json:"author,omitempty"`
	License      string         `yaml:"license,omitempty" json:"license,omitempty"`
	Target       TargetSpec     `yaml:"target" json:"target"`
	Dependencies []Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Env          []EnvVar       `yaml:"env,omitempty" json:"env,omitempty"`
	Models       []schema.Model `yaml:"models,omitempty" json:"models,omitempty"`
	Files        []FileMapping  `yaml:"files,omitempty" json:"files,omitempty"`
	Hooks        Hooks          `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	AgentPrompt  string         `yaml:"agentPrompt,omitempty" json:"agentPrompt,omitempty"`
}

// BlueprintSummary provides a lightweight summary for catalog listing and search results.
type BlueprintSummary struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	ModelCount  int      `json:"modelCount"`
	RequiredEnv []string `json:"requiredEnv"`
}

// Summary returns a BlueprintSummary from a full Blueprint.
func (b *Blueprint) Summary() BlueprintSummary {
	var reqEnv []string
	for _, env := range b.Env {
		if env.Required {
			reqEnv = append(reqEnv, env.Key)
		}
	}
	return BlueprintSummary{
		ID:          b.ID,
		Version:     b.Version,
		Title:       b.Title,
		Description: b.Description,
		Category:    b.Category,
		Tags:        b.Tags,
		ModelCount:  len(b.Models),
		RequiredEnv: reqEnv,
	}
}
