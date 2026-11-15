// CLI contract tests (SPEC.md §8). No Blender / Python needed.
package main

import (
	"encoding/json"
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

func TestReorderFlags(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"init", "hero-knight", "--humanoid"},
			[]string{"init", "-humanoid", "hero-knight"}},
		{[]string{"run", "hero-knight", "--from", "topology-uv", "--dry-run"},
			[]string{"run", "-from", "topology-uv", "-dry-run", "hero-knight"}},
		{[]string{"reference", "prepare", "in.png", "out", "--size", "512"},
			[]string{"reference", "prepare", "-size", "512", "in.png", "out"}},
		{[]string{"blender", "x.py", "--", "--asset", "hero"}, // untouched passthrough
			[]string{"blender", "x.py", "--", "--asset", "hero"}},
	}
	for _, tc := range cases {
		got := reorderFlags(tc.in)
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Fatalf("reorderFlags(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestFlagTakesValue(t *testing.T) {
	for _, f := range []string{"-strategy", "-supplier", "-kind", "-root", "-from", "-only", "-blender", "-size", "-bg", "-handoff", "-source"} {
		if !flagTakesValue(f) {
			t.Fatalf("%s must consume a value", f)
		}
	}
	for _, f := range []string{"-dry-run", "-humanoid", "-expect-humanoid", "-expect-textures"} {
		if flagTakesValue(f) {
			t.Fatalf("%s is a bool flag and must not consume a value", f)
		}
	}
}

func TestDispatchReadonlyCommands(t *testing.T) {
	for _, args := range [][]string{
		{"version"}, {"help"}, {"strategies"}, {"stages"},
	} {
		if err := dispatch(args); err != nil {
			t.Fatalf("dispatch(%v): %v", args, err)
		}
	}
	if err := dispatch([]string{"no-such-command"}); err == nil {
		t.Fatal("unknown command must fail")
	}
}

func TestCmdInitDefaultsToTripo(t *testing.T) {
	root := t.TempDir()
	if err := dispatch([]string{"init", "hero-knight", "--root", root, "--humanoid"}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "assets", "hero-knight", "asset.json")
	c, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Strategy != "tripo" || c.Handoff.Mode != "manual" {
		t.Fatalf("init defaults drift: strategy=%q handoff=%q", c.Strategy, c.Handoff.Mode)
	}
	if !c.Humanoid || len(c.Animations) != 3 {
		t.Fatalf("--humanoid did not set rig/clip defaults: %+v", c)
	}
}

func TestCmdInitHandoffMatrix(t *testing.T) {
	for _, mode := range []string{"manual", "fixture"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if err := dispatch([]string{"init", "prop-sword", "--root", root, "--handoff", mode}); err != nil {
				t.Fatal(err)
			}
			c, err := config.Load(filepath.Join(root, "assets", "prop-sword", "asset.json"))
			if err != nil {
				t.Fatal(err)
			}
			if c.Handoff.Mode != mode {
				t.Fatalf("handoff = %q, want %q", c.Handoff.Mode, mode)
			}
		})
	}
	// file handoff without a source must be rejected by config validation.
	root := t.TempDir()
	if err := dispatch([]string{"init", "prop-sword", "--root", root, "--handoff", "file"}); err == nil {
		t.Fatal("--handoff file without --source must fail")
	}
	if err := dispatch([]string{"init", "prop-sword", "--root", root,
		"--handoff", "file", "--source", "drops/model.glb"}); err != nil {
		t.Fatalf("valid file handoff rejected: %v", err)
	}
	if err := dispatch([]string{"init", "BAD_ID", "--root", root}); err == nil {
		t.Fatal("uppercase asset id must be rejected")
	}
}

func TestCmdInitSupplierAlias(t *testing.T) {
	root := t.TempDir()
	if err := dispatch([]string{"init", "prop-sword", "--root", root,
		"--strategy", "tripo", "--supplier", "blender"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(filepath.Join(root, "assets", "prop-sword", "asset.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Strategy != "blender" {
		t.Fatalf("--supplier must override --strategy, got %q", c.Strategy)
	}
}

func TestCmdRunRejectsBadFlags(t *testing.T) {
	root := t.TempDir()
	if err := dispatch([]string{"init", "hero-knight", "--root", root, "--handoff", "fixture"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"run", "hero-knight", "--root", root, "--from", "not-a-stage"},
		{"run", "hero-knight", "--root", root, "--only", "modeling,bogus"},
		{"run", "--root", root},
	} {
		if err := dispatch(args); err == nil {
			t.Fatalf("dispatch(%v) should fail", args)
		}
	}
	// dry-run is informational: no manifest, no Blender lookup.
	if err := dispatch([]string{"run", "hero-knight", "--root", root, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "hero-knight", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not create a manifest (err=%v)", err)
	}
}

func TestCmdValidateDoDGate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hero-knight")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("body")...)
	_ = os.WriteFile(filepath.Join(dir, "hero-knight.fbx"), []byte("FBX"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "hero-knight_basecolor.png"), png, 0o644)
	if err := dispatch([]string{"validate", dir, "--expect-manifest=false"}); err != nil {
		t.Fatalf("valid deliverable rejected: %v", err)
	}
	if err := dispatch([]string{"validate", dir}); err == nil {
		t.Fatal("missing manifest with DoD gate must fail")
	}
	doc := map[string]interface{}{
		"assetId": "hero-knight",
		"stages":  map[string]string{},
	}
	for _, s := range config.Stages {
		doc["stages"].(map[string]string)[s] = "ok"
	}
	data, _ := json.Marshal(doc)
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
	if err := dispatch([]string{"validate", dir}); err != nil {
		t.Fatalf("DoD-complete deliverable rejected: %v", err)
	}
}

func TestCmdDoctor(t *testing.T) {
	t.Setenv("ASSETCTL_SKIP_BLENDER_VERSION", "1")
	if err := dispatch([]string{"doctor", "--root", repoRoot(t)}); err != nil {
		t.Fatalf("doctor must pass on a complete repo: %v", err)
	}
	// A directory lacking scripts/blender is a genuine repo defect.
	if err := dispatch([]string{"doctor", "--root", t.TempDir()}); err == nil {
		t.Fatal("doctor must fail when stage scripts are missing")
	}
}
