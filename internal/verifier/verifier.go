package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Options configures project verification.
type Options struct {
	SkipTests bool          `json:"skipTests,omitempty"`
	Race      bool          `json:"race,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	RunTidy   bool          `json:"runTidy,omitempty"`
}

// StepResult stores the outcome of an individual verification phase.
type StepResult struct {
	Name     string   `json:"name"`
	Passed   bool     `json:"passed"`
	Output   string   `json:"output,omitempty"`
	Duration int64    `json:"durationMs"`
	Errors   []string `json:"errors,omitempty"`
}

// Result is the comprehensive outcome of verifying a Go project.
type Result struct {
	Success      bool         `json:"success"`
	Directory    string       `json:"directory"`
	Summary      string       `json:"summary"`
	Steps        []StepResult `json:"steps"`
	TestsPassed  int          `json:"testsPassed"`
	TestsFailed  int          `json:"testsFailed"`
	TotalElapsed int64        `json:"totalElapsedMs"`
}

var (
	testPassRe = regexp.MustCompile(`--- PASS:\s+([^\s]+)`)
	testFailRe = regexp.MustCompile(`--- FAIL:\s+([^\s]+)`)
)

// Verify runs tidy, vet, build, and test on a target Go directory.
func Verify(ctx context.Context, dir string, opts Options) (*Result, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid directory: %w", err)
	}

	goModPath := filepath.Join(absDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no go.mod found in %s; ensure target directory is a Go module", absDir)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()
	res := &Result{
		Success:   true,
		Directory: absDir,
		Steps:     make([]StepResult, 0, 4),
	}

	// 1. go mod tidy
	if opts.RunTidy {
		step, passed := runCommand(ctx, absDir, "go", "mod", "tidy")
		step.Name = "tidy"
		res.Steps = append(res.Steps, step)
		if !passed {
			res.Success = false
			res.Summary = fmt.Sprintf("go mod tidy failed: %s", formatErrors(step.Errors))
			res.TotalElapsed = time.Since(startTime).Milliseconds()
			return res, nil
		}
	}

	// 2. go vet ./...
	vetStep, passed := runCommand(ctx, absDir, "go", "vet", "./...")
	vetStep.Name = "vet"
	res.Steps = append(res.Steps, vetStep)
	if !passed {
		res.Success = false
		res.Summary = fmt.Sprintf("go vet failed with %d issue(s)", len(vetStep.Errors))
		res.TotalElapsed = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// 3. go build ./...
	buildStep, passed := runCommand(ctx, absDir, "go", "build", "./...")
	buildStep.Name = "build"
	res.Steps = append(res.Steps, buildStep)
	if !passed {
		res.Success = false
		res.Summary = fmt.Sprintf("go build failed with %d error(s)", len(buildStep.Errors))
		res.TotalElapsed = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// 4. go test ./...
	if !opts.SkipTests {
		testArgs := []string{"test"}
		if opts.Race {
			testArgs = append(testArgs, "-race")
		}
		testArgs = append(testArgs, "-v", "./...")

		testStep, passed := runCommand(ctx, absDir, "go", testArgs...)
		testStep.Name = "test"

		// Parse test output for pass/fail counts
		passMatches := testPassRe.FindAllStringSubmatch(testStep.Output, -1)
		failMatches := testFailRe.FindAllStringSubmatch(testStep.Output, -1)
		res.TestsPassed = len(passMatches)
		res.TestsFailed = len(failMatches)

		for _, m := range failMatches {
			if len(m) > 1 {
				testStep.Errors = append(testStep.Errors, fmt.Sprintf("Test failed: %s", m[1]))
			}
		}

		res.Steps = append(res.Steps, testStep)
		if !passed {
			res.Success = false
			res.Summary = fmt.Sprintf("go test failed: %d failed, %d passed", res.TestsFailed, res.TestsPassed)
			res.TotalElapsed = time.Since(startTime).Milliseconds()
			return res, nil
		}
	}

	res.TotalElapsed = time.Since(startTime).Milliseconds()
	if opts.SkipTests {
		res.Summary = "Verification successful: build and vet passed cleanly (tests skipped)."
	} else {
		res.Summary = fmt.Sprintf("Verification successful: build, vet, and all tests passed cleanly (%d test(s) passed).", res.TestsPassed)
	}

	return res, nil
}

func runCommand(ctx context.Context, dir, name string, args ...string) (StepResult, bool) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	combined := strings.TrimSpace(outBuf.String() + "\n" + errBuf.String())
	lines := splitCleanLines(combined)

	step := StepResult{
		Duration: elapsed,
		Output:   combined,
	}

	if err != nil {
		step.Passed = false
		for _, l := range lines {
			if isDiagnosticLine(l) {
				step.Errors = append(step.Errors, l)
			}
		}
		if len(step.Errors) == 0 && len(lines) > 0 {
			step.Errors = lines
		}
		return step, false
	}

	step.Passed = true
	return step, true
}

func splitCleanLines(s string) []string {
	raw := strings.Split(s, "\n")
	res := make([]string, 0, len(raw))
	for _, l := range raw {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func isDiagnosticLine(l string) bool {
	// Detect typical Go compiler / vet diagnostics: path/file.go:line:col: message
	if strings.Contains(l, ".go:") {
		return true
	}
	if strings.HasPrefix(l, "# ") || strings.HasPrefix(l, "FAIL") {
		return true
	}
	return false
}

func formatErrors(errs []string) string {
	if len(errs) == 0 {
		return "unknown error"
	}
	if len(errs) > 3 {
		return strings.Join(errs[:3], "; ") + fmt.Sprintf(" (+%d more)", len(errs)-3)
	}
	return strings.Join(errs, "; ")
}

// FormatMarkdown creates an agent-friendly markdown report.
func (r *Result) FormatMarkdown() string {
	var sb strings.Builder
	if r.Success {
		sb.WriteString("### ✅ Project Verification Passed\n\n")
	} else {
		sb.WriteString("### ❌ Project Verification Failed\n\n")
	}

	sb.WriteString(fmt.Sprintf("**Target:** `%s`  \n", r.Directory))
	sb.WriteString(fmt.Sprintf("**Summary:** %s  \n", r.Summary))
	sb.WriteString(fmt.Sprintf("**Total Duration:** %dms\n\n", r.TotalElapsed))

	sb.WriteString("| Phase | Status | Duration | Details |\n")
	sb.WriteString("|---|---|---|---|\n")

	for _, step := range r.Steps {
		status := "✅ PASS"
		if !step.Passed {
			status = "❌ FAIL"
		}
		detail := "-"
		if len(step.Errors) > 0 {
			detail = fmt.Sprintf("%d error(s)", len(step.Errors))
		} else if step.Name == "test" {
			detail = fmt.Sprintf("%d passed", r.TestsPassed)
		}
		sb.WriteString(fmt.Sprintf("| `go %s` | %s | %dms | %s |\n", step.Name, status, step.Duration, detail))
	}

	for _, step := range r.Steps {
		if !step.Passed && len(step.Errors) > 0 {
			sb.WriteString(fmt.Sprintf("\n#### Errors in `go %s`:\n```\n", step.Name))
			for _, e := range step.Errors {
				sb.WriteString(e + "\n")
			}
			sb.WriteString("```\n")
		}
	}

	return sb.String()
}
