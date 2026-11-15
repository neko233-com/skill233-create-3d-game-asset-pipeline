package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

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
