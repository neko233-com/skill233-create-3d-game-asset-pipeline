package runner

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

func repoScriptsDir(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "scripts", "blender"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeConfig(t *testing.T, root, id string) string {
	t.Helper()
	dir := filepath.Join(root, "assets", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "asset.json")
	if err := config.Default(id).Save(p); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestRunDryRunProvesResumeInput fixes the --from bug: a resumed run must feed
// the previously-completed stage's output as --in to the first selected stage.
func TestRunDryRunProvesResumeInput(t *testing.T) {
	root := t.TempDir()
	cfg := writeConfig(t, root, "hero-knight")
	var out bytes.Buffer
	m, err := Run(Options{
		RootDir: root, ConfigPath: cfg, AssetID: "hero-knight",
		From: "topology-uv", DryRun: true,
		Stdout: &out, Stderr: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.Stages["modeling"] != "skipped" {
		t.Fatalf("modeling should be skipped, got %q", m.Stages["modeling"])
	}
	want := filepath.Join(root, "work", "hero-knight", "01-modeling.blend")
	if !strings.Contains(out.String(), "--in "+want) {
		t.Fatalf("resume must read %s as --in; log:\n%s", want, out.String())
	}
}

// TestRunDryRunOnlyStage feeds stage N-1's output when --only selects a single
// later stage.
func TestRunDryRunOnlyStage(t *testing.T) {
	root := t.TempDir()
	cfg := writeConfig(t, root, "hero-knight")
	var out bytes.Buffer
	_, err := Run(Options{
		RootDir: root, ConfigPath: cfg, AssetID: "hero-knight",
		Only: []string{"unity-export"}, DryRun: true,
		Stdout: &out, Stderr: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "work", "hero-knight", "04-pbr.blend")
	if !strings.Contains(out.String(), "--in "+want) {
		t.Fatalf("--only unity-export must read %s as --in; log:\n%s", want, out.String())
	}
}

// TestRunDryRunWritesNothing proves dry-run is informational only (no manifest,
// no work/build dirs created, no Blender lookup).
func TestRunDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	cfg := writeConfig(t, root, "hero-knight")
	var out bytes.Buffer
	_, err := Run(Options{
		RootDir: root, ConfigPath: cfg, AssetID: "hero-knight", DryRun: true,
		Stdout: &out, Stderr: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "hero-knight", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write manifest.json (err=%v)", err)
	}
	if n := strings.Count(out.String(), "[dry-run] blender"); n != 6 {
		t.Fatalf("expected 6 dry-run stage lines, got %d:\n%s", n, out.String())
	}
}

// ---------------------------------------------------------------------------
// Non-dry-run tests. helperLauncher re-executes this test binary so Run()
// actually spawns six stage processes on every OS with NO Blender / Python.
// ---------------------------------------------------------------------------
type helperLauncher struct{}

func (helperLauncher) Command(argv []string, dir string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestHelperTouchOut", "--"}, argv...)...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	cmd.Dir = dir
	return cmd
}

func (helperLauncher) Label() string { return "test-helper" }

// TestHelperTouchOut is the child process body; it only creates the declared
// --out file (with valid magic bytes where applicable) and exits.
func TestHelperTouchOut(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	out := ""
	for i, a := range args {
		if a == "--out" && i+1 < len(args) {
			out = args[i+1]
			break
		}
	}
	if out == "" { // rig_check.py has no --out
		os.Exit(0)
	}
	_ = os.MkdirAll(filepath.Dir(out), 0o755)
	data := []byte("helper-artifact")
	lower := strings.ToLower(out)
	if strings.HasSuffix(lower, ".png") {
		data = append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, data...)
	} else if strings.HasSuffix(lower, ".glb") {
		data = append([]byte("glTF"), data...)
	}
	_ = os.WriteFile(out, data, 0o644)
	os.Exit(0)
}

func writeFixtureConfig(t *testing.T, root, id string, humanoid bool) string {
	t.Helper()
	c := config.Default(id)
	c.Handoff.Mode = "fixture"
	c.Humanoid = humanoid
	if humanoid {
		c.Kind = "character"
		c.Animations = []string{"idle", "walk", "attack"}
	}
	p := filepath.Join(root, "assets", id, "asset.json")
	if err := c.Save(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func assertManifestAllOK(t *testing.T, root, id string) *Manifest {
	t.Helper()
	m, err := readManifest(filepath.Join(root, "build", id, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range config.Stages {
		if m.Stages[stage] != "ok" {
			t.Fatalf("stage %s = %q, want ok (manifest=%+v)", stage, m.Stages[stage], m.Stages)
		}
	}
	return m
}

func TestRunFullSixStagesNonDryRun(t *testing.T) {
	root := t.TempDir()
	cfg := writeFixtureConfig(t, root, "hero-knight", false)
	m, err := Run(Options{
		RootDir: root, ConfigPath: cfg, AssetID: "hero-knight",
		Launcher: helperLauncher{}, ScriptsDir: repoScriptsDir(t), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.Strategy != "tripo" || m.BlenderVersion != "5.2" {
		t.Fatalf("manifest header drift: %+v", m)
	}
	assertManifestAllOK(t, root, "hero-knight")
	for _, p := range []string{
		filepath.Join(root, "raw", "hero-knight.glb"),
		filepath.Join(root, "work", "hero-knight", "01-modeling.blend"),
		filepath.Join(root, "build", "hero-knight", "hero-knight.fbx"),
		filepath.Join(root, "build", "hero-knight", "preview.png"),
	} {
		if st, err := os.Stat(p); err != nil || st.Size() == 0 {
			t.Fatalf("missing/empty artifact %s (err=%v)", p, err)
		}
	}
}

// TestRunResumePreservesOKStages is the regression test for the resume bug:
// --from / --only must not downgrade already-completed stages to "skipped".
func TestRunResumePreservesOKStages(t *testing.T) {
	root := t.TempDir()
	cfg := writeFixtureConfig(t, root, "hero-knight", false)
	base := Options{RootDir: root, ConfigPath: cfg, AssetID: "hero-knight",
		Launcher: helperLauncher{}, ScriptsDir: repoScriptsDir(t), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if _, err := Run(base); err != nil {
		t.Fatal(err)
	}
	base.From = "unity-export"
	if _, err := Run(base); err != nil {
		t.Fatal(err)
	}
	assertManifestAllOK(t, root, "hero-knight")
	base.From = ""
	base.Only = []string{"render"}
	if _, err := Run(base); err != nil {
		t.Fatal(err)
	}
	assertManifestAllOK(t, root, "hero-knight")
}

func TestRunHumanoidGateInvoked(t *testing.T) {
	root := t.TempDir()
	cfg := writeFixtureConfig(t, root, "hero-knight", true)
	var out bytes.Buffer
	_, err := Run(Options{RootDir: root, ConfigPath: cfg, AssetID: "hero-knight",
		Launcher: helperLauncher{}, ScriptsDir: repoScriptsDir(t), Stdout: &out, Stderr: &bytes.Buffer{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[rig] humanoid rig_check OK") {
		t.Fatalf("humanoid gate not run:\n%s", out.String())
	}
}

func TestFindBlenderExplicitWithVersionProbeSkipped(t *testing.T) {
	t.Setenv("ASSETCTL_SKIP_BLENDER_VERSION", "1")
	dir := t.TempDir()
	fake := filepath.Join(dir, "blender")
	if err := os.WriteFile(fake, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindBlender(fake)
	if err != nil {
		t.Fatal(err)
	}
	if got != fake {
		t.Fatalf("FindBlender = %q, want %q", got, fake)
	}
}

func TestBlenderCandidatesPlatformCoverage(t *testing.T) {
	got := blenderCandidates("/explicit/blender")
	if len(got) == 0 || got[0] != "/explicit/blender" {
		t.Fatalf("explicit candidate must come first: %v", got)
	}
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" && runtime.GOOS != "darwin" &&
		runtime.GOOS != "freebsd" {
		return
	}
	// Every supported desktop OS contributes at least one install-location
	// candidate (plus explicit + PATH + env).
	if runtime.GOOS != "freebsd" && len(got) < 2 {
		t.Fatalf("no platform-specific Blender candidates on %s: %v", runtime.GOOS, got)
	}
}

func TestSelectStagesContract(t *testing.T) {
	defs := stageDefs()
	if got := selectStages(defs, "", nil); len(got) != 6 {
		t.Fatalf("full run should select 6 stages, got %d", len(got))
	}
	if got := selectStages(defs, "pbr", nil); len(got) != 3 || got[0].id != "pbr" {
		t.Fatalf("--from pbr mismatch: %+v", got)
	}
	if got := selectStages(defs, "", []string{"bake", "render"}); len(got) != 2 ||
		got[0].id != "bake" || got[1].id != "render" {
		t.Fatalf("--only must preserve pipeline order: %+v", got)
	}
}
