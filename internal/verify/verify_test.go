// Unit tests for the verification gate itself (SPEC.md §3A.3).
//
// These cover the pure/cheap steps directly. The full Run() pipeline shells
// out to go/py_compile/unittest, so it is exercised end-to-end by CI and by
// `assetctl verify`, not from a unit test.
package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReportOK(t *testing.T) {
	ok := &Report{Results: []StepResult{{Name: "a"}, {Name: "b"}}}
	if !ok.OK() {
		t.Fatal("empty error list must be OK")
	}
	bad := &Report{Results: []StepResult{{Name: "a"}, {Name: "b", Err: os.ErrNotExist}}}
	if bad.OK() {
		t.Fatal("a failed step must make the report fail")
	}
	if (&Report{}).OK() != true {
		t.Fatal("no steps must be trivially OK")
	}
}

func TestCheckLayout(t *testing.T) {
	if err := checkLayout(repoRoot(t)); err != nil {
		t.Fatalf("real repo must pass layout: %v", err)
	}

	// A temp tree missing every stage script must fail.
	if err := checkLayout(t.TempDir()); err == nil {
		t.Fatal("empty tree must fail layout")
	}

	// Build a minimal but complete tree, then remove exactly one file and one
	// harness file to prove both halves of the check bite.
	root := t.TempDir()
	for _, name := range []string{
		"common.py", "stage_01_modeling.py", "stage_02_topology_uv.py",
		"stage_03_bake.py", "stage_04_pbr.py", "stage_05_unity_export.py",
		"stage_06_render.py", "rig_check.py", "humanoid_rig.py",
		"tripo_to_unity.py", "skill_run.py",
	} {
		touch(t, filepath.Join(root, "scripts", "blender", name))
	}
	for _, name := range []string{"fake_blender.py", "mock_bpy.py", "harness.py"} {
		touch(t, filepath.Join(root, "testing", name))
	}
	if err := checkLayout(root); err != nil {
		t.Fatalf("complete synthetic tree must pass: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "scripts", "blender", "skill_run.py")); err != nil {
		t.Fatal(err)
	}
	if err := checkLayout(root); err == nil || !strings.Contains(err.Error(), "skill_run.py") {
		t.Fatalf("missing skill_run.py must be reported, got %v", err)
	}
	touch(t, filepath.Join(root, "scripts", "blender", "skill_run.py"))
	if err := os.Remove(filepath.Join(root, "testing", "harness.py")); err != nil {
		t.Fatal(err)
	}
	if err := checkLayout(root); err == nil || !strings.Contains(err.Error(), "harness.py") {
		t.Fatalf("missing harness must be reported, got %v", err)
	}
}

func TestCheckGoMod(t *testing.T) {
	if err := checkGoMod(repoRoot(t)); err != nil {
		t.Fatalf("real go.mod must pass: %v", err)
	}
	cases := []struct {
		name string
		body string
		want string // substring expected in the error ("" => must pass)
	}{
		{"clean", "module m\n\ngo " + config.DefaultGo + "\n", ""},
		{"third-party require", "module m\n\ngo " + config.DefaultGo + "\n\nrequire github.com/x/y v1.2.3\n", "third-party"},
		{"require block", "module m\n\ngo " + config.DefaultGo + "\n\nrequire (\n\tgithub.com/x/y v1.2.3\n)\n", "require block"},
		{"wrong go directive", "module m\n\ngo 1.21\n", "go directive"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(c.body), 0o644); err != nil {
			t.Fatal(err)
		}
		err := checkGoMod(dir)
		if c.want == "" {
			if err != nil {
				t.Fatalf("%s: unexpected error %v", c.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: want error containing %q, got %v", c.name, c.want, err)
		}
	}
	if err := checkGoMod(t.TempDir()); err == nil {
		t.Fatal("missing go.mod must fail")
	}
}

func TestCheckGofmt(t *testing.T) {
	if err := checkGofmt(repoRoot(t)); err != nil {
		t.Fatalf("repo must be gofmt-clean: %v", err)
	}
	dir := t.TempDir()
	clean := "package x\n\nfunc F() int { return 1 }\n"
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkGofmt(dir); err != nil {
		t.Fatalf("formatted file must pass: %v", err)
	}
	dirty := "package x\nfunc   F( )  int {  return 1 }\n"
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkGofmt(dir); err == nil || !strings.Contains(err.Error(), "bad.go") {
		t.Fatalf("unformatted file must be reported, got %v", err)
	}
}

func TestPythonFiles(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "scripts", "blender", "stage_01_modeling.py"))
	touch(t, filepath.Join(root, "scripts", "blender", "notes.txt")) // ignored: not .py
	touch(t, filepath.Join(root, "testing", "mock_bpy.py"))
	touch(t, filepath.Join(root, "testing", "harness.py"))
	if err := os.MkdirAll(filepath.Join(root, "testing", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(root, "testing", "sub", "deep.py")) // ignored: non-recursive

	got := pythonFiles(root)
	want := []string{
		filepath.Join(root, "scripts", "blender", "stage_01_modeling.py"),
		filepath.Join(root, "testing", "harness.py"),
		filepath.Join(root, "testing", "mock_bpy.py"),
	}
	if len(got) != len(want) {
		t.Fatalf("pythonFiles = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pythonFiles[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if got := pythonFiles(t.TempDir()); len(got) != 0 {
		t.Fatalf("empty tree must yield no python files, got %v", got)
	}
}

func TestMinCoverage(t *testing.T) {
	t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", "")
	if got := minCoverage(); got != defaultMinCoverage {
		t.Fatalf("default floor = %v, want %v", got, defaultMinCoverage)
	}
	if got := minCoverageLabel(); got != "70" {
		t.Fatalf("label = %q, want 70", got)
	}
	t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", "0")
	if got := minCoverage(); got != 0 {
		t.Fatalf("override to 0 not honoured: %v", got)
	}
	// Garbage / negative values fall back to the default instead of silently
	// disabling the gate.
	for _, v := range []string{"abc", "-5", ""} {
		t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", v)
		if got := minCoverage(); got != defaultMinCoverage {
			t.Fatalf("value %q must fall back to %v, got %v", v, defaultMinCoverage, got)
		}
	}
}

func TestCheckCoverage(t *testing.T) {
	// NOTE: never run `go test ./...` from inside this package — verify is part
	// of that pattern, so it would recurse forever. Reuse the profile written by
	// `assetctl verify` (or skip) and keep `go tool cover` rooted in the module.
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "coverage.out"))
	if err != nil {
		t.Skip("no coverage.out yet; run `assetctl verify` or `go test ./... -coverprofile=coverage.out`")
	}
	profile := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(profile, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", "0")
	if err := checkCoverageFile(root, profile); err != nil {
		t.Fatalf("coverage gate must pass with floor 0: %v", err)
	}
	// An impossible floor must fail with a message naming the floor.
	t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", "150")
	if err := checkCoverageFile(root, profile); err == nil || !strings.Contains(err.Error(), "below the") {
		t.Fatalf("floor 150 must fail, got %v", err)
	}
	// A missing profile must fail explicitly.
	t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", "0")
	if err := checkCoverage(t.TempDir()); err == nil || !strings.Contains(err.Error(), "coverage.out") {
		t.Fatalf("missing profile must be reported, got %v", err)
	}
	// A corrupt profile is rejected by the parser with a precise reason.
	corrupt := filepath.Join(t.TempDir(), "bad.out")
	if err := os.WriteFile(corrupt, []byte("not a profile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkCoverageFile(root, corrupt); err == nil || !strings.Contains(err.Error(), "mode:") {
		t.Fatalf("corrupt profile must be reported, got %v", err)
	}
}

func TestParseCoverage(t *testing.T) {
	profile := "mode: set\n" +
		"github.com/x/a.go:1.1,1.10 2 1\n" +
		"github.com/x/a.go:2.1,2.10 3 0\n" +
		"\n" +
		"github.com/x/b.go:5.1,7.2 4 2\n"
	covered, total, err := parseCoverage([]byte(profile))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if covered != 6 || total != 9 {
		t.Fatalf("covered/total = %d/%d, want 6/9", covered, total)
	}
	// mode: count / atomic behave identically (zero vs non-zero).
	for _, mode := range []string{"count", "atomic"} {
		if _, _, err := parseCoverage([]byte("mode: " + mode + "\n" + "f.go:1.1,1.2 2 7\n")); err != nil {
			t.Fatalf("mode %s: %v", mode, err)
		}
	}
	for _, bad := range []string{
		"",
		"not a profile\n",
		"mode: set\nbroken line\n",
		"mode: set\nf.go:1.1,1.2 x 1\n",
		"mode: set\nf.go:1.1,1.2 2 -1\n",
	} {
		if _, _, err := parseCoverage([]byte(bad)); err == nil {
			t.Fatalf("profile %q must be rejected", bad)
		}
	}
	if _, _, err := parseCoverage([]byte("mode: set\n")); err != nil {
		t.Fatalf("header-only profile is valid (no statements): %v", err)
	}
}

func TestCheckCoverageEmptyProfile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(p, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASSETCTL_VERIFY_MIN_COVERAGE", "0")
	if err := checkCoverageFile(dir, p); err == nil || !strings.Contains(err.Error(), "no statements") {
		t.Fatalf("header-only profile must be rejected, got %v", err)
	}
}

func TestPyCompileRejectsSyntaxError(t *testing.T) {
	py := findPython(repoRoot(t))
	if py == "" {
		t.Skip("no Python interpreter available")
	}
	root := t.TempDir()
	good := filepath.Join(root, "ok.py")
	bad := filepath.Join(root, "bad.py")
	if err := os.WriteFile(good, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("def broken(:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pyCompile(root, py, []string{good}); err != nil {
		t.Fatalf("valid file must compile: %v", err)
	}
	if err := pyCompile(root, py, []string{bad}); err == nil {
		t.Fatal("syntax error must fail py_compile")
	}
	if err := pyCompile(root, "", []string{good}); err == nil {
		t.Fatal("missing interpreter must fail explicitly")
	}
}
