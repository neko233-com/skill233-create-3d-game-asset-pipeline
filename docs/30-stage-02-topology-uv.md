# 30 · 阶段 ② 拓扑 / UV（topology-uv）

> 阶段 id：`topology-uv`。实现：`scripts/blender/stage_02_topology_uv.py`（Blender **5.2** 内置 Python / `bpy`）。
> 输入 `work/<id>/01-modeling.blend` → 输出 `work/<id>/02-topology-uv.blend`。契约见 [SPEC.md §5/§10](../SPEC.md)。

## 1. 阶段职责

把阶段 1 的原始网格整理成**游戏可用拓扑**：控制面数、清理脏数据、展开 UV、准备 LightmapUV、标记缝合边。TA 在本阶段最关心两件事：**面数是否达标**、**UV 是否可烘焙/可光照贴图**。

| 项 | 值 |
|----|----|
| `--in` | `work/<id>/01-modeling.blend` |
| `--out` | `work/<id>/02-topology-uv.blend` |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
| 运行 | `blender --background --factory-startup --python scripts/blender/stage_02_topology_uv.py -- --asset <id> --config <asset.json> --in ... --out ...` |

## 2. 面数预算表（三角面）

阶段 1 若是 AI / 摄影测量 / 外包高模，三角面往往远超预算；本阶段按资产类型压到目标区间。`asset.json.budget.maxTriangles` 是硬上限（默认 `30000`）。

| 资产类型 | 三角面预算 | 说明 |
|----------|-----------|------|
| **主要角色** | **15k–30k** | 近景、主角，可高一些；不超过 `budget.maxTriangles` |
| **普通 NPC** | **5k–15k** | 中景，批量出现，优先省 |
| **道具** | **500–5k** | 视屏幕占比，小道具压到 500 左右 |
| **远景物** | **<500** | 可考虑 impostor / LOD 最低级 |

> 预算以**三角面**计。Blender 里四边形按 2 个三角面折算。低多边形（Low-Poly）美术风格靠本阶段的面数预算 + 阶段 4 的 Palette 材质共同实现（SPEC.md §10 的「Low-Poly Art workflow」）。

## 3. Decimate 减面

用 Blender 5.2 的 **Decimate 修改器**（Collapse 模式，`ratio` 为保留比例）把高模压到预算内，并应用修改器（`bpy.ops.object.modifier_apply`），保证下游拿到的是真实几何。

要点：

- 先读 `asset.json.budget.maxTriangles` 与资产类型预算，反推目标 `ratio = 目标三角面 / 当前三角面`。
- 对顶点数很少的网格跳过减面（避免把道具削没）。
- 保护边界：有锐利硬边的道具优先保留轮廓，必要时改 Planar / 手动清理而非一味 Collapse。
- 减面前先应用缩放（`transform_apply(scale=True)`），否则比例失真。
- 减面后检查：无破面、无法线翻转、轮廓可辨识。

> 旧仓库 `scripts/blender/tripo_to_unity.py` 的 `--decimate 0.6`（保留 60% 面）是这一能力的早期单体版本；新架构把它固化在阶段 2 脚本里，由 `asset.json` 驱动，不再靠命令行临时参数。

## 4. 清理（Cleanup）

对齐 SPEC.md §10 的 ACT 行为（Cleanup Empties / Negative Scale / Duplicated Materials、Origin Align Tool、Renaming Tools）：

| 清理项 | 处理 |
|--------|------|
| **空物体 Empty** | 删除（FBX 导出会带出多余节点） |
| **负缩放 / 未应用变换** | `transform_apply(location=False, rotation=False, scale=True)`，避免 Unity 里法线翻转 / 缩放异常 |
| **重复材质** | 合并同名 / 同参数冗余材质（也见阶段 4 的 Material Tools） |
| **原点对齐** | 对齐 ACT 的 Origin Align Tool：把物体原点规整到几何中心/底部，Unity 里摆放与 Prefab 轴心才稳定 |
| **命名 / LOD 命名** | 对齐 ACT 的 Renaming Tools / LOD to Name：网格、LOD 层级用规范名（见 [70](70-naming-and-dod.md)） |
| **非流形 / 松散几何** | 删除游离顶点、零面积面，修复非流形边 |

## 5. UV：Smart UV Project + Mark Seams

主 UV（**UVChannel 0**，给 PBR 贴图用）用 Blender 5.2 的 **Smart UV Project**（`bpy.ops.uv.smart_project`）：

- 按角度阈值自动分岛，适合 AI 生成 / 摄影测量这种没有人工拓扑的网格。
- 设置合理的 **Island Margin**（岛间距，如 `0.02`），给阶段 3 烘焙留 padding，避免贴图串色。
- 对齐 ACT 的 **UV Tools**：
  - **重命名 UV**：两套 UV 用固定名字，如 `UVMap`（主 UV / 纹理）、`LightmapUV`（第二套）。
  - **Mark Seams from UV**：从已有 UV 岛边界反标缝合边（`bpy.ops.uv.seams_from_islands`），便于后续人工重展或检查。
  - **Atlas Packing**：多个小物件的贴图可打包成图集，减少 Unity 的材质/纹理数量与 draw call。
- 手动控制时：**Mark Seams**（标记缝合边）→ `UV Unwrap`。人形角色在关节内侧、腋下、胯下、道具隐蔽面切缝，避免正面出现拉伸。

检查项：UV 无重叠（烘焙用）、无超出 0–1 象限、岛间距足够、拉伸比可接受。

## 6. 第二套 LightmapUV

为 Unity 烘焙光照贴图（Lightmapping / 光照探针烘焙）准备**第二套 UV（UVChannel 1）**：

| 要求 | 说明 |
|------|------|
| **独立不重叠** | LightmapUV 必须每块面独占空间，不能像主 UV 那样复用/镜像，否则 Unity 烘焙会错乱 |
| **命名** | 固定 `LightmapUV`，对齐 ACT「重命名 UV」行为 |
| **生成方式** | 复制主 UV 后做 Lightmap Pack（`bpy.ops.uv.lightmap_pack`）或重新 Smart UV Project + 加大 padding |
| **Padding** | 比主 UV 更大，防止 lightmap 漏光/渗色 |
| **静态物件必做** | 道具 / 场景件要参与 Unity 光照烘焙，必须提供；纯动态角色可按需，但统一生成更省事 |

> Unity 在 Model Import Settings 里可按 **UV Channel** 读第二套作为 Lightmap UV；两套 UV 的命名/顺序稳定是关键。

## 7. 对齐 ACT 的 Geometry Tools

SPEC.md §10 把 ACT 的 **Geometry Tools（Dissolve Checker Loops / Collapse Checker Edges）** 映射到本阶段减面：

| ACT 能力 | 本阶段落点 |
|----------|-----------|
| Dissolve Checker Loops | 棋盘式溶解多余环线，配合 Decimate 清理规整但过密的布线 |
| Collapse Checker Edges | 棋盘式塌陷边，在低多边形化时保持面分布均匀、避免局部挤成一团 |
| Decimate（Collapse） | 阶段 2 减面主力，由 `budget.maxTriangles` 驱动 |

ACT 只作为**行为参考**，本仓库**不 vendoring 其源码**（GPL-3.0 vs MIT），详见 [60-act-integration.md](60-act-integration.md)。

## 8. 产物与检查清单

| 项 | 期望 |
|----|------|
| `work/<id>/02-topology-uv.blend` | 可被阶段 3 直接打开 |
| 三角面 | 落在资产类型预算区间，且 ≤ `budget.maxTriangles` |
| 主 UV `UVMap` | 无重叠、岛间距足够、0–1 内 |
| 第二套 `LightmapUV` | 独立不重叠、命名固定 |
| 变换 | 缩放已应用、无负缩放 |
| 场景 | 无 Empty、无重复材质、命名规范 |

- [ ] 面数符合预算表
- [ ] Decimate 已应用、无破面
- [ ] Empty / 负缩放 / 重复材质已清理
- [ ] 主 UV（Smart UV / 手工 + Seams）无重叠
- [ ] 第二套 LightmapUV 已生成且独立
- [ ] UV 命名对齐 ACT UV Tools 约定

下一步：[40-stage-03-bake.md](40-stage-03-bake.md)（烘焙）。
