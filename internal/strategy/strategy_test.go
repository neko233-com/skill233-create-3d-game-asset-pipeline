package strategy

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

func TestRegistryAndDefault(t *testing.T) {
	ids := IDs()
	if len(ids) != 3 {
		t.Fatalf("expected 3 suppliers, got %v", ids)
	}
	if DefaultID() != "tripo" {
		t.Fatalf("default = %q, want tripo (production supplier)", DefaultID())
	}
	if _, err := Get("nope"); err == nil {
		t.Fatal("expected error for unknown strategy")
	}
	// Tripo is the default; blender and import are alternatives.
	tripo, _ := Get("tripo")
	blender, _ := Get("blender")
	imp, _ := Get("import")
	if !tripo.IsDefault() || blender.IsDefault() || imp.IsDefault() {
		t.Fatal("exactly one supplier (tripo) must be IsDefault()")
	}
}

func TestBlenderProducesProcedural(t *testing.T) {
	s, _ := Get("blender")
	ctx := &RunContext{RootDir: ".", Config: config.Default("hero-knight")}
	res, err := s.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.InputPath != "" {
		t.Fatalf("blender supplier should be procedural, got input %q", res.InputPath)
	}
}

func TestTripoManualStepWhenNoGLB(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "")
	c := config.Default("hero-knight")
	s, _ := Get("tripo")
	ctx := &RunContext{RootDir: root, Config: c}
	_, err := s.Acquire(ctx)
	if err == nil {
		t.Fatal("expected manual-step error")
	}
	if !IsManualStep(err) {
		t.Fatalf("expected ErrManualStepRequired, got %v", err)
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
	s, _ := Get("tripo")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	if res.InputPath != glb {
		t.Fatalf("got %q, want %q", res.InputPath, glb)
	}
}

func TestTripoFixtureHandoffGeneratesValidGLB(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "")
	c := config.Default("hero-knight")
	c.Handoff.Mode = "fixture"
	s, _ := Get("tripo")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "raw", "hero-knight.glb")
	if res.InputPath != want {
		t.Fatalf("got %q, want %q", res.InputPath, want)
	}
	data, err := os.ReadFile(res.InputPath)
	if err != nil {
		t.Fatal(err)
	}
	// Validate GLB header: magic "glTF", version 2, declared length == file size.
	if len(data) < 20 {
		t.Fatal("GLB too short")
	}
	if string(data[0:4]) != "glTF" {
		t.Fatalf("bad GLB magic %q", data[0:4])
	}
	if binary.LittleEndian.Uint32(data[4:8]) != 2 {
		t.Fatal("bad GLB version")
	}
	if binary.LittleEndian.Uint32(data[8:12]) != uint32(len(data)) {
		t.Fatal("GLB length mismatch")
	}
	// Deterministic: a second acquire (with the file removed) yields identical bytes.
	_ = os.Remove(res.InputPath)
	res2, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(res2.InputPath)
	if string(data) != string(data2) {
		t.Fatal("fixture GLB is not deterministic")
	}
}

func TestTripoFileHandoff(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "upstream", "drop.glb")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("tripo-drop"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "")
	c := config.Default("hero-knight")
	c.Handoff = config.Handoff{Mode: "file", Source: "upstream/drop.glb"}
	s, _ := Get("tripo")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(res.InputPath)
	if string(data) != "tripo-drop" {
		t.Fatal("file handoff copy mismatch")
	}
}

func TestTripoEnvHandoffForcesFixture(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "fixture")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "")
	c := config.Default("hero-knight") // handoff.mode = manual in config
	s, _ := Get("tripo")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.InputPath); err != nil {
		t.Fatalf("env handoff override did not produce a GLB: %v", err)
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

// IsManualStep mirrors runner.IsManualStep without importing runner (avoids a
// package cycle in this test); the real assertion lives in runner tests too.
func IsManualStep(err error) bool {
	return errors.Is(err, ErrManualStepRequired)
}

// TestFixtureGLBContainsRealTriangle proves the fixture is a complete,
// importable glTF 2.0 mesh (not an empty stand-in). Real Blender 5.2 must be
// able to import geometry from it, otherwise every downstream stage is empty.
func TestFixtureGLBContainsRealTriangle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "")
	c := config.Default("hero-knight")
	c.Handoff.Mode = "fixture"
	s, _ := Get("tripo")
	res, err := s.Acquire(&RunContext{RootDir: root, Config: c})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(res.InputPath)
	if err != nil {
		t.Fatal(err)
	}
	jsonLen := binary.LittleEndian.Uint32(data[12:16])
	if binary.LittleEndian.Uint32(data[16:20]) != 0x4E4F534A {
		t.Fatal("first GLB chunk is not JSON")
	}
	jsonStart := uint32(20)
	binHeader := int(jsonStart + jsonLen)
	if binHeader+8 > len(data) {
		t.Fatal("GLB truncated before BIN chunk")
	}
	binLen := binary.LittleEndian.Uint32(data[binHeader : binHeader+4])
	if binary.LittleEndian.Uint32(data[binHeader+4:binHeader+8]) != 0x004E4942 {
		t.Fatal("second GLB chunk is not BIN")
	}
	if binLen != 44 {
		t.Fatalf("BIN chunk = %d bytes, want 44 (9 floats + 3 uint16 + 2 pad)", binLen)
	}
	var doc struct {
		Asset struct {
			Version string `json:"version"`
		} `json:"asset"`
		Nodes []struct {
			Name string `json:"name"`
			Mesh int    `json:"mesh"`
		} `json:"nodes"`
		Meshes []struct {
			Primitives []struct {
				Attributes map[string]int `json:"attributes"`
				Indices    int            `json:"indices"`
			} `json:"primitives"`
		} `json:"meshes"`
		Buffers []struct {
			ByteLength int `json:"byteLength"`
		} `json:"buffers"`
		Accessors []struct {
			ComponentType int    `json:"componentType"`
			Count         int    `json:"count"`
			Type          string `json:"type"`
		} `json:"accessors"`
	}
	if err := json.Unmarshal(data[jsonStart:jsonStart+jsonLen], &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Asset.Version != "2.0" || len(doc.Nodes) != 1 || doc.Nodes[0].Name != "hero-knight" {
		t.Fatalf("bad glTF header/node: %+v", doc)
	}
	if len(doc.Meshes) != 1 || len(doc.Meshes[0].Primitives) != 1 {
		t.Fatal("fixture must contain exactly one mesh primitive")
	}
	prim := doc.Meshes[0].Primitives[0]
	if prim.Attributes["POSITION"] != 0 || prim.Indices != 1 {
		t.Fatalf("primitive indices/attributes drift: %+v", prim)
	}
	if len(doc.Buffers) != 1 || doc.Buffers[0].ByteLength != int(binLen) {
		t.Fatalf("buffer byteLength %v != BIN chunk %d", doc.Buffers, binLen)
	}
	if len(doc.Accessors) != 2 || doc.Accessors[0].Count != 3 ||
		doc.Accessors[1].Count != 3 || doc.Accessors[0].Type != "VEC3" {
		t.Fatalf("accessors must describe 3 vertices + 3 indices: %+v", doc.Accessors)
	}
}

func TestKnownStrategiesMatchRegistry(t *testing.T) {
	if len(config.KnownStrategies) != len(IDs()) {
		t.Fatalf("config.KnownStrategies=%v but registry=%v", config.KnownStrategies, IDs())
	}
	for _, id := range config.KnownStrategies {
		if _, err := Get(id); err != nil {
			t.Fatalf("config declares unregistered supplier %q: %v", id, err)
		}
	}
	if config.DefaultSupplier != DefaultID() {
		t.Fatalf("config default %q != registry default %q", config.DefaultSupplier, DefaultID())
	}
}
