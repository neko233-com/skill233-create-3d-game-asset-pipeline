package strategy

import (
	"fmt"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

// BlenderStrategy is an ALTERNATIVE supplier: procedural modeling inside
// Blender 5.2 using its native Python (bpy). No external service, fully
// automatic — ideal for CI blockouts and pipeline self-tests. It is NOT the
// production default (Tripo is; see SPEC.md §3).
type BlenderStrategy struct{}

func init() { Register(BlenderStrategy{}) }

func (BlenderStrategy) ID() string { return "blender" }
func (BlenderStrategy) Description() string {
	return "参数化建模（可选策略 / CI baseline）· Blender 5.2 bpy · 全自动"
}
func (BlenderStrategy) IsDefault() bool { return false }

func (BlenderStrategy) Acquire(ctx *RunContext) (*AcquireResult, error) {
	c := ctx.Config
	if c.Modeling.Primitive == "" {
		return nil, fmt.Errorf("blender strategy requires config.modeling.primitive")
	}
	// Empty InputPath tells stage_01_modeling.py to generate geometry itself.
	return &AcquireResult{
		InputPath: "",
		Note: fmt.Sprintf("procedural %s (size=%.2f, subdivisions=%d) via Blender %s bpy",
			c.Modeling.Primitive, c.Modeling.Size, c.Modeling.Subdivisions, config.BlenderVersion),
	}, nil
}
