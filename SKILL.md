---
name: create-3d-game-asset-pipeline
description: 游戏美术管线向的 3D 游戏资产生产流水线：Go 1.27 编排器(assetctl) + 多策略建模(默认供应商 Tripo 图生 3D，半自动、人在回路，经 handoff 交付机制 manual|file|fixture 可脚本化/可无人值守；blender 参数化 blockout 与 import 为一等公民可选策略，registry 可扩展 meshy/rodin/scenario) → 六阶段(建模→拓扑/UV→烘焙→PBR 材质→Unity 导出/同步→渲染合成) → FBX+PBR 贴图+人形动画交付 Unity URP。当用户要做图生 3D、Blender 批处理、Unity 人形骨骼与动画、游戏美术管线、TA 管线、ACT 工具对齐、CI 无 Blender 自测时使用。
---

# Create 3D Game Asset Pipeline

把参考图 / 参数变成 Unity（URP）里可用的 3D 游戏资产的端到端、**多策略（多供应商）、六阶段**流水线。

> 📐 规范性契约：[SPEC.md](SPEC.md)。硬性版本 / 语言分工：[AGENTS.md](AGENTS.md)。

## 何时使用

- 做游戏美术管线向资产：建模 → 拓扑/UV → 烘焙 → PBR 材质 → Unity 导出/同步 → 渲染合成。
- 用**默认供应商 `tripo`**（https://studio.tripo3d.com/ 浏览器图生 3D，半自动、人在回路）出生产资产；通过 `asset.json.handoff`（`manual | file | fixture`）把它脚本化、无人值守化。
- 切到 `blender`（Blender 5.2 `bpy` 参数化 blockout）/ `import`（导入已有模型）等一等公民可选策略，或在 registry 扩展新供应商。
- 需要 Unity Humanoid 绑定、动画 clip、rig 校验门禁；或想对齐参考插件 [ACT](https://github.com/mrven/Blender-Asset-Creation-Toolset) 的导出轴/UV/材质/命名行为。
- 要在 CI / 无 Blender 5.2 环境里端到端验证流水线（`go test ./...` 即完整验证）。

## 工具链（硬性）

| 运行位置 | 语言 / 版本 |
|----------|-------------|
| CLI、编排器、策略、配置、校验、参考图 | **Go 1.27**（仅标准库，零第三方依赖） |
| Blender 内部六阶段脚本 | **Blender 5.2 原生内置 `bpy`** |
| 系统 Python 3.14.3 | 仅开发辅助（`py_compile` / 测试），不进交付流水线 |
| Tripo | **https://studio.tripo3d.com/** （**默认供应商 / DEFAULT**） |
| 宿主平台 | **Windows x86_64**（`GOOS=windows GOARCH=amd64`；无 CI 服务，禁止 GitHub Actions） |
| 目标引擎 | Unity（URP） |

> 术语：**供应商（Supplier）== 建模策略（Strategy）**，指「阶段 1 由谁产出原始网格」；Go `Strategy` 类型与 `asset.json.strategy` 字段保持原名，不重命名。

## 多策略 / 供应商（SPEC.md §3）

| id | 名称 | 自动化 | 默认 | 说明 |
|----|------|--------|------|------|
| `tripo` | 浏览器图生 3D | 半自动（人在回路）+ handoff 可脚本化 | **DEFAULT** | 在 https://studio.tripo3d.com/ 上传 → 下载 GLB 到 `raw/<id>.glb`；`IsDefault() == true` |
| `blender` | 参数化 blockout | 全自动 | 否 | Blender 5.2 `bpy` 参数化建模，无外部服务，CI 基线 |
| `import` | 导入已有模型 | 全自动 | 否 | 复制 `config.source` 到 `raw/` |

扩展：实现 `internal/strategy/strategy.go` 的 `Strategy` 接口并 `Register()`；新供应商一律 `IsDefault() == false`。

## Handoff 交付机制（v2，默认供应商可脚本化）

`asset.json` 新增 `"handoff": { "mode": "manual", "source": "" }`：

| mode | 行为 |
|------|------|
| `manual`（默认） | 人工在 https://studio.tripo3d.com/ 上传、下载 GLB 到 `raw/<id>.glb`，再 `assetctl run <id> --from modeling`；缺文件打印指引并 exit `3`（`ErrManualStepRequired`） |
| `file` | 复制 `handoff.source`（或 `ASSETCTL_TRIPO_SOURCE`）到 `raw/<id>.<ext>`；`assetctl init <id> --handoff file --source <path>` |
| `fixture` | 生成确定性最小合法 GLB，使**默认供应商完全无人值守**（CI 逃生舱）；`ASSETCTL_TRIPO_HANDOFF=fixture` 亦可强制 |

环境变量：`ASSETCTL_TRIPO_HANDOFF`（覆盖 mode）、`ASSETCTL_TRIPO_SOURCE`（覆盖 source）。`Acquire` 优先级：已存在 `raw/<id>.{glb,gltf,fbx,obj}` 优先 → handoff mode → manual 指引。`--from` / `--only`（尤其 `--from topology-uv`）断点续跑保持不变。

## 六阶段（顺序固定，SPEC.md §4）

```
1 modeling       ← 所选 Strategy / Supplier 提供（或生成）
2 topology-uv    ← scripts/blender/stage_02_topology_uv.py
3 bake           ← scripts/blender/stage_03_bake.py
4 pbr            ← scripts/blender/stage_04_pbr.py
5 unity-export   ← scripts/blender/stage_05_unity_export.py  (+ humanoid: rig_check.py 门禁)
6 render         ← scripts/blender/stage_06_render.py
```

## 工作流（顺序执行）

1. **初始化配置** — `assetctl init <id> --humanoid` 生成 `assets/<id>/asset.json`（默认 `strategy=tripo`、`handoff.mode=manual`）。
2. **（可选）准备参考图** — `assetctl reference prepare in.png raw/ --size 1024`（纯 Go 归一化：正方形、中性背景）。
3. **生产：人工 handoff** — 打开 https://studio.tripo3d.com/ ，Image-to-3D 上传参考图，下载 GLB 到 `raw/<id>.glb`；随后 `assetctl run <id> --from modeling` 续跑。
4. **无人值守 CI：`--handoff fixture`** — `assetctl init <id> --handoff fixture` 生成确定性 GLB，无需人工、无需 Blender 安装即可跑通；`go test ./...` 本身就是完整自动化验证。
5. **跑流水线** — `assetctl run <id>`；Go 依次拉起六阶段，写 `build/<id>/manifest.json`。`--dry-run` 只打印命令。
6. **校验交付** — `assetctl doctor`（onboarding 自检）+ `assetctl validate build/<id> --expect-humanoid --expect-textures`（含 `manifest.json` 六阶段全 `ok` 门禁）；按 [docs/42-stage-05-unity-export.md](docs/42-stage-05-unity-export.md) 导入 Unity（勾 **Bake Axis Conversion**）。

## 命令速查（SPEC.md §8）

```bash
# 构建与自测（go test ./... 即完整自动化验证，无需装 Blender）
go build ./... && go vet ./... && go test ./...   # 含非 dry-run 六阶段 E2E
python -m unittest discover -s testing           # 11 个 bpy 模块 + 28 skills + 契约 + mock_bpy
python scripts/verify.py                         # 兼容 shim，实际转调 Go 的 assetctl verify
go run ./cmd/assetctl verify                     # 一键生产级验证（推荐，13 步）

# 查看供应商 / 六阶段 / 工具链
go run ./cmd/assetctl strategies                 # tripo 标注 DEFAULT
go run ./cmd/assetctl stages
go run ./cmd/assetctl doctor                     # 工具链报告

# 28 个 Blender skills（Go 编排 + bpy 执行，SPEC.md §8A）
go run ./cmd/assetctl skills list                # id / category / summary
go run ./cmd/assetctl skills doctor              # 注册表一致性
go run ./cmd/assetctl skills run blender-retopology-uv --asset hero-knight

# 生产：默认供应商 manual handoff（人工下载 GLB）
go run ./cmd/assetctl init hero-knight --humanoid --kind character
# ... 人工在 https://studio.tripo3d.com/ 下载 GLB 到 raw/hero-knight.glb ...
go run ./cmd/assetctl run hero-knight --from modeling

# CI 无人值守：默认供应商也能全自动
go run ./cmd/assetctl init hero-knight --handoff fixture
go run ./cmd/assetctl run hero-knight

# 断点续跑 / 切换策略
go run ./cmd/assetctl run hero-knight --from topology-uv
go run ./cmd/assetctl run hero-knight --only pbr,unity-export
go run ./cmd/assetctl init hero-knight --strategy blender   # 可选：切到参数化 blockout

# 参考图归一化 / 交付校验
go run ./cmd/assetctl reference prepare concept.png raw/ --size 1024
go run ./cmd/assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

## 约定

- 资产 ID 小写连字符；坐标 `-Z forward / Y up / scale 1.0`；FBX + PBR 贴图交付。
- 导出参数对齐 ACT `"Unity"` 档（支持动画）；ACT 为 GPL-3.0，**只引用不 vendoring**，本仓库 MIT。
- 任何版本 / URL 变更先改 `AGENTS.md`；架构变更先改 `SPEC.md`。
- 六阶段各写出 `build/<id>/contracts/<stage>.json`（SHA-256 + 字节数 + FBX 七参数），`assetctl validate` 默认重算校验；11 个 bpy 模块（含 `skill_run.py`）均可在 mock_bpy 下独立执行（SPEC.md §5A）。
- 一键验证用 `assetctl verify`（Go 实现）；系统 Python 仅用于测试 harness，不进入交付流水线。
- 文档索引见 [docs/00-overview.md](docs/00-overview.md)；28 个 Skills 见 [docs/90-skills.md](docs/90-skills.md)。
- Skills 子系统（SPEC.md §8A）：`skills/<id>/SKILL.md`（照抄 EL4CTEO/blender-skills，MIT）+ Go 注册表 `internal/skills` + 单一 bpy 入口 `scripts/blender/skill_run.py`，每个 skill 产出 `build/<id>/contracts/skill-<id>.json`。
