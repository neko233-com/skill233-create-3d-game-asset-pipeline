// Package skills is the Go 1.27 registry for the 28 Blender Agent Skills
// reference corpus in skills/ (MIT, from EL4CTEO/blender-skills).
//
// The SKILL.md documents are copied verbatim as knowledge; NO Python source
// from that repository is vendored. Execution is entirely ours:
//   - Go: this registry, `assetctl skills` CLI, argument validation, launch
//   - bpy: scripts/blender/skill_run.py, one operation per skill id
//   - tests: testing/test_skills.py + internal/skills tests under mock_bpy
//
// See SPEC.md §8A and skills/README.md.
package skills

import "sort"

// Skill is one executable, registered Blender workflow.
type Skill struct {
	ID        string   // directory name in skills/ (= catalog id)
	Category  string   // grouping for `assetctl skills list`
	Summary   string   // short Chinese description
	Operation string   // dispatch key inside scripts/blender/skill_run.py
	Stages    []string // related six-stage pipeline ids (SPEC.md §4)
}

// Catalog is the single compile-time source of truth for executable skills.
// skills/manifest.json is the documentation index; ValidateCatalog asserts
// both stay in sync (tested in catalog_test.go).
var Catalog = []Skill{
	{"blender-core", "foundation", "环境自检、5.x API 规则与任务路由", "core", nil},
	{"blender-scene-setup", "foundation", "单位、集合、命名与资产库", "scene_setup", []string{"modeling"}},
	{"blender-hardsurface-modeling", "modeling", "倒角、布尔与加权法线", "hardsurface_modeling", []string{"modeling", "topology-uv"}},
	{"blender-sculpting", "modeling", "有机雕刻、多级细分与重网格", "sculpting", []string{"modeling"}},
	{"blender-retopology-uv", "modeling", "重拓扑、UV 展开与纹素密度", "retopology_uv", []string{"topology-uv"}},
	{"blender-geometry-nodes", "modeling", "几何节点程序化建模", "geometry_nodes", []string{"modeling"}},
	{"blender-pbr-materials", "materials", "PBR 材质、色彩空间与贴图集", "pbr_materials", []string{"pbr"}},
	{"blender-texturing", "materials", "边缘磨损、贴花与 Trim Sheet", "texturing", []string{"pbr"}},
	{"blender-texture-baking", "materials", "法线 / AO / 曲率 / ORM 烘焙", "texture_baking", []string{"bake"}},
	{"blender-character-rigging", "rigging", "Rigify、蒙皮与游戏骨骼", "character_rigging", []string{"pbr", "unity-export"}},
	{"blender-facial-shapekeys", "rigging", "ARKit 52 表情与 viseme", "facial_shapekeys", []string{"pbr"}},
	{"blender-animation", "rigging", "Action / NLA、动作重定向与循环检查", "animation", []string{"unity-export"}},
	{"blender-clothing", "rigging", "服装跟随与穿模处理", "clothing", []string{"topology-uv"}},
	{"blender-hair-grooming", "rigging", "毛发、Groom 与游戏发卡", "hair_grooming", []string{"topology-uv"}},
	{"blender-game-optimization", "pipeline", "面数预算、LOD 与 Draw Call", "game_optimization", []string{"topology-uv"}},
	{"blender-export-pipelines", "pipeline", "glTF / FBX / USD 导出到 Unity 等引擎", "export_pipelines", []string{"unity-export"}},
	{"blender-pipeline-automation", "pipeline", "批处理与文件转换", "pipeline_automation", nil},
	{"blender-asset-qa", "qa", "Z-fighting、浮空件、法线与权重检查", "asset_qa", []string{"unity-export", "render"}},
	{"blender-workflow-recipes", "qa", "端到端资产 playbook", "workflow_recipes", nil},
	{"blender-rendering", "rendering", "Cycles / EEVEE 与批量渲染", "rendering", []string{"render"}},
	{"blender-lighting-lookdev", "rendering", "影棚灯光、HDRI 与转台", "lighting_lookdev", []string{"render"}},
	{"blender-stylized-npr", "rendering", "卡通 / 赛璐璐与描边", "stylized_npr", []string{"pbr"}},
	{"blender-environment-kits", "rendering", "地形、散布与模块化关卡套件", "environment_kits", []string{"modeling"}},
	{"blender-time-of-day", "rendering", "黄昏、夜景、雾与氛围光", "time_of_day", []string{"render"}},
	{"blender-weather-fx", "rendering", "雨雪、湿地与闪电", "weather_fx", []string{"render"}},
	{"blender-simulation", "rendering", "刚体、布料与流体", "simulation", nil},
	{"blender-video-editing", "rendering", "VSE 剪辑与摄像机跟踪", "video_editing", nil},
	{"blender-addon-development", "pipeline", "Blender 5.x 扩展与插件开发", "addon_development", nil},
}

var index map[string]Skill

func init() {
	index = make(map[string]Skill, len(Catalog))
	for _, s := range Catalog {
		if _, dup := index[s.ID]; dup {
			panic("skills: duplicate id " + s.ID)
		}
		index[s.ID] = s
	}
}

// ByID returns a registered skill.
func ByID(id string) (Skill, bool) {
	s, ok := index[id]
	return s, ok
}

// IDs returns sorted skill ids.
func IDs() []string {
	out := make([]string, 0, len(Catalog))
	for _, s := range Catalog {
		out = append(out, s.ID)
	}
	sort.Strings(out)
	return out
}

// Operations returns the bpy dispatch keys in catalog order.
func Operations() []string {
	out := make([]string, 0, len(Catalog))
	for _, s := range Catalog {
		out = append(out, s.Operation)
	}
	return out
}

// Count is the number of registered skills (28 for the reference corpus).
func Count() int { return len(Catalog) }
