package strategy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

// TripoSemiAutomaticStrategy is one of several strategies (NOT the default).
// Tripo runs in the browser at https://studio.tripo3d.com/ ; a human uploads a
// reference image and downloads a GLB into raw/. The runner detects it and
// supports resume via `assetctl run <id> --from topology-uv`.
type TripoSemiAutomaticStrategy struct{}

func init() { Register(TripoSemiAutomaticStrategy{}) }

func (TripoSemiAutomaticStrategy) ID() string          { return "tripo" }
func (TripoSemiAutomaticStrategy) Description() string { return "浏览器图生 3D · studio.tripo3d.com · 半自动" }
func (TripoSemiAutomaticStrategy) IsDefault() bool     { return false }

func (TripoSemiAutomaticStrategy) Acquire(ctx *RunContext) (*AcquireResult, error) {
	c := ctx.Config
	rawDir := ctx.RawDir()

	var found string
	for _, ext := range []string{".glb", ".gltf", ".fbx", ".obj"} {
		p := filepath.Join(rawDir, c.ID+ext)
		if _, err := os.Stat(p); err == nil {
			found = p
			break
		}
	}
	if found != "" {
		return &AcquireResult{
			InputPath: found,
			Note:      "found Tripo export " + found,
		}, nil
	}

	// Not downloaded yet: give actionable guidance, do not fail cryptically.
	ref := c.ReferenceImage
	if ref == "" {
		ref = filepath.Join("raw", c.ID+".png")
	}
	fmt.Fprintf(os.Stderr, "\n[tripo] 需要人工步骤 (strategy=tripo)：\n")
	fmt.Fprintf(os.Stderr, "  1. 参考图：%s （可用 assetctl reference prepare 生成）\n", ref)
	fmt.Fprintf(os.Stderr, "  2. 打开 %s 登录，选 Image-to-3D 上传参考图\n", config.TripoURL)
	fmt.Fprintf(os.Stderr, "  3. 生成并下载 GLB 到：%s\n", filepath.Join(rawDir, c.ID+".glb"))
	fmt.Fprintf(os.Stderr, "  4. 然后重跑：assetctl run %s --from modeling\n\n", c.ID)
	if ctx.DryRun {
		return &AcquireResult{InputPath: "", Note: "dry-run: Tripo manual step pending"}, nil
	}
	return nil, fmt.Errorf("%w: tripo export missing at %s", ErrManualStepRequired,
		filepath.Join(rawDir, c.ID+".glb"))
}
