// End-to-end integration: the REAL six bpy stages run through
// testing/fake_blender.py (mock_bpy + mathutils) with NO Blender installed.
// Every stage is still a separate process with the exact blender-style argv,
// so this exercises the production file contract, not a shortcut.
package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/validate"
)

func e2eRepoRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func e2eLauncher(t *testing.T) ScriptLauncher {
	t.Helper()
	l, err := NewScriptLauncher(e2eRepoRoot(t))
	if err != nil {
		t.Fatalf("fake_blender.py harness unavailable (set ASSETCTL_PYTHON): %v", err)
	}
	return l
}

func e2eConfig(t *testing.T, root, id string, humanoid bool) string {
	t.Helper()
	c := config.Default(id)
	c.Handoff.Mode = "fixture"
	c.Humanoid = humanoid
	if humanoid {
		c.Kind = "character"
		c.Animations = []string{"idle", "walk", "attack"}
	}
	c.Budget.MaxTextureSize = 256 // keep the stdlib PNG writer cheap in CI
	p := filepath.Join(root, "assets", id, "asset.json")
	if err := c.Save(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func e2eOptions(t *testing.T, root, cfg string, l ScriptLauncher) (Options, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return Options{
		RootDir:    root,
		ConfigPath: cfg,
		AssetID:    "hero-knight",
		Launcher:   l,
		ScriptsDir: filepath.Join(e2eRepoRoot(t), "scripts", "blender"),
		Stdout:     &out,
		Stderr:     &bytes.Buffer{},
	}, &out
}

func TestE2ESixStagesWithFakeBlender(t *testing.T) {
	repo := e2eRepoRoot(t)
	l := e2eLauncher(t)
	root := t.TempDir()
	cfg := e2eConfig(t, root, "hero-knight", true)
	opts, out := e2eOptions(t, root, cfg, l)

	m, err := Run(opts)
	if err != nil {
		t.Fatalf("E2E run failed: %v\n%s", err, out.String())
	}
	if m.Strategy != "tripo" || m.BlenderVersion != "5.2" ||
		m.TripoURL != "https://studio.tripo3d.com/" {
		t.Fatalf("manifest header drift: %+v", m)
	}
	assertManifestAllOK(t, root, "hero-knight")

	buildDir := filepath.Join(root, "build", "hero-knight")
	for _, rel := range []string{
		filepath.Join("raw", "hero-knight.glb"),
		filepath.Join("work", "hero-knight", "01-modeling.blend"),
		filepath.Join("work", "hero-knight", "04-pbr.blend"),
		filepath.Join("build", "hero-knight", "hero-knight.fbx"),
		filepath.Join("build", "hero-knight", "hero-knight_basecolor.png"),
		filepath.Join("build", "hero-knight", "hero-knight_normal.png"),
		filepath.Join("build", "hero-knight", "hero-knight_roughness.png"),
		filepath.Join("build", "hero-knight", "hero-knight_metallic.png"),
		filepath.Join("build", "hero-knight", "preview.png"),
	} {
		st, err := os.Stat(filepath.Join(root, rel))
		if err != nil || st.Size() == 0 {
			t.Fatalf("missing/empty artifact %s (err=%v)", rel, err)
		}
	}

	// Six stage contracts (plus rig-check for humanoid assets) with real bytes.
	for _, stage := range config.Stages {
		p := filepath.Join(buildDir, "contracts", stage+".json")
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("missing contract %s: %v", stage, err)
		}
		var c struct {
			Schema  int `json:"schema"`
			Outputs []struct {
				Bytes  int64  `json:"bytes"`
				SHA256 string `json:"sha256"`
			} `json:"outputs"`
			FBXExport map[string]interface{} `json:"fbxExport"`
		}
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatalf("contract %s invalid: %v", stage, err)
		}
		if c.Schema != 1 || len(c.Outputs) == 0 {
			t.Fatalf("contract %s has no verified outputs: %+v", stage, c)
		}
		if c.FBXExport["axis_forward"] != "-Z" || c.FBXExport["axis_up"] != "Y" {
			t.Fatalf("contract %s lost the Unity FBX axes: %v", stage, c.FBXExport)
		}
	}
	if _, err := os.Stat(filepath.Join(buildDir, "contracts", "rig-check.json")); err != nil {
		t.Fatalf("humanoid gate did not publish its contract: %v", err)
	}
	if !strings.Contains(out.String(), "@@JSON@@") {
		t.Fatal("stage scripts must emit structured @@JSON@@ contract lines")
	}

	// Full DoD gate, including SHA-256 re-verification of every contracted file.
	r, err := validate.Validate(buildDir, validate.Options{
		ExpectHumanoid: true, ExpectTextures: true,
		ExpectManifest: true, ExpectContracts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK() {
		t.Fatalf("E2E deliverable failed DoD: %v", r.Issues)
	}

	// The mock must be a strict Blender 5.2 stand-in.
	if !strings.Contains(out.String(), "[launcher] fake_blender.py") {
		t.Fatalf("E2E did not use the no-Blender launcher:\n%s", out.String())
	}
	_ = repo
}

func TestE2EResumeWithFakeBlender(t *testing.T) {
	l := e2eLauncher(t)
	root := t.TempDir()
	cfg := e2eConfig(t, root, "hero-knight", false)
	opts, _ := e2eOptions(t, root, cfg, l)
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	opts.From = "render"
	if _, err := Run(opts); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	// Earlier stages must stay "ok" so the DoD manifest gate still passes.
	assertManifestAllOK(t, root, "hero-knight")
}

func TestE2EBlenderStrategyProcedural(t *testing.T) {
	l := e2eLauncher(t)
	root := t.TempDir()
	cfg := e2eConfig(t, root, "hero-knight", false)
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.Strategy = "blender" // alternative supplier, no raw GLB needed
	if err := c.Save(cfg); err != nil {
		t.Fatal(err)
	}
	opts, out := e2eOptions(t, root, cfg, l)
	if _, err := Run(opts); err != nil {
		t.Fatalf("blender alternative strategy failed: %v\n%s", err, out.String())
	}
	assertManifestAllOK(t, root, "hero-knight")
}
