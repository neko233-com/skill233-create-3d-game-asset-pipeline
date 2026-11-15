# AGENTS.md

本仓库面向「**游戏美术管线向**」的 3D 游戏资产生产：把参考图 / 参数变为 Unity（URP）可直接使用的资产，覆盖人形角色、道具与人形角色动画。
任何在本仓库工作的 Agent / 贡献者都必须遵守以下事实与约定。

> 📐 **规范性契约以 [SPEC.md](SPEC.md) 为准**（多策略、六阶段、目录、CLI、导出参数、许可证）。
> 本文件只负责锁定**版本 / URL / 语言分工 / 仓库约定**这些硬性事实；冲突时 SPEC.md 优先。

## 工具链版本（硬性约束）

- **Go：`1.27`** —— 我们自己写的编排代码（CLI、策略、配置、校验、参考图处理）全部 Go 1.27，**零第三方依赖**（仅标准库）。
- **Blender：`5.2`**（本机安装版本）。
  - Blender 内部脚本用 **Blender 5.2 原生内置 Python（`bpy`）**，原生自带什么 Python 就用什么；不假设系统 Python、不在 Blender 里 `pip install`。
  - 所有脚本以 `bpy` / Blender 5.2 为基准，必须能在 `--factory-startup` 下运行。
  - 无头调用：`blender --background --factory-startup --python <script> -- --asset <id> --config <path> --in <prev> --out <out> --blend-out <blend>`。
  - 禁止在文档/脚本中假设其它 Blender 大版本；升级版本必须先改本文件再改脚本。
- **Tripo：`https://studio.tripo3d.com/`**（浏览器端，图生 3D / 文生 3D / 自动绑定与动画）。
  - Tripo 是**默认生产供应商（DEFAULT Supplier）**（`IsDefault() == true`、`config.Default().Strategy == "tripo"`、`assetctl init` 默认 `--strategy tripo`），半自动、人在回路；「供应商（Supplier）== 建模策略（Strategy）」，Go `Strategy` 类型不改名（见 SPEC.md §3）。
  - 通过 `asset.json.handoff` 的 `manual | file | fixture` 三种交付模式（及 `ASSETCTL_TRIPO_HANDOFF` / `ASSETCTL_TRIPO_SOURCE` 环境变量）令默认供应商可脚本化、可无人值守测试；`fixture` 模式是 CI 逃生舱。
  - `blender`（参数化 blockout）与 `import`（复制已有文件）是一等公民的可选策略；registry 是 `meshy` / `rodin` / `scenario` / 摄影测量等未来供应商的扩展点。
  - 所有 Tripo 相关说明、链接、脚本注释统一使用该 URL，不得写成旧域名或镜像。
- **参考插件 ACT**：`https://github.com/mrven/Blender-Asset-Creation-Toolset`（GPL-3.0）。
  - 只对齐其行为与参数，**不复制 / 不 vendoring 其源码**，避免 GPL 污染本仓库 MIT 许可。
  - 采用其 `"Unity"` 导出档（支持动画）；不采用 `"Unity (Pre-Rotated)"`（不支持动画）。Unity 导入需勾 **Bake Axis Conversion**。
- **系统 Python** 仅用于开发机 / CI 的**测试 harness 与辅助检查**（`testing/mock_bpy.py` / `fake_blender.py` / 阶段级 unittest / `py_compile`），**不进入交付流水线**；`scripts/verify.py` 只是转调 Go `assetctl verify` 的兼容 shim，不含验证逻辑。交付物 = Go CLI/编排 + Blender 5.2 内置 bpy 脚本。
- **目标引擎**：Unity（URP 优先），资产以 FBX + PBR 贴图交付。

## 语言分工（硬性）

| 运行位置 | 语言 | 说明 |
|----------|------|------|
| CLI / 编排器 / 策略 / 配置 / 校验 / 参考图处理 | **Go 1.27** | 仅标准库，`go build ./...` 可过 |
| Blender 内部（六阶段脚本 / rig 校验） | **Blender 5.2 内置 `bpy`** | 经 `assetctl run` 由 Go 拉起，不直接依赖系统 Python |
| 系统 Python | 仅开发辅助 | 不产出交付物 |

用户原话（m00078）：「如果涉及自己写代码，用 go 1.27」「blender 用 python3 ?（他原生代码用什么就用什么）」。

## 仓库约定

- 语言：文档用中文；代码标识符、文件名、CLI 参数用英文。
- 命名：资产 ID 全小写连字符 `hero-knight-idle`；贴图后缀 `_basecolor/_normal/_roughness/_metallic/_ao`；动画 clip `hero-knight-idle`。
- 坐标系：Blender 导出 Unity 用 `axis_forward=-Z`、`axis_up=Y`、缩放 `1.0`（见 `asset.json.axes` 与 SPEC.md §9）。
- 人形角色：骨骼命名对齐 Unity Humanoid（`Hips/Spine/Chest/Neck/Head/Left…/Right…`），`humanoid: true` 时导出前必过 `rig_check.py` 门禁。
- 六阶段顺序固定：`modeling → topology-uv → bake → pbr → unity-export → render`（SPEC.md §4）。
- 提交：Conventional Commits（`feat:`、`fix:`、`docs:`、`chore:`）。

## 目录职责

| 路径 | 作用 |
|------|------|
| `SPEC.md` | 规范性契约（多策略 / 六阶段 / 目录 / CLI / 导出参数 / ACT 映射 / 许可证） |
| `AGENTS.md` | 版本、URL、语言分工的硬性事实 |
| `SKILL.md` | 可复用 Skill 入口，供其它项目引用本流水线 |
| `cmd/assetctl/` | Go 1.27 CLI 入口（`assetctl`） |
| `internal/{config,strategy,runner,reference,validate}/` | Go 编排器：配置、策略注册、六阶段执行、参考图、校验 |
| `scripts/blender/` | Blender 5.2 内运行的 `bpy` 脚本（10 个：六阶段 + rig_check + humanoid_rig + tripo_to_unity + common） |
| `internal/verify/` + `scripts/verify.py` | Go 实现的一键验证（9 步）+ 兼容 shim；验证逻辑不在系统 Python 中 |
| `testing/` | `mock_bpy.py` / `fake_blender.py` / `harness.py` / 阶段级 unittest（仅测试，不进交付） |
| `skills/` | 参考 EL4CTEO/blender-skills（MIT）的 28 个 SKILL.md 与 Go+bpy 适配（见 skills/README.md） |
| `.github/workflows/ci.yml` | 3-OS 矩阵（Win/Linux/macOS）一键验证 |
| `docs/` | 分阶段流水线文档（00/10/20…80 编号） |
| `examples/` | `asset.json`、参考图、Tripo 提示词示例 |
| `assets/<id>/asset.json` | 每个资产的配置 |
| `raw/`、`work/`、`build/` | 策略产物、阶段中间 .blend、最终交付 |

## 成功标准（DoD）

一个资产「完成」意味着：选定策略产出模型 → 六阶段流水线跑完 → `assetctl validate` 通过 →
Unity 导入后贴图/骨骼/动画均可用（无粉材质、无 T-pose、Avatar 映射无红项），且 `build/<id>/manifest.json` 全 `ok`。

## 常用命令

```bash
go build ./... && go vet ./... && go test ./...        # go test ./... 即完整自动化验证，无需装 Blender
go run ./cmd/assetctl doctor                            # 工具链报告（v2 新增）
go run ./cmd/assetctl verify                           # 一键生产级验证（9 步，Go 实现，v2.1）
go run ./cmd/assetctl strategies                       # 列出供应商，tripo 标注 DEFAULT
go run ./cmd/assetctl init hero-knight --humanoid       # 默认 strategy=tripo
go run ./cmd/assetctl init hero-knight --handoff fixture  # CI 无人值守（默认供应商也能全自动）
go run ./cmd/assetctl run hero-knight --dry-run          # 只打印命令，不执行
go run ./cmd/assetctl run hero-knight                    # 真正执行（生产需 Blender 5.2）
python scripts/verify.py                                 # 一键验证：build/vet/test/py_compile/unittest
```
