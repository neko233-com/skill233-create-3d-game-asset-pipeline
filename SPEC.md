# SPEC.md — 流水线规范（唯一契约）

本文件是本仓库的**规范性契约**。所有文档、Go 编排器、Blender 脚本必须与之一致；冲突时以本文件为准。
版本 / URL 的硬性约束另见 [AGENTS.md](AGENTS.md)。

## 1. 目标

把「一张参考图 / 一段参数」变成 Unity（URP）可直接使用的 3D 游戏资产，覆盖人形角色、道具与人形角色动画。

## 2. 硬约束

- **我们自己写的代码 = Go 1.27**：CLI、编排器、策略注册表、配置、校验、参考图处理，全部 Go 1.27，且**零第三方依赖**（仅标准库）。
- **Blender 内部代码 = Blender 5.2 内置 Python（`bpy`）**：通过 `blender --background --factory-startup --python <script> -- ...` 运行。Blender 原生自带什么 Python 就用什么，不假设系统 Python，也不在 Blender 里 `pip install`。（用户原话：「他原生代码用什么就用什么」）
- **Blender 版本 = 5.2**；**Tripo 入口 = https://studio.tripo3d.com/**。
- **参考插件 ACT**：https://github.com/mrven/Blender-Asset-Creation-Toolset （GPL-3.0）。只对齐其行为与参数，**不复制 / 不 vendoring 其源码**，避免 GPL 污染本仓库 MIT 许可。
- 系统 Python 3.14 只用于开发机上的辅助检查，不进入交付流水线。

## 3. 多策略（Strategy）

「建模」阶段是**可替换策略**，其余阶段共享。Tripo 是策略之一，**Blender 是默认策略**。

| id | 名称 | 阶段 1 实现 | 自动化程度 |
|----|------|-------------|-----------|
| `blender` | 参数化建模（**默认**） | `scripts/blender/stage_01_modeling.py` | 全自动 |
| `tripo` | 浏览器图生 3D | 人工在 https://studio.tripo3d.com/ 上传 → 下载 GLB 到 `raw/<id>.glb` | 半自动（Go 检测文件、给出指引、可断点续跑） |
| `import` | 导入已有模型 | 复制 `config.source` 到 `raw/<id>.<ext>` | 全自动 |

扩展方式：实现 `internal/strategy/strategy.go` 的 `Strategy` 接口并 `Register()`。典型可扩展项：`scenario` / `meshy` / `rodin` / 摄影测量。

```go
type Strategy interface {
    ID() string
    Description() string
    IsDefault() bool
    Acquire(ctx *RunContext) (*AcquireResult, error)
}
```

所有策略的产物统一落到 `raw/<id>.glb`（或 `.fbx` / `.obj`），随后进入**同一条**共享流水线。

## 4. 游戏美术全流程（六阶段，顺序固定）

| # | 阶段 id | 名称 | 实现 |
|---|---------|------|------|
| 1 | `modeling` | 建模 | 由所选 **Strategy** 提供 |
| 2 | `topology-uv` | 拓扑 / UV | `scripts/blender/stage_02_topology_uv.py` |
| 3 | `bake` | 烘焙 | `scripts/blender/stage_03_bake.py` |
| 4 | `pbr` | PBR 材质 | `scripts/blender/stage_04_pbr.py` |
| 5 | `unity-export` | Unity 导出 / 同步 | `scripts/blender/stage_05_unity_export.py` + Go 同步/校验 |
| 6 | `render` | 渲染合成 | `scripts/blender/stage_06_render.py` |

横切关注点：**人形角色动画**（Humanoid 绑定、rig 校验、clip 导出）贯穿阶段 4/5，见 `docs/50-humanoid-animation.md`。

## 5. 阶段间文件契约

Go runner 为每个阶段统一传递：

```
--asset <id> --config <asset.json 路径> --in <上一阶段产物> --out <本阶段产物> [--blend-out <中间 .blend>]
```

退出码：`0` 成功；非 `0` 失败，错误写 stderr。所有脚本必须容忍 `--factory-startup`（不依赖用户偏好）。

| 阶段 | `--in` | `--out` |
|------|--------|---------|
| 1 modeling | 参考图 / `config.source` | `work/<id>/01-modeling.blend` |
| 2 topology-uv | `work/<id>/01-modeling.blend` | `work/<id>/02-topology-uv.blend` |
| 3 bake | `work/<id>/02-topology-uv.blend` | `work/<id>/03-bake.blend` + `build/<id>/*_*.png` |
| 4 pbr | `work/<id>/03-bake.blend` | `work/<id>/04-pbr.blend` |
| 5 unity-export | `work/<id>/04-pbr.blend` | `build/<id>/<id>.fbx` + PBR 贴图 |
| 6 render | `build/<id>/<id>.fbx` | `build/<id>/preview.png` |

人形门禁：阶段 5 之后，若 `humanoid: true`，额外跑 `scripts/blender/rig_check.py`。

## 6. 目录约定

```
assets/<id>/asset.json          资产配置（JSON，Go encoding/json）
raw/<id>.*                      策略产物（原始，保留，便于回溯）
work/<id>/NN-<stage>.blend      阶段中间 .blend
build/<id>/<id>.fbx             最终 Unity 资产
build/<id>/<id>_basecolor.png   PBR 贴图（normal/roughness/metallic/ao 同理）
build/<id>/preview.png          阶段 6 渲染合成图
build/<id>/manifest.json        阶段状态 + DoD 结果
```

## 7. `asset.json` 字段

```json
{
  "id": "hero-knight",
  "strategy": "blender",
  "kind": "character",
  "humanoid": true,
  "referenceImage": "raw/hero-knight.png",
  "source": "",
  "tripoUrl": "https://studio.tripo3d.com/",
  "blenderVersion": "5.2",
  "targetEngine": "unity-urp",
  "axes": { "forward": "-Z", "up": "Y", "scale": 1.0 },
  "budget": { "maxTriangles": 30000, "maxTextureSize": 2048 },
  "textures": ["basecolor", "normal", "roughness", "metallic"],
  "animations": ["idle", "walk", "attack"],
  "modeling": { "primitive": "cube", "size": 1.0, "subdivisions": 2 },
  "stages": ["modeling", "topology-uv", "bake", "pbr", "unity-export", "render"]
}
```

## 8. CLI（`assetctl`，Go 1.27）

```
assetctl version
assetctl strategies                          # 列出策略，标注 DEFAULT
assetctl stages                              # 列出六阶段
assetctl init <asset-id> [--strategy blender] [--kind character] [--humanoid]
assetctl run <asset-id> [--strategy <id>] [--from <stage>] [--only <stage,...>] [--dry-run]
assetctl reference prepare <in.png> <outdir> [--size 1024] [--bg 808080]
assetctl validate <build-dir> [--expect-humanoid] [--expect-textures]
assetctl blender <script> [-- --args...]     # 透传 Blender 5.2
```

`assetctl run` 依次执行六阶段，写 `build/<id>/manifest.json`；任一阶段失败即停止并记录状态。
`--from` / `--only` 支持断点续跑（Tripo 半自动场景必需）。

## 9. 坐标与导出参数（对齐 Unity）

```
axis_forward = "-Z"
axis_up      = "Y"
global_scale = 1.0
apply_unit_scale = True
add_leaf_bones   = False
bake_anim        = True (humanoid)
path_mode        = "COPY"
```

Unity Import Settings 勾选 **Bake Axis Conversion**（对齐 ACT 的 `"Unity"` 导出档；ACT 的 `"Unity (Pre-Rotated)"` 不支持动画，故不采用）。

## 10. ACT 能力映射

| ACT 工具 | 本仓库落点 |
|----------|-----------|
| Batch Export FBX/GLTF for Unity/UE/Godot | 阶段 5 `stage_05_unity_export.py` + Go `assetctl run` |
| Export Axes / Scale for Unity = 1 | `asset.json.axes`，导出默认值 |
| Origin Align Tool | 阶段 1/2 清理步骤 |
| Renaming Tools / LOD to Name | 阶段 2 + `docs/70-naming-and-dod.md` |
| UV Tools（重命名 UV、Mark Seams from UV、Atlas Packing） | 阶段 2 `stage_02_topology_uv.py` |
| Material/Texture Tools（清理重复材质、Palette Bake、删未用材质） | 阶段 3/4 |
| Geometry Tools（Dissolve Checker Loops / Collapse Checker Edges） | 阶段 2 减面 |
| Cleanup Empties / Negative Scale / Duplicated Materials | 阶段 1/2 清理 |
| Low-Poly Art workflow | 阶段 2 面数预算 + 阶段 4 Palette 材质 |

安装方式：Blender Extensions（4.2+）或 https://github.com/mrven/Blender-Asset-Creation-Toolset/raw/master/release/act_2025_2_1.zip 。
ACT 2025.2 支持 Blender 3.0–5.0+，可与本机 **Blender 5.2** 并存；其 "Scale for models in Unity is 1" 与本仓库 `scale: 1.0` 一致。

## 11. 命名与 DoD

- 资产 ID：小写连字符 `hero-knight`；clip：`hero-knight-idle`。
- 贴图后缀：`_basecolor` / `_normal` / `_roughness` / `_metallic` / `_ao`。
- Humanoid 骨骼：`Hips` / `Spine` / `Chest` / `Neck` / `Head` / `Left|Right...`，左右用 `Left`/`Right` 前缀。
- DoD：`assetctl validate` 通过 + Unity 中无粉材质、无 T-pose、Avatar 映射无红项。

## 12. 许可证

本仓库 **MIT**。ACT 为 **GPL-3.0**，仅作为可选外部插件在文档中引用，不纳入本仓库代码。
