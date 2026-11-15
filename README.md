# 3D Game Asset Pipeline — 游戏美术管线向

[![Go 1.27](https://img.shields.io/badge/Go-1.27-00ADD8)](https://go.dev/)
[![Blender 5.2](https://img.shields.io/badge/Blender-5.2-orange)](https://www.blender.org/)
[![Tripo · DEFAULT](https://img.shields.io/badge/Tripo-DEFAULT%20Supplier-blue)](https://studio.tripo3d.com/)
[![License MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)

**多策略 · 六阶段 · Go 1.27 编排**的 3D 游戏资产生产流水线：把参考图 / 参数变成 **Unity（URP）** 可直接使用的 FBX + PBR 贴图，覆盖人形角色、道具与人形角色动画。

> - **默认生产供应商：`tripo`** —— 浏览器端 https://studio.tripo3d.com/ （Image-to-3D / Text-to-3D / auto-rig，半自动、人在回路；`IsDefault() == true`，`assetctl init` 默认 `--strategy tripo`）
> - 一等公民可选策略：**`blender`**（Blender 5.2 参数化 blockout）/ **`import`**（复制已有文件）；registry 可扩展 `meshy` / `rodin` / `scenario` / 摄影测量
> - 「**供应商（Supplier）== 建模策略（Strategy）**」；Go `Strategy` 类型保持原名
> - 默认供应商经 `handoff`（`manual | file | fixture`）交付机制实现脚本化 / 无人值守
> - 参考插件：[ACT — Blender Asset Creation Toolset](https://github.com/mrven/Blender-Asset-Creation-Toolset)（GPL-3.0，只对齐行为与参数、不 vendoring 源码）
> - 📐 规范性契约：[SPEC.md](SPEC.md) · 硬性版本 / 语言分工：[AGENTS.md](AGENTS.md) · 可复用 Skill：[SKILL.md](SKILL.md)

## 为什么存在

把「一张图 / 一段参数 → Unity 里能跑的角色」标准化为**游戏美术管线**顺序，适合 Unity 主程 + TA 管线：固定工具版本、固定坐标系、固定命名与校验、可断点续跑，避免每次手动踩坑。v2 起以**生产供应商 Tripo 为默认**，并用 handoff 机制把「人在回路」补齐为可脚本化、可 CI 全量回归的流水线。

```
                 ┌─────────────── 多策略 / 供应商（阶段 1 可替换）───────────────┐
                 │  tripo(DEFAULT,半自动+handoff)  /  blender(全自动)  /  import  │
                 └──────────────────────┬───────────────────────────────┘
                                        ▼
   参考图 ──> ① 建模 ──> ② 拓扑/UV ──> ③ 烘焙 ──> ④ PBR 材质 ──> ⑤ Unity 导出/同步 ──> ⑥ 渲染合成 ──> Unity (URP)
               ▲                                                     │                    │
               │                                          humanoid: rig_check.py 门禁     preview.png
         assets/<id>/asset.json                            FBX + PBR 贴图 + manifest.json
         （Go 1.27 assetctl 编排，Blender 5.2 内置 bpy 执行六阶段脚本）
```

## 语言分工（硬性，AGENTS.md）

| 运行位置 | 语言 |
|----------|------|
| CLI / 编排器 / 策略 / 配置 / 校验 / 参考图处理 | **Go 1.27**（零第三方依赖，仅标准库） |
| Blender 内部六阶段脚本 / rig 校验 | **Blender 5.2 原生内置 `bpy`** |
| 系统 Python 3.14.3 | 仅开发辅助检查（`py_compile` / 测试），不进交付流水线 |

## 快速开始

> `assetctl init <id>` 现在**默认 `strategy=tripo`**。下面给出两条路径：**(a) 生产 manual handoff**（人工下载 GLB）与 **(b) 无人值守 `--handoff fixture`**（CI / 无 Blender 全量回归）。

### (a) 生产路径 — 默认供应商 Tripo 人工 handoff

```bash
# 0. 构建 + 自测（Go 1.27）
go build ./... && go vet ./... && go test ./...

# 1. 查看供应商（DEFAULT=tripo）、六阶段、工具链报告
go run ./cmd/assetctl strategies
go run ./cmd/assetctl stages
go run ./cmd/assetctl doctor

# 2. 初始化资产配置（默认 strategy=tripo、handoff.mode=manual）
go run ./cmd/assetctl init hero-knight --humanoid --kind character

# 3. 人工 handoff：打开 https://studio.tripo3d.com/ ，Image-to-3D 上传参考图，
#    下载 GLB 到 raw/hero-knight.glb；缺文件时 run 打印指引并 exit 3
go run ./cmd/assetctl run hero-knight --from modeling

# 4. 交付前校验（含 manifest.json 六阶段全 ok 门禁）
go run ./cmd/assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

### (b) 无人值守路径 — `--handoff fixture`（CI 逃生舱）

```bash
# 生成确定性最小合法 GLB，使「默认供应商」也完全无人值守；无需人工、无需安装 Blender
go run ./cmd/assetctl init hero-knight --handoff fixture
go run ./cmd/assetctl run hero-knight
go run ./cmd/assetctl validate build/hero-knight --expect-humanoid --expect-textures

# 等价的一键验证：go test ./... 本身就是完整自动化验证
go test ./...
python scripts/verify.py
```

> `handoff` 三模式：`manual`（默认，人工下载 GLB，缺文件 exit `3` / `ErrManualStepRequired`）、`file`（复制 `handoff.source` 或 `ASSETCTL_TRIPO_SOURCE` 到 `raw/<id>.<ext>`，`assetctl init <id> --handoff file --source <path>`）、`fixture`（确定性 GLB）。环境变量 `ASSETCTL_TRIPO_HANDOFF`（覆盖 mode）、`ASSETCTL_TRIPO_SOURCE`（覆盖 source）。`Acquire` 优先级：已存在 `raw/<id>.{glb,gltf,fbx,obj}` > handoff mode > manual 指引。

### 切到可选策略

```bash
go run ./cmd/assetctl init hero-knight --strategy blender   # 参数化 blockout（CI 基线）
go run ./cmd/assetctl init hero-knight --supplier import --source D:/art/knight.fbx  # --supplier 是 --strategy 别名
go run ./cmd/assetctl run hero-knight --from topology-uv   # 断点续跑
```

参考图可用纯 Go 归一化：`go run ./cmd/assetctl reference prepare concept.png raw/ --size 1024`。

## CLI 一览（SPEC.md §8）

| 命令 | 作用 |
|------|------|
| `assetctl version` | 版本与工具链 |
| `assetctl strategies` | 列出建模策略 / 供应商，标注 DEFAULT（tripo） |
| `assetctl stages` | 列出六阶段 |
| `assetctl init <id>` | 创建 `assets/<id>/asset.json`（默认 tripo；`--handoff` / `--source` / `--supplier`） |
| `assetctl run <id>` | 执行六阶段，写 `build/<id>/manifest.json` |
| `assetctl reference prepare <in> <out>` | 参考图归一化（纯 Go） |
| `assetctl validate <build-dir>` | 交付前校验（manifest 六 ok + 契约 SHA-256 + 产物完整性） |
| `assetctl verify` | 一键生产级验证（Go 实现，9 步；`make verify` / `python scripts/verify.py` 转调） |
| `assetctl blender <script> [-- --args]` | 透传 Blender 5.2 |
| `assetctl doctor` | 工具链报告（Go / Blender 5.2 / `ASSETCTL_BLENDER` / 供应商+DEFAULT / Tripo URL / axes / 脚本 / 退出码） |

`run` 支持 `--strategy`（覆盖配置）、`--from` / `--only`（断点续跑，尤其 `--from topology-uv`）、`--dry-run`（只打印命令）；受 `ASSETCTL_TRIPO_HANDOFF` / `ASSETCTL_TRIPO_SOURCE` 环境变量覆盖。

## 测试 / CI

- **`go test ./...` 本身就是完整的自动化验证**：`internal/runner` 的 `Launcher` 抽象在生产用 `CommandLauncher`（真实 Blender 5.2），在 CI / 测试用 `ScriptLauncher` 运行 `testing/fake_blender.py`——一个纯标准库的 `bpy` + `mathutils` 模拟器。
- `fake_blender.py` 通过 `.blend` / `.fbx` 快照在多个阶段进程间**持久化场景状态**，用标准库 `zlib` / `struct` 写出**真实 PNG**，并给 FBX 写入真实二进制头；六阶段端到端跑通，**无需安装 Blender 5.2**。
- `go test ./...` 内含 **非 dry-run 六阶段 E2E**（含 humanoid rig 门禁）与 `--from` 断点续跑；`python -m unittest discover -s testing` 覆盖全部 **10 个 bpy 模块**、契约与 mock_bpy 自身（18 项）。
- 防静默失败：每个阶段执行后 runner 校验 `--out` 存在；每个阶段写出 `build/<id>/contracts/<stage>.json`（SHA-256 + 字节数 + FBX 七参数），`assetctl validate --expect-contracts` 重新读盘核对。
- `assetctl validate` 检查 `manifest.json` 六阶段全 `ok`、契约摘要、产物非零且 PNG/GLB magic 正确；`assetctl doctor` 用于生产 onboarding 自检。
- **一键验证 `assetctl verify`**（Go 1.27 实现，9 步）：布局 / 零第三方依赖 / build / vet / race+coverage / py_compile / unittest / doctor / CLI 冒烟。`python scripts/verify.py` 与 `make verify` 都只是转调它。
- `.github/workflows/ci.yml`：3-OS 矩阵（ubuntu / macos / windows）+ Go 1.27 + setup-python，上传 `coverage.out`。

## 六阶段与文档索引

| # | 阶段 | 实现 | 文档 |
|---|------|------|------|
| 1 | 建模（策略 / 供应商） | `internal/strategy/*` + `scripts/blender/stage_01_modeling.py` | [docs/10-strategies.md](docs/10-strategies.md) · [docs/20-stage-01-modeling.md](docs/20-stage-01-modeling.md) |
| 2 | 拓扑 / UV | `scripts/blender/stage_02_topology_uv.py` | [docs/30-stage-02-topology-uv.md](docs/30-stage-02-topology-uv.md) |
| 3 | 烘焙 | `scripts/blender/stage_03_bake.py` | [docs/40-stage-03-bake.md](docs/40-stage-03-bake.md) |
| 4 | PBR 材质 | `scripts/blender/stage_04_pbr.py` | [docs/41-stage-04-pbr.md](docs/41-stage-04-pbr.md) |
| 5 | Unity 导出 / 同步 | `scripts/blender/stage_05_unity_export.py` | [docs/42-stage-05-unity-export.md](docs/42-stage-05-unity-export.md) |
| 6 | 渲染合成 | `scripts/blender/stage_06_render.py` | [docs/43-stage-06-render-composite.md](docs/43-stage-06-render-composite.md) |

横切与参考文档：[总览](docs/00-overview.md) · [人形动画](docs/50-humanoid-animation.md) · [ACT 整合](docs/60-act-integration.md) · [语言分工](docs/61-language-split.md) · [命名与 DoD](docs/70-naming-and-dod.md) · [CLI 参考](docs/80-cli-reference.md)。

## 目录结构

```
SPEC.md / AGENTS.md / SKILL.md     契约 + 硬性事实 + 可复用 Skill
cmd/assetctl/                      Go 1.27 CLI 入口
internal/config/                   asset.json 配置与校验（含 handoff）
internal/strategy/                 多策略 / 供应商注册表（tripo DEFAULT / blender / import）
internal/runner/                   六阶段执行 + Launcher 抽象 + manifest.json + rig 门禁
internal/reference/                参考图归一化（纯 Go）
internal/validate/                 交付校验（manifest 六 ok + 契约 SHA-256 + 产物完整性）
internal/verify/                   一键验证（Go 1.27，9 步）
scripts/blender/                   Blender 5.2 内置 bpy 脚本（10 个模块）
scripts/verify.py                  一键验证兼容 shim（转调 assetctl verify）
testing/                           mock_bpy.py / fake_blender.py / harness.py / 阶段级 unittest
skills/                            28 个参考 skills（MIT）+ Go/bpy 适配
.github/workflows/ci.yml            Win/Linux/macOS 三平台 CI
docs/                              分阶段文档（00/10/20…80）
examples/                          asset.json / 提示词 / 清单示例
assets/<id>/asset.json             每个资产的配置
raw/ work/ build/                  策略产物 / 阶段中间 .blend / 最终交付
```

## 导出参数（对齐 Unity + ACT，SPEC.md §9）

`axis_forward=-Z` · `axis_up=Y` · `global_scale=1.0` · `apply_unit_scale=True` · `add_leaf_bones=False` · `bake_anim=True` · `path_mode=COPY`。
Unity Import Settings 勾选 **Bake Axis Conversion**（采用 ACT `"Unity"` 档；`"Unity (Pre-Rotated)"` 不支持动画故不采用）。

## DoD

`assetctl validate` 通过（含 `manifest.json` 六阶段全 `ok`、契约 SHA-256 一致）+ `assetctl doctor` 通过 + `assetctl verify` 全绿 + Unity 中无粉材质、无 T-pose、Avatar 映射无红项。

## License

本仓库 **MIT**（https://github.com/neko233-com/skill233-create-3d-game-asset-pipeline ）；ACT 为 **GPL-3.0**，仅作为可选外部插件在文档中引用，不纳入本仓库代码。
