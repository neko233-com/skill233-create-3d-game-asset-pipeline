package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
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
