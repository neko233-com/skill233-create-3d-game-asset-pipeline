---
name: create-3d-game-asset-pipeline
description: 游戏美术管线向的 3D 游戏资产生产流水线：Go 1.27 编排器(assetctl) + 多策略建模(默认 Blender 5.2 参数化，Tripo 图生 3D 为可选策略) → 六阶段(建模→拓扑/UV→烘焙→PBR 材质→Unity 导出/同步→渲染合成) → FBX+PBR 贴图+人形动画交付 Unity URP。当用户要做图生 3D、Blender 批处理、Unity 人形骨骼与动画、游戏美术管线、TA 管线、ACT 工具对齐时使用。
---

# Create 3D Game Asset Pipeline

把参考图 / 参数变成 Unity（URP）里可用的 3D 游戏资产的端到端、**多策略、六阶段**流水线。

> 📐 规范性契约：[SPEC.md](SPEC.md)。硬性版本 / 语言分工：[AGENTS.md](AGENTS.md)。

## 何时使用

- 做游戏美术管线向资产：建模 → 拓扑/UV → 烘焙 → PBR 材质 → Unity 导出/同步 → 渲染合成。
- 用**默认策略 `blender`**（Blender 5.2 `bpy` 参数化建模）全自动出资产，或切换到 `tripo`（浏览器图生 3D）/ `import`（导入已有模型）。
- 需要 Unity Humanoid 绑定、动画 clip、rig 校验门禁。
- 想对齐参考插件 [ACT](https://github.com/mrven/Blender-Asset-Creation-Toolset) 的导出轴/UV/材质/命名行为。

## 工具链（硬性）

| 运行位置 | 语言 / 版本 |
|----------|-------------|
| CLI、编排器、策略、配置、校验、参考图 | **Go 1.27**（仅标准库） |
| Blender 内部六阶段脚本 | **Blender 5.2 原生内置 `bpy`** |
| 系统 Python 3.14.3 | 仅开发辅助（`py_compile`），不进交付流水线 |
| Tripo | **https://studio.tripo3d.com/** （策略之一，非默认） |
| 目标引擎 | Unity（URP） |

## 多策略（SPEC.md §3）

| id | 名称 | 自动化 | 说明 |
|----|------|--------|------|
| `blender` | **默认** | 全自动 | Blender 5.2 `bpy` 参数化建模，无外部服务 |
| `tripo` | 浏览器图生 3D | 半自动 | 人工在 https://studio.tripo3d.com/ 上传 → 下载 GLB 到 `raw/<id>.glb`，Go 检测 + 断点续跑 |
| `import` | 导入已有模型 | 全自动 | 复制 `config.source` 到 `raw/` |

扩展：实现 `internal/strategy/strategy.go` 的 `Strategy` 接口并 `Register()`。

## 六阶段（顺序固定，SPEC.md §4）

```
1 modeling       ← 所选 Strategy 提供（或生成）
2 topology-uv    ← scripts/blender/stage_02_topology_uv.py
3 bake           ← scripts/blender/stage_03_bake.py
4 pbr            ← scripts/blender/stage_04_pbr.py
5 unity-export   ← scripts/blender/stage_05_unity_export.py  (+ humanoid: rig_check.py 门禁)
6 render         ← scripts/blender/stage_06_render.py
```

## 工作流（顺序执行）

1. **初始化配置** — `assetctl init <id> --humanoid` 生成 `assets/<id>/asset.json`（默认 `strategy=blender`）。
2. **（可选）准备参考图** — `assetctl reference prepare in.png raw/ --size 1024`（纯 Go 归一化：正方形、中性背景）。
3. **（tripo 策略时）人工生成** — 打开 https://studio.tripo3d.com/ ，Image-to-3D 上传参考图，下载 GLB 到 `raw/<id>.glb`；随后 `assetctl run <id> --from modeling` 续跑。
4. **跑流水线** — `assetctl run <id>`；Go 依次拉起 Blender 5.2 六阶段脚本，写 `build/<id>/manifest.json`。`--dry-run` 只打印命令。
5. **校验交付** — `assetctl validate build/<id> --expect-humanoid --expect-textures`；按 [docs/42-stage-05-unity-export.md](docs/42-stage-05-unity-export.md) 导入 Unity（勾 **Bake Axis Conversion**）。

## 命令速查（SPEC.md §8）

```bash
# 构建与自测
go build ./... && go vet ./... && go test ./...

# 查看策略 / 六阶段
go run ./cmd/assetctl strategies
go run ./cmd/assetctl stages

# 初始化一个资产（humanoid 角色）
go run ./cmd/assetctl init hero-knight --humanoid --kind character

# 切到 tripo 策略并只跑某阶段 / 断点续跑
go run ./cmd/assetctl init hero-knight --strategy tripo
go run ./cmd/assetctl run hero-knight --from topology-uv
go run ./cmd/assetctl run hero-knight --only pbr,unity-export

# 参考图归一化 / 交付校验
go run ./cmd/assetctl reference prepare concept.png raw/ --size 1024
go run ./cmd/assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

## 约定

- 资产 ID 小写连字符；坐标 `-Z forward / Y up / scale 1.0`；FBX + PBR 贴图交付。
- 导出参数对齐 ACT `"Unity"` 档（支持动画）；ACT 为 GPL-3.0，**只引用不 vendoring**，本仓库 MIT。
- 任何版本 / URL 变更先改 `AGENTS.md`；架构变更先改 `SPEC.md`。
- 文档索引见 [docs/00-overview.md](docs/00-overview.md)。
