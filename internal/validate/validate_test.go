package validate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

// write creates a structurally plausible artifact: PNG/GLB carry their magic
// bytes (validate.go now checks deliverable integrity, not just existence).
func write(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	var data []byte
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png":
		data = append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("IHDR-test")...)
	case ".glb":
		data = append([]byte("glTF"), []byte{2, 0, 0, 0, 32, 0, 0, 0}...)
	default:
		data = []byte("assetctl-test-artifact")
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateOK(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hero-knight")
	write(t, filepath.Join(dir, "hero-knight.fbx"))
	write(t, filepath.Join(dir, "hero-knight_basecolor.png"))
	write(t, filepath.Join(dir, "hero-knight_normal.png"))
	r, err := Validate(dir, Options{ExpectHumanoid: true, ExpectTextures: true})
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK() {
		t.Fatalf("unexpected issues: %v", r.Issues)
	}
}

func TestValidateMissingModel(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hero-knight")
	write(t, filepath.Join(dir, "hero-knight_basecolor.png"))
	r, err := Validate(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.OK() {
		t.Fatal("expected failure for missing model")
	}
}

func TestValidateBadTextureName(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hero-knight")
	write(t, filepath.Join(dir, "hero-knight.fbx"))
	write(t, filepath.Join(dir, "random.png"))
	r, err := Validate(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.OK() {
		t.Fatal("expected failure for bad texture name")
	}
}

func TestValidatePreviewExempt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hero-knight")
	write(t, filepath.Join(dir, "hero-knight.fbx"))
	write(t, filepath.Join(dir, "hero-knight_basecolor.png"))
	write(t, filepath.Join(dir, "preview.png")) // stage-6 QA render, not a PBR map
	r, err := Validate(dir, Options{ExpectTextures: true})
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK() {
		t.Fatalf("preview.png must be exempt, got: %v", r.Issues)
	}
}

func TestValidateHumanoidMarker(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "prop-sword")
	write(t, filepath.Join(dir, "prop-sword.fbx"))
	write(t, filepath.Join(dir, "prop-sword_basecolor.png"))
	r, err := Validate(dir, Options{ExpectHumanoid: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.OK() {
		t.Fatal("expected humanoid marker failure")
	}
	write(t, filepath.Join(dir, ".humanoid"))
	r, err = Validate(dir, Options{ExpectHumanoid: true})
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK() {
		t.Fatalf("marker should satisfy: %v", r.Issues)
	}
}

func TestValidateRejectsCorruptArtifacts(t *testing.T) {
	cases := []struct {
		name string
		file string
		data []byte
	}{
		{"empty model", "hero-knight.fbx", []byte{}},
		{"empty texture", "hero-knight_basecolor.png", []byte{}},
		{"bad png magic", "hero-knight_normal.png", []byte("not a png at all")},
		{"bad glb magic", "hero-knight.glb", []byte("NOPE-binary-payload")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "hero-knight")
			write(t, filepath.Join(dir, "hero-knight.fbx"))
			write(t, filepath.Join(dir, "hero-knight_basecolor.png"))
			if err := os.WriteFile(filepath.Join(dir, tc.file), tc.data, 0o644); err != nil {
				t.Fatal(err)
			}
			r, err := Validate(dir, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if r.OK() {
				t.Fatalf("corrupt artifact %s must fail, issues=%v", tc.file, r.Issues)
			}
		})
	}
}

// sixOKManifest is a DoD-complete manifest.json (all six stages "ok").
const sixOKManifest = `{
  "assetId": "hero-knight",
  "strategy": "tripo",
  "stages": {
    "modeling": "ok",
    "topology-uv": "ok",
    "bake": "ok",
    "pbr": "ok",
    "unity-export": "ok",
    "render": "ok"
  }
}`

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func baseDeliverable(t *testing.T, dir string) {
	t.Helper()
	write(t, filepath.Join(dir, "hero-knight.fbx"))
	write(t, filepath.Join(dir, "hero-knight_basecolor.png"))
}

// TestValidateManifestGate proves the DoD gate: all six stages must be "ok".
// TestCheckContracts verifies the SHA-256 stage-contract gate in isolation.
func TestCheckContracts(t *testing.T) {
	root := t.TempDir()
	id := "hero-knight"
	dir := filepath.Join(root, "build", id)
	cd := filepath.Join(dir, "contracts")
	if err := os.MkdirAll(cd, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, id+".fbx"))
	fbx, _ := os.ReadFile(filepath.Join(dir, id+".fbx"))
	digest := fmt.Sprintf("%x", sha256.Sum256(fbx))
	fbxKeys := map[string]interface{}{
		"axis_forward": "-Z", "axis_up": "Y", "global_scale": 1.0,
		"apply_unit_scale": true, "add_leaf_bones": false, "path_mode": "COPY",
	}
	makeContract := func(stage, sha string) []byte {
		doc := map[string]interface{}{
			"schema": 1, "asset": id, "stage": stage,
			"fbxExport": fbxKeys,
			"outputs": []map[string]interface{}{{
				"path":  "build/" + id + "/" + id + ".fbx",
				"bytes": len(fbx), "sha256": sha,
			}},
		}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	for _, stage := range config.Stages {
		if err := os.WriteFile(filepath.Join(cd, stage+".json"), makeContract(stage, digest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if issues := checkContracts(dir, true); len(issues) != 0 {
		t.Fatalf("valid contracts rejected: %v", issues)
	}
	// A single corrupted digest must be reported.
	if err := os.WriteFile(filepath.Join(cd, "bake.json"), makeContract("bake", strings.Repeat("0", 64)), 0o644); err != nil {
		t.Fatal(err)
	}
	issues := checkContracts(dir, true)
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, " "), "SHA-256") {
		t.Fatalf("tampered digest not detected: %v", issues)
	}
	// A missing stage contract must be reported.
	_ = os.Remove(filepath.Join(cd, "render.json"))
	issues = checkContracts(dir, true)
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, " "), "missing stage contract") {
		t.Fatalf("missing contract not detected: %v", issues)
	}
	// Without the flag, a missing contracts directory is not an error.
	empty := filepath.Join(root, "build", "prop-sword")
	_ = os.MkdirAll(empty, 0o755)
	if issues := checkContracts(empty, false); len(issues) != 0 {
		t.Fatalf("optional contracts should not fail: %v", issues)
	}
	if issues := checkContracts(empty, true); len(issues) == 0 {
		t.Fatal("required contracts must fail when directory is missing")
	}
}

func TestValidateManifestGate(t *testing.T) {
	root := t.TempDir()

	t.Run("six ok passes when required", func(t *testing.T) {
		dir := filepath.Join(root, "a-hero-knight")
		baseDeliverable(t, dir)
		writeManifest(t, dir, sixOKManifest)
		r, err := Validate(dir, Options{ExpectManifest: true})
		if err != nil {
			t.Fatal(err)
		}
		if !r.OK() {
			t.Fatalf("six-ok manifest should pass: %v", r.Issues)
		}
	})

	t.Run("one failed stage fails even when not required", func(t *testing.T) {
		dir := filepath.Join(root, "b-hero-knight")
		baseDeliverable(t, dir)
		writeManifest(t, dir, `{"stages":{"modeling":"ok","topology-uv":"ok","bake":"failed","pbr":"ok","unity-export":"ok","render":"ok"}}`)
		r, err := Validate(dir, Options{}) // ExpectManifest false — a present-but-bad manifest still fails
		if err != nil {
			t.Fatal(err)
		}
		if r.OK() {
			t.Fatal("a failed stage must fail validation")
		}
	})

	t.Run("missing manifest only fails when required", func(t *testing.T) {
		dir := filepath.Join(root, "c-hero-knight")
		baseDeliverable(t, dir)
		r, err := Validate(dir, Options{ExpectManifest: true})
		if err != nil {
			t.Fatal(err)
		}
		if r.OK() {
			t.Fatal("missing manifest + ExpectManifest must fail")
		}
		r2, err := Validate(dir, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if !r2.OK() {
			t.Fatalf("missing manifest without ExpectManifest should pass: %v", r2.Issues)
		}
	})

	t.Run("object-form statuses accepted", func(t *testing.T) {
		dir := filepath.Join(root, "d-hero-knight")
		baseDeliverable(t, dir)
		writeManifest(t, dir, `{"stages":{"modeling":{"status":"ok"},"topology-uv":{"status":"ok"},"bake":{"status":"ok"},"pbr":{"status":"ok"},"unity-export":{"status":"ok"},"render":{"status":"ok"}}}`)
		r, err := Validate(dir, Options{ExpectManifest: true})
		if err != nil {
			t.Fatal(err)
		}
		if !r.OK() {
			t.Fatalf("object statuses should pass: %v", r.Issues)
		}
	})

	t.Run("missing one stage is reported", func(t *testing.T) {
		dir := filepath.Join(root, "e-hero-knight")
		baseDeliverable(t, dir)
		writeManifest(t, dir, `{"stages":{"modeling":"ok","topology-uv":"ok","bake":"ok","pbr":"ok","unity-export":"ok"}}`)
		r, err := Validate(dir, Options{ExpectManifest: true})
		if err != nil {
			t.Fatal(err)
		}
		if r.OK() {
			t.Fatal("missing render stage must fail")
		}
		found := false
		for _, i := range r.Issues {
			if strings.Contains(i, "render") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a missing-stage issue, got %v", r.Issues)
		}
	})
}
