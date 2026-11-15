package strategy

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

// TripoSupplierStrategy is the DEFAULT production supplier. Tripo runs in the
// browser at https://studio.tripo3d.com/ (Image-to-3D / Text-to-3D / auto-rig);
// it is semi-automatic and human-in-the-loop. Because it is the default, the
// Handoff mechanism (asset.json handoff.mode manual|file|fixture) lets the very
// same default path be scripted and tested unattended. See SPEC.md §3.
type TripoSupplierStrategy struct{}

func init() { Register(TripoSupplierStrategy{}) }

func (TripoSupplierStrategy) ID() string { return "tripo" }
func (TripoSupplierStrategy) Description() string {
	return "Tripo 图生 3D（默认生产供应商）· https://studio.tripo3d.com/ · 半自动 + handoff"
}
func (TripoSupplierStrategy) IsDefault() bool { return true }

// rawExports is the set of file extensions a Tripo (or upstream) drop may use.
var rawExports = []string{".glb", ".gltf", ".fbx", ".obj"}

func (TripoSupplierStrategy) Acquire(ctx *RunContext) (*AcquireResult, error) {
	c := ctx.Config
	rawDir := ctx.RawDir()

	// Priority 1: an export already present in raw/ always wins. This makes
	// `--from modeling` / resume idempotent and lets a human drop a GLB any time.
	if found := findRawExport(rawDir, c.ID); found != "" {
		return &AcquireResult{InputPath: found, Note: "found supplier export " + found}, nil
	}

	handoff := c.EffectiveHandoff()
	glb := filepath.Join(rawDir, c.ID+".glb")

	switch handoff.Mode {
	case "fixture":
		if ctx.DryRun {
			return &AcquireResult{InputPath: glb, Note: "dry-run: would synthesize fixture GLB " + glb}, nil
		}
		if err := os.MkdirAll(rawDir, 0o755); err != nil {
			return nil, err
		}
		if err := writeFixtureGLB(glb, c.ID); err != nil {
			return nil, fmt.Errorf("write fixture GLB: %w", err)
		}
		return &AcquireResult{InputPath: glb,
			Note: "fixture handoff: deterministic minimal GLB (CI/unattended) " + glb}, nil

	case "file":
		src := handoff.Source
		if src == "" {
			return nil, fmt.Errorf("handoff mode %q requires handoff.source (or ASSETCTL_TRIPO_SOURCE)", handoff.Mode)
		}
		if !filepath.IsAbs(src) {
			src = filepath.Join(ctx.RootDir, src)
		}
		if _, err := os.Stat(src); err != nil {
			return nil, fmt.Errorf("handoff file source not found: %s", src)
		}
		ext := strings.ToLower(filepath.Ext(src))
		if !oneOf(rawExports, ext) {
			return nil, fmt.Errorf("handoff source extension %q unsupported (want .glb/.gltf/.fbx/.obj)", ext)
		}
		dst := filepath.Join(rawDir, c.ID+ext)
		if ctx.DryRun {
			return &AcquireResult{InputPath: dst, Note: "dry-run: would handoff " + src + " -> " + dst}, nil
		}
		if err := os.MkdirAll(rawDir, 0o755); err != nil {
			return nil, err
		}
		if err := copyFile(src, dst); err != nil {
			return nil, err
		}
		return &AcquireResult{InputPath: dst, Note: "file handoff: copied " + src + " -> " + dst}, nil

	case "manual":
		fallthrough
	default:
		return manualGuidance(ctx, handoff.Mode)
	}
}

func manualGuidance(ctx *RunContext, mode string) (*AcquireResult, error) {
	c := ctx.Config
	rawDir := ctx.RawDir()
	ref := c.ReferenceImage
	if ref == "" {
		ref = filepath.Join("raw", c.ID+".png")
	}
	fmt.Fprintf(os.Stderr, "\n[tripo] 需要人工步骤 (supplier=tripo, handoff=%s)：\n", mode)
	fmt.Fprintf(os.Stderr, "  1. 参考图：%s （可用 assetctl reference prepare 生成）\n", ref)
	fmt.Fprintf(os.Stderr, "  2. 打开 %s 登录，选 Image-to-3D 上传参考图\n", config.TripoURL)
	fmt.Fprintf(os.Stderr, "  3. 生成并下载 GLB 到：%s\n", filepath.Join(rawDir, c.ID+".glb"))
	fmt.Fprintf(os.Stderr, "  4. 然后重跑：assetctl run %s --from modeling\n", c.ID)
	fmt.Fprintf(os.Stderr, "  （无人值守可用 --handoff fixture/file 或 ASSETCTL_TRIPO_HANDOFF）\n\n")
	if ctx.DryRun {
		return &AcquireResult{InputPath: "", Note: "dry-run: Tripo manual step pending"}, nil
	}
	return nil, fmt.Errorf("%w: tripo export missing at %s", ErrManualStepRequired,
		filepath.Join(rawDir, c.ID+".glb"))
}

func findRawExport(rawDir, id string) string {
	for _, ext := range rawExports {
		p := filepath.Join(rawDir, id+ext)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func oneOf(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// writeFixtureGLB writes a deterministic, STRUCTURALLY COMPLETE glTF 2.0
// binary (.glb) containing exactly one indexed triangle. Unlike an empty
// stand-in, real Blender 5.2 (and the stdlib bpy mock) import a real mesh from
// it, so every downstream stage has geometry to operate on. All bytes are
// fixed (no timestamps / random IDs) so CI snapshots are reproducible.
//
// BIN layout: 3 x VEC3 float32 (36 bytes) + 3 x uint16 indices (6 bytes) +
// 2 zero bytes of 4-byte alignment = 44 bytes.
func writeFixtureGLB(path, id string) error {
	var bin bytes.Buffer
	for _, v := range []float32{-0.5, 0, 0, 0.5, 0, 0, 0, 1, 0} {
		if err := binary.Write(&bin, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	for _, v := range []uint16{0, 1, 2} {
		if err := binary.Write(&bin, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	bin.Write([]byte{0, 0}) // pad BIN chunk to a multiple of 4
	binBytes := bin.Bytes()

	doc := map[string]interface{}{
		"asset":  map[string]interface{}{"version": "2.0", "generator": "assetctl fixture handoff (https://studio.tripo3d.com/ stand-in)"},
		"scene":  0,
		"scenes": []interface{}{map[string]interface{}{"nodes": []int{0}}},
		"nodes":  []interface{}{map[string]interface{}{"name": id, "mesh": 0}},
		"meshes": []interface{}{map[string]interface{}{
			"name": id + "_mesh",
			"primitives": []interface{}{map[string]interface{}{
				"attributes": map[string]int{"POSITION": 0},
				"indices":    1,
				"mode":       4, // TRIANGLES
			}},
		}},
		"buffers": []interface{}{map[string]interface{}{"byteLength": len(binBytes)}},
		"bufferViews": []interface{}{
			map[string]interface{}{"buffer": 0, "byteOffset": 0, "byteLength": 36, "target": 34962},
			map[string]interface{}{"buffer": 0, "byteOffset": 36, "byteLength": 6, "target": 34963},
		},
		"accessors": []interface{}{
			map[string]interface{}{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3",
				"min": []float32{-0.5, 0, 0}, "max": []float32{0.5, 1, 0}},
			map[string]interface{}{"bufferView": 1, "componentType": 5123, "count": 3, "type": "SCALAR",
				"min": []int{0}, "max": []int{2}},
		},
	}
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	// JSON chunk must be padded to 4 bytes with spaces; BIN chunk padded with zeros.
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	for len(binBytes)%4 != 0 {
		binBytes = append(binBytes, 0)
	}
	jsonLen := uint32(len(jsonBytes))
	binLen := uint32(len(binBytes))
	total := uint32(12 + 8 + jsonLen + 8 + binLen)

	var buf bytes.Buffer
	writeU32 := func(v uint32) { _ = binary.Write(&buf, binary.LittleEndian, v) }
	writeU32(0x46546C67) // "glTF" magic
	writeU32(2)          // version
	writeU32(total)
	writeU32(jsonLen)
	writeU32(0x4E4F534A) // "JSON"
	buf.Write(jsonBytes)
	writeU32(binLen)
	writeU32(0x004E4942) // "BIN\0"
	buf.Write(binBytes)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
