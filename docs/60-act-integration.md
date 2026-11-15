# 60 · ACT 参考插件整合

> 本文对齐 [SPEC.md §10](../SPEC.md) 的 ACT 能力映射表。ACT 是**外部可选参考插件**，本仓库 MIT，只对齐行为、不纳入源码。

## 1. 插件信息

| 项 | 值 |
|----|----|
| 名称 | **Blender Asset Creation Toolset（ACT）** |
| 仓库 | **https://github.com/mrven/Blender-Asset-Creation-Toolset** |
| 许可证 | **GPL-3.0** |
| 默认分支 | **`master`** |
| 版本 | **ACT 2025.2** |
| 支持 Blender | **3.0–5.0+**（可与本机 **Blender 5.2** 并存） |
| 关键点 | 其 "Scale for models in Unity is 1" 与本仓库 `axes.scale = 1.0`（[SPEC.md §7/§9](../SPEC.md)）一致 |

> AGENTS.md / SPEC.md 锁定本机 Blender 为 **5.2**；ACT 声称支持 3.0–5.0+，作为本机可选插件并存，但**本仓库所有脚本仍以 Blender 5.2 / `bpy` 为基准**，不依赖 ACT 运行。

## 2. 安装方式（二选一）

ACT 不进入交付流水线，仅供美术在 Blender GUI 里手动对齐行为时使用。

### 2.1 Blender Extensions（4.2+）

Blender 4.2 起内置 Extensions 机制：在 Blender 5.2 的 **Edit → Preferences → Get Extensions** 中搜索 / 安装 ACT，或用 `blender --command extension` 系列命令管理。适合已联网、希望自动更新的环境。

### 2.2 zip 手动安装

直接下载官方 release zip（SPEC.md §10 给出的地址）：

```
https://github.com/mrven/Blender-Asset-Creation-Toolset/raw/master/release/act_2025_2_1.zip
```

在 Blender 5.2 里 **Edit → Preferences → Add-ons → Install from Disk…** 选中该 zip，启用插件即可。

> 注意：仓库默认分支是 `master`，上面 URL 走 `raw/master/`。不要把分支写成 `main` 或改成镜像域名。

## 3. 逐项能力映射到本仓库阶段（严格照 SPEC.md §10 表格）

| ACT 工具 | 本仓库落点 | 对应文档 |
|----------|-----------|----------|
| Batch Export FBX/GLTF for Unity/UE/Godot | 阶段 5 `stage_05_unity_export.py` + Go `assetctl run` | [42](42-stage-05-unity-export.md) |
| Export Axes / Scale for Unity = 1 | `asset.json.axes`，导出默认值 | [42](42-stage-05-unity-export.md) |
| Origin Align Tool | 阶段 1/2 清理步骤 | [20](20-stage-01-modeling.md) / [30](30-stage-02-topology-uv.md) |
| Renaming Tools / LOD to Name | 阶段 2 + `docs/70-naming-and-dod.md` | [30](30-stage-02-topology-uv.md) / [70](70-naming-and-dod.md) |
| UV Tools（重命名 UV、Mark Seams from UV、Atlas Packing） | 阶段 2 `stage_02_topology_uv.py` | [30](30-stage-02-topology-uv.md) |
| Material/Texture Tools（清理重复材质、Palette Bake、删未用材质） | 阶段 3/4 | [40](40-stage-03-bake.md) / [41](41-stage-04-pbr.md) |
| Geometry Tools（Dissolve Checker Loops / Collapse Checker Edges） | 阶段 2 减面 | [30](30-stage-02-topology-uv.md) |
| Cleanup Empties / Negative Scale / Duplicated Materials | 阶段 1/2 清理 | [20](20-stage-01-modeling.md) / [30](30-stage-02-topology-uv.md) |
| Low-Poly Art workflow | 阶段 2 面数预算 + 阶段 4 Palette 材质 | [30](30-stage-02-topology-uv.md) / [41](41-stage-04-pbr.md) |

映射要点：
- ACT 的「批量导出 / 轴与缩放 / 原点 / 命名 / UV / 材质 / 几何清理 / 低多边形」几乎覆盖本仓库阶段 1–5 的手工等价操作。
- 本仓库把这些**行为固化进 `bpy` 脚本 + Go 编排**，做到可重复、可进 CI、不依赖人工点插件按钮。
- ACT 的 **Palette Bake** 同时跨阶段 3（烘焙）与阶段 4（Palette 材质），SPEC.md §10 把它记在「阶段 3/4」。

## 4. 「只对齐行为、不 vendoring 源码」（避免 GPL 污染 MIT）

SPEC.md §2 与 §12 明确：

| 事实 | 约束 |
|------|------|
| 本仓库许可 | **MIT** |
| ACT 许可 | **GPL-3.0** |
| 做法 | **只对齐其行为与参数，不复制 / 不 vendoring 其源码**，避免 GPL 污染本仓库 MIT 许可 |

具体含义：
- 文档里可以**引用** ACT 的工具名、参数语义、导出档名称、仓库 URL 作为「行为参照」。
- 不得把 ACT 的 `.py` 源码、函数实现、整段逻辑拷贝进 `scripts/blender/` 或 `internal/`。
- 我们自己的 `stage_*.py` 与 Go 代码用 `bpy` 原生 API 与标准库**独立实现等价行为**（如用 `bpy.ops.export_scene.fbx` 对齐 ACT 的 Unity 导出参数、用 Decimate 对齐 Geometry Tools）。
- ACT 仅作为**可选外部插件**在文档中引用，不纳入本仓库代码，也不进入交付流水线（CI 不装、脚本不 import）。
- 若美术在本机 Blender 5.2 装了 ACT 辅助操作，那是个人工具选择，不改变本仓库的许可与产物契约。

## 5. ACT "Unity" 与 "Unity (Pre-Rotated)" 两档区别与我们的选择

SPEC.md §9 明确：Unity Import Settings 勾 **Bake Axis Conversion**（对齐 ACT 的 **`"Unity"`** 导出档）；ACT 的 **`"Unity (Pre-Rotated)"`** 不支持动画，故不采用。

| 档 | 做法 | 动画支持 | 本仓库 |
|----|------|----------|--------|
| **`"Unity"`** | 导出按 `-Z forward / Y up / global_scale 1.0`，轴转换交给 Unity 导入时的 **Bake Axis Conversion** 处理 | **支持动画 / Humanoid** | ✅ **采用** |
| **`"Unity (Pre-Rotated)"`** | 在 Blender 侧预先把网格/骨架旋转好再导出，省掉 Unity 侧转换 | **不支持动画** | ❌ 不采用 |

为什么选 `"Unity"` + 勾 **Bake Axis Conversion**：

1. **人形角色必须带动画**：clip、Humanoid Avatar、Root Motion 都依赖 FBX 里正确的骨骼变换曲线。`"Unity (Pre-Rotated)"` 预旋转会破坏动画与 Avatar 映射，导致 Unity 里 T-pose / 骨骼错位（[50](50-humanoid-animation.md)）。
2. **与 SPEC.md §9 的 7 项导出参数一致**：`axis_forward = "-Z"`、`axis_up = "Y"`、`global_scale = 1.0`、`apply_unit_scale = True`、`add_leaf_bones = False`、`bake_anim = True (humanoid)`、`path_mode = "COPY"`——这套正是 ACT "Unity" 档的语义。
3. **Bake Axis Conversion 把轴转换「烤」进网格**：Unity 导入时统一处理，模型朝向（面朝 +Z）、缩放 1.0、动画曲线三者不打架。

> 简记：**导出用 ACT 语义的 "Unity" 档，Unity 里勾 Bake Axis Conversion；永远不要用 "Unity (Pre-Rotated)"，因为它不支持动画。**

## 6. 检查清单

- [ ] 文档中 ACT 仓库写 https://github.com/mrven/Blender-Asset-Creation-Toolset ，分支 `master`，版本 ACT 2025.2，GPL-3.0
- [ ] 安装方式写 Blender Extensions（4.2+）或 `release/act_2025_2_1.zip`
- [ ] 能力映射逐项对齐 SPEC.md §10 表格，不增删行
- [ ] 明确「不 vendoring 源码、避免 GPL 污染 MIT」
- [ ] 明确选 "Unity" 档 + Bake Axis Conversion，弃用 "Unity (Pre-Rotated)"（不支持动画）
- [ ] 所有脚本仍以 Blender **5.2** 为基准，不依赖 ACT 运行
