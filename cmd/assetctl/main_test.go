// CLI contract tests (SPEC.md §8). No Blender / Python needed.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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

func writeStageContracts(t *testing.T, dir, id string, fbx []byte) {
	t.Helper()
	cd := filepath.Join(dir, "contracts")
	if err := os.MkdirAll(cd, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(fbx))
	doc := map[string]interface{}{
		"fbxExport": map[string]interface{}{
			"axis_forward": "-Z", "axis_up": "Y", "global_scale": 1.0,
			"apply_unit_scale": true, "add_leaf_bones": false, "path_mode": "COPY",
		},
		"outputs": []map[string]interface{}{{
			"path":  "build/" + id + "/" + id + ".fbx",
			"bytes": len(fbx), "sha256": sum,
		}},
	}
	for _, s := range config.Stages {
		doc["schema"] = 1
		doc["asset"] = id
		doc["stage"] = s
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cd, s+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCmdValidateDoDGate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "build", "hero-knight")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("body")...)
	fbx := []byte("FBX")
	_ = os.WriteFile(filepath.Join(dir, "hero-knight.fbx"), fbx, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "hero-knight_basecolor.png"), png, 0o644)
	if err := dispatch([]string{"validate", dir, "--expect-manifest=false", "--expect-contracts=false"}); err != nil {
		t.Fatalf("valid deliverable rejected: %v", err)
	}
	if err := dispatch([]string{"validate", dir}); err == nil {
		t.Fatal("missing manifest/contracts with DoD gate must fail")
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
	writeStageContracts(t, dir, "hero-knight", fbx)
	if err := dispatch([]string{"validate", dir}); err != nil {
		t.Fatalf("DoD-complete deliverable rejected: %v", err)
	}
	// A tampered artifact must be caught by the SHA-256 contract check.
	_ = os.WriteFile(filepath.Join(dir, "hero-knight.fbx"), []byte("XXXX"), 0o644)
	if err := dispatch([]string{"validate", dir}); err == nil {
		t.Fatal("tampered artifact must fail the contract digest gate")
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

func TestParseParamString(t *testing.T) {
	got := parseParamString("supplier=tripo, maxTriangles=30000 ,bad, =x")
	if got["supplier"] != "tripo" || got["maxTriangles"] != "30000" || len(got) != 2 {
		t.Fatalf("parseParamString = %v", got)
	}
}

func TestCmdSkillsCLI(t *testing.T) {
	root := repoRoot(t)
	for _, args := range [][]string{
		{"skills", "list"},
		{"skills", "list", "--category", "qa"},
		{"skills", "list", "--json"},
		{"skills", "show", "blender-core", "--root", root},
		{"skills", "doctor", "--root", root},
		{"skills", "run", "blender-core", "--dry-run", "--asset", "hero-knight",
			"--root", root, "--param", "supplier=tripo,maxTriangles=30000"},
	} {
		if err := dispatch(args); err != nil {
			t.Fatalf("dispatch(%v): %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"skills", "show", "blender-nope", "--root", root},
		{"skills", "run", "blender-nope", "--dry-run", "--root", root},
		{"skills", "doctor", "--root", t.TempDir()},
		{"skills", "bogus"},
	} {
		if err := dispatch(args); err == nil {
			t.Fatalf("dispatch(%v) should fail", args)
		}
	}
}

func TestCmdSkillsListJSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := dispatch([]string{"skills", "list", "--json"}); err != nil {
			t.Fatalf("skills list --json: %v", err)
		}
	})
	var doc struct {
		Count          int    `json:"count"`
		Shown          int    `json:"shown"`
		BlenderVersion string `json:"blenderVersion"`
		Skills         []struct {
			ID        string   `json:"id"`
			Category  string   `json:"category"`
			Operation string   `json:"operation"`
			Stages    []string `json:"stages"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("skills list --json is not valid JSON: %v\n%s", err, out)
	}
	if doc.Count != 28 || doc.Shown != 28 || len(doc.Skills) != 28 {
		t.Fatalf("count=%d shown=%d skills=%d, want 28", doc.Count, doc.Shown, len(doc.Skills))
	}
	// Only Blender 5.2 is a supported target (SPEC.md §2).
	if doc.BlenderVersion != "5.2" {
		t.Fatalf("blenderVersion = %q, want 5.2", doc.BlenderVersion)
	}
	seen := map[string]bool{}
	for _, s := range doc.Skills {
		if s.ID == "" || s.Category == "" || s.Operation == "" || s.Stages == nil {
			t.Fatalf("incomplete skill row: %+v", s)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate skill id %q", s.ID)
		}
		seen[s.ID] = true
	}
	// --category must filter and keep the JSON envelope valid.
	filtered := captureStdout(t, func() {
		if err := dispatch([]string{"skills", "list", "--json", "--category", "rigging"}); err != nil {
			t.Fatalf("skills list --category: %v", err)
		}
	})
	var fdoc struct {
		Count  int `json:"count"`
		Shown  int `json:"shown"`
		Skills []struct {
			Category string `json:"category"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(filtered), &fdoc); err != nil {
		t.Fatalf("filtered JSON invalid: %v", err)
	}
	if fdoc.Count != 28 || fdoc.Shown == 0 || fdoc.Shown == 28 || fdoc.Shown != len(fdoc.Skills) {
		t.Fatalf("filtered count=%d shown=%d rows=%d", fdoc.Count, fdoc.Shown, len(fdoc.Skills))
	}
	for _, s := range fdoc.Skills {
		if s.Category != "rigging" {
			t.Fatalf("category filter leaked %q", s.Category)
		}
	}
}

// captureStdout runs fn with os.Stdout redirected and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		buf := new(strings.Builder)
		_, _ = io.Copy(buf, r)
		done <- buf.String()
	}()
	defer func() { os.Stdout = orig }()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func TestReorderFlagsKeepsSkillsSubcommand(t *testing.T) {
	got := reorderFlags([]string{"skills", "run", "blender-core", "--asset", "hero", "--dry-run"})
	want := []string{"skills", "run", "-asset", "hero", "-dry-run", "blender-core"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("reorderFlags(skills) = %v, want %v", got, want)
	}
	if !flagTakesValue("-category") || !flagTakesValue("-param") {
		t.Fatal("new value flags must be registered in flagTakesValue")
	}
}
