package blueprint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
	"github.com/v-pat/fiberforge/internal/verifier"
)

// ApplyResult contains the outcome of applying a blueprint to a project.
type ApplyResult struct {
	BlueprintID    string           `json:"blueprintId"`
	Title          string           `json:"title"`
	AddedModels    []string         `json:"addedModels"`
	GeneratedFiles []string         `json:"generatedFiles"`
	InjectedRoutes []string         `json:"injectedRoutes"`
	AgentPrompt    string           `json:"agentPrompt,omitempty"`
	Verification   *verifier.Result `json:"verification,omitempty"`
}

// Apply executes or previews the application of a blueprint to a target project directory.
func Apply(targetDir string, blueprintID string, dryRun bool, skipVerify bool) (*ApplyResult, error) {
	safeTargetDir, err := schema.ResolveSafePath(targetDir, "")
	if err != nil {
		return nil, fmt.Errorf("invalid target directory: %w", err)
	}
	targetDir = safeTargetDir

	bp, err := Get(blueprintID)
	if err != nil {
		return nil, err
	}

	result := &ApplyResult{
		BlueprintID: bp.ID,
		Title:       bp.Title,
		AgentPrompt: bp.AgentPrompt,
	}

	var allFiles []string
	var addedModels []string
	var injectedRoutes []string

	// 1. Files in blueprint bundle
	for _, f := range bp.Files {
		allFiles = append(allFiles, f.Target)
		if !dryRun {
			destPath := filepath.Join(targetDir, f.Target)
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return nil, fmt.Errorf("creating directory for %s: %w", f.Target, err)
			}
			if err := os.WriteFile(destPath, []byte(f.Content), 0o644); err != nil {
				return nil, fmt.Errorf("writing file %s: %w", f.Target, err)
			}
		}
	}

	// 2. Models injection via engine.AddModelWithOptions
	for _, m := range bp.Models {
		files, err := engine.AddModelWithOptions(targetDir, m, dryRun)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return nil, fmt.Errorf("failed adding model %q: %w", m.Name, err)
		}
		addedModels = append(addedModels, m.Name)
		allFiles = append(allFiles, files...)
	}

	// 3. Hooks injection (AST routes, middleware, imports)
	routesFilePath := filepath.Join(targetDir, "routes", "routes.go")
	mainFilePath := filepath.Join(targetDir, "main.go")

	for _, r := range bp.Hooks.Routes {
		routeDesc := fmt.Sprintf("%s %s -> %s", strings.ToUpper(r.Method), r.Path, r.Handler)
		injectedRoutes = append(injectedRoutes, routeDesc)
		if !dryRun && fileExists(routesFilePath) {
			_ = engine.GraftRouteHook(routesFilePath, engine.RouteHook{
				Group:   r.Group,
				Path:    r.Path,
				Method:  r.Method,
				Handler: r.Handler,
				Auth:    r.Auth,
			})
		}
	}

	if !dryRun {
		if fileExists(routesFilePath) {
			if len(bp.Hooks.Imports) > 0 {
				_ = engine.GraftImports(routesFilePath, bp.Hooks.Imports)
			}
			for _, mw := range bp.Hooks.Middleware {
				_ = engine.GraftMiddleware(routesFilePath, mw)
			}
		}

		if fileExists(mainFilePath) {
			for _, initCall := range bp.Hooks.Init {
				_ = engine.GraftInitCall(mainFilePath, initCall)
			}
		}

		// 4. Update .env.example with required variables
		updateEnvExample(targetDir, bp.Env)

		// 5. Record applied blueprint in fiberforge.yaml
		_ = RecordAppliedBlueprint(targetDir, AppliedBlueprintRecord{
			ID:        bp.ID,
			Version:   bp.Version,
			Files:     allFiles,
			Models:    addedModels,
			AppliedAt: time.Now().UTC().Format(time.RFC3339),
		})

		// 6. Optional Closed-Loop Verification
		if !skipVerify && fileExists(filepath.Join(targetDir, "go.mod")) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			verifyRes, _ := verifier.Verify(ctx, targetDir, verifier.Options{
				RunTidy:   true,
				SkipTests: true,
			})
			result.Verification = verifyRes
		}
	}

	result.AddedModels = addedModels
	result.GeneratedFiles = allFiles
	result.InjectedRoutes = injectedRoutes

	return result, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func updateEnvExample(targetDir string, envs []EnvVar) {
	if len(envs) == 0 {
		return
	}
	envPath := filepath.Join(targetDir, ".env.example")
	var existingContent string
	if data, err := os.ReadFile(envPath); err == nil {
		existingContent = string(data)
	}

	var sb strings.Builder
	for _, env := range envs {
		if !strings.Contains(existingContent, env.Key+"=") {
			val := env.Default
			if val == "" {
				val = env.Example
			}
			sb.WriteString(fmt.Sprintf("\n# %s\n%s=%s\n", env.Description, env.Key, val))
		}
	}

	if sb.Len() > 0 {
		f, err := os.OpenFile(envPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			defer f.Close()
			_, _ = f.WriteString(sb.String())
		}
	}
}
