// Package verify implements the single-command production verification gate.
//
// Per AGENTS.md the language split is strict: everything WE write outside
// Blender's bundled Python is Go 1.27 (zero third-party dependencies). Hence
// this verifier is Go; `scripts/verify.py` is only a thin compatibility shim
// that shells out to `assetctl verify`.
package verify

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/runner"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/skills"
)

// StepResult is one check in the verification report.
type StepResult struct {
	Name     string
	Err      error
	Duration time.Duration
}

// Report is the full verification result.
type Report struct {
	Root    string
	Results []StepResult
}

func (r *Report) OK() bool {
	for _, s := range r.Results {
		if s.Err != nil {
			return false
		}
	}
	return true
}

type step struct {
	name string
	fn   func() error
}

// Run executes every verification step in order and fails fast. It is safe on
// Windows / Linux / macOS: no shell, no Unix-only tools, paths via filepath.
func Run(root string, stdout, stderr io.Writer) (*Report, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rep := &Report{Root: abs}

	py := findPython(abs)
	pyFiles := pythonFiles(abs)

	steps := []step{
		{"repo layout (11 bpy modules + harness)", func() error { return checkLayout(abs) }},
		{"skills registry (28 docs + Go catalog + bpy)", func() error { return skills.ValidateCatalog(abs) }},
		{"go.mod: zero third-party dependencies", func() error { return checkGoMod(abs) }},
		{"gofmt -l (no unformatted Go files)", func() error { return checkGofmt(abs) }},
		{"go build ./...", func() error { return command(abs, "go", "build", "./...") }},
		{"go vet ./...", func() error { return command(abs, "go", "vet", "./...") }},
		{"go test ./... (race + coverage on Unix)", func() error { return goTest(abs) }},
		{"coverage >= " + minCoverageLabel() + "%", func() error { return checkCoverage(abs) }},
		{"python py_compile (scripts/blender + testing)", func() error {
			return pyCompile(abs, py, pyFiles)
		}},
		{"python unittest discover -s testing (stage tests)", func() error {
			if py == "" {
				return fmt.Errorf("no Python interpreter; set ASSETCTL_PYTHON")
			}
			return command(abs, py, "-m", "unittest", "discover", "-s", "testing")
		}},
		{"assetctl doctor", func() error {
			return command(abs, "go", "run", "./cmd/assetctl", "doctor", "--root", abs)
		}},
		{"assetctl CLI smoke (version/strategies/stages)", func() error {
			for _, args := range [][]string{
				{"version"}, {"strategies"}, {"stages"},
			} {
				full := append([]string{"run", "./cmd/assetctl"}, args...)
				if err := command(abs, "go", full...); err != nil {
					return err
				}
			}
			return nil
		}},
	}

	fmt.Fprintf(stdout, "[verify] root=%s goos=%s python=%s\n", abs, runtime.GOOS, py)
	for _, s := range steps {
		start := time.Now()
		err := s.fn()
		res := StepResult{Name: s.name, Err: err, Duration: time.Since(start)}
		rep.Results = append(rep.Results, res)
		if err != nil {
			fmt.Fprintf(stdout, "[verify] FAIL %-46s %v (%s)\n", s.name, err, res.Duration.Round(time.Millisecond))
			return rep, fmt.Errorf("verify: %s: %w", s.name, err)
		}
		fmt.Fprintf(stdout, "[verify] OK   %-46s (%s)\n", s.name, res.Duration.Round(time.Millisecond))
	}
	fmt.Fprintf(stdout, "[verify] PASS: %d steps, all green (no Blender 5.2 required)\n", len(rep.Results))
	return rep, nil
}

func command(dir, exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	// Blender-less CI must not inherit a stale override that points at a
	// different checkout.
	cmd.Env = append(os.Environ(), "ASSETCTL_SKIP_BLENDER_VERSION=1")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w\n%s", exe, strings.Join(args, " "), err, buf.String())
	}
	return nil
}

func goTest(dir string) error {
	args := []string{"test", "./...", "-count=1", "-coverprofile=coverage.out"}
	if runtime.GOOS != "windows" && os.Getenv("ASSETCTL_VERIFY_NO_RACE") != "1" {
		args = append(args, "-race")
	}
	return command(dir, "go", args...)
}

// defaultMinCoverage is the statement-coverage floor for the production gate.
// Override with ASSETCTL_VERIFY_MIN_COVERAGE (e.g. 0 to disable while
// refactoring); CI keeps the default so coverage cannot silently regress.
const defaultMinCoverage = 70.0

func minCoverage() float64 {
	if v := strings.TrimSpace(os.Getenv("ASSETCTL_VERIFY_MIN_COVERAGE")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return defaultMinCoverage
}

func minCoverageLabel() string {
	return strconv.FormatFloat(minCoverage(), 'f', -1, 64)
}

// checkCoverage parses coverage.out (written by goTest) and enforces the floor.
func checkCoverage(root string) error {
	return checkCoverageFile(root, filepath.Join(root, "coverage.out"))
}

// checkCoverageFile is the testable core. The profile is parsed in pure Go
// instead of shelling out to `go tool cover`: the tool re-execs `go list`
// internally, which fails in restricted sandboxes and adds no information we
// do not already have in the profile.
func checkCoverageFile(root, profile string) error {
	data, err := os.ReadFile(profile)
	if err != nil {
		return fmt.Errorf("coverage %s missing (run go test -coverprofile first): %w", profile, err)
	}
	covered, total, err := parseCoverage(data)
	if err != nil {
		return fmt.Errorf("%s is not a usable Go coverage profile: %w", profile, err)
	}
	if total == 0 {
		return fmt.Errorf("coverage profile %s has no statements", profile)
	}
	got := float64(covered) / float64(total) * 100
	if got < minCoverage() {
		return fmt.Errorf("statement coverage %.1f%% (%d/%d) is below the %.1f%% floor "+
			"(set ASSETCTL_VERIFY_MIN_COVERAGE to override while refactoring)",
			got, covered, total, minCoverage())
	}
	return nil
}

// parseCoverage totals the statement coverage of a Go cover profile.
// Mirrors `go tool cover -func` totals: a block contributes its NumStmt to the
// denominator always, and to the numerator when Count > 0. Works for the
// set / count / atomic modes because only zero-vs-nonzero matters.
func parseCoverage(data []byte) (covered, total int, err error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "mode:") {
		return 0, 0, errors.New(`missing "mode:" header`)
	}
	for i, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// <file>:<startLine>.<startCol>,<endLine>.<endCol> <numStmt> <count>
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return 0, 0, fmt.Errorf("line %d: want 3 fields, got %d (%q)", i+2, len(fields), line)
		}
		nStmt, e1 := strconv.Atoi(fields[1])
		count, e2 := strconv.Atoi(fields[2])
		if e1 != nil || e2 != nil || nStmt < 0 || count < 0 {
			return 0, 0, fmt.Errorf("line %d: malformed counts (%q)", i+2, line)
		}
		total += nStmt
		if count > 0 {
			covered += nStmt
		}
	}
	return covered, total, nil
}

func findPython(root string) string {
	if l, err := runner.NewScriptLauncher(root); err == nil {
		return l.Python
	}
	return ""
}

func pythonFiles(root string) []string {
	var out []string
	for _, d := range []string{
		filepath.Join(root, "scripts", "blender"),
		filepath.Join(root, "testing"),
	} {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
				out = append(out, filepath.Join(d, e.Name()))
			}
		}
	}
	sort.Strings(out)
	return out
}

func pyCompile(root, py string, files []string) error {
	if py == "" {
		return fmt.Errorf("no Python interpreter; set ASSETCTL_PYTHON")
	}
	// Compile in batches to stay well under OS argument-length limits.
	for i := 0; i < len(files); i += 20 {
		end := i + 20
		if end > len(files) {
			end = len(files)
		}
		args := append([]string{"-m", "py_compile"}, files[i:end]...)
		if err := command(root, py, args...); err != nil {
			return err
		}
	}
	return nil
}

// checkGofmt fails when any tracked Go file is not gofmt-clean. Formatting is
// part of the production gate: an unformatted diff means the author skipped a
// local step, which historically correlates with un-run tests.
func checkGofmt(root string) error {
	cmd := exec.Command("gofmt", "-l", ".")
	cmd.Dir = root
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofmt -l: %w\n%s", err, buf.String())
	}
	if out := strings.TrimSpace(buf.String()); out != "" {
		return fmt.Errorf("unformatted Go files (run `gofmt -w`):\n%s", out)
	}
	return nil
}

func checkLayout(root string) error {
	want := []string{
		"common.py", "stage_01_modeling.py", "stage_02_topology_uv.py",
		"stage_03_bake.py", "stage_04_pbr.py", "stage_05_unity_export.py",
		"stage_06_render.py", "rig_check.py", "humanoid_rig.py",
		"tripo_to_unity.py", "skill_run.py",
	}
	for _, name := range want {
		p := filepath.Join(root, "scripts", "blender", name)
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			return fmt.Errorf("missing bpy script %s", p)
		}
	}
	for _, name := range []string{"fake_blender.py", "mock_bpy.py", "harness.py"} {
		if _, err := os.Stat(filepath.Join(root, "testing", name)); err != nil {
			return fmt.Errorf("missing test harness file testing/%s", name)
		}
	}
	if len(config.Stages) != 6 {
		return fmt.Errorf("config.Stages must have 6 stages")
	}
	return nil
}

// checkGoMod enforces the "Go 1.27 standard library only" hard constraint.
func checkGoMod(root string) error {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "require ") && !strings.HasPrefix(trimmed, "require (") {
			return fmt.Errorf("third-party dependency forbidden: %s", trimmed)
		}
		if strings.HasPrefix(trimmed, "go ") && !strings.HasPrefix(trimmed, "go "+config.DefaultGo) {
			return fmt.Errorf("go directive must be %s, got %q", config.DefaultGo, trimmed)
		}
	}
	if strings.Contains(string(data), "require (") {
		return fmt.Errorf("third-party require block forbidden in go.mod")
	}
	return nil
}
