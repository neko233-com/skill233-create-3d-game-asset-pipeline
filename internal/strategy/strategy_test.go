package strategy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

func TestRegistryAndDefault(t *testing.T) {
	ids := IDs()
	if len(ids) != 3 {
		t.Fatalf("expected 3 strategies, got %v", ids)
	}
	if DefaultID() != "blender" {
		t.Fatalf("default = %q, want blender", DefaultID())
	}
	if _, err := Get("nope"); err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestBlenderDefaultProducesProcedural(t *testing.T) {
	s, _ := Get("blender")
	ctx := &RunContext{RootDir: ".", Config: config.Default("hero-knight")}
	res, err := s.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.InputPath != "" {
		t.Fatalf("blender strategy should be procedural, got input %q", res.InputPath)
	}
}

func TestTripoManualStepWhenNoGLB(t *testing.T) {
	root := t.TempDir()
	c := config.Default("hero-knight")
	c.Strategy = "tripo"
	s, _ := Get("tripo")
	ctx := &RunContext{RootDir: root, Config: c}
	_, err := s.Acquire(ctx)
	if err == nil {
		t.Fatal("expected manual-step error")
	}
}

func TestTripoFindsGLB(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "raw"), 0o755); err != nil {
		t.Fatal(err)
	}
	glb := filepath.Join(root, "raw", "hero-knight.glb")
	if err := os.WriteFile(glb, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := config.Default("hero-knight")
	c.Strategy = "tripo"
	s, _ := Get("tripo")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	if res.InputPath != glb {
		t.Fatalf("got %q, want %q", res.InputPath, glb)
	}
}

func TestImportCopy(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "somewhere", "model.glb")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("model-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := config.Default("hero-knight")
	c.Strategy = "import"
	c.Source = "somewhere/model.glb"
	s, _ := Get("import")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(res.InputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "model-bytes" {
		t.Fatal("copy mismatch")
	}
}
