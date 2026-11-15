# SPEC.md — 流水线规范（唯一契约 · v2.2 / production-grade）

本文件是本仓库的**规范性契约**。所有文档、Go 编排器、Blender 脚本必须与之一致；冲突时以本文件为准。
版本 / URL 的硬性约束另见 [AGENTS.md](AGENTS.md)。

> **v2.2 changelog（相对 v2.1）**：新增 **Skills 子系统**（§8A）——照抄 EL4CTEO/blender-skills（MIT）的 28 个 Agent Skill 文档，实现为 Go 1.27 注册表（`internal/skills`）+ 单个 bpy 调度器（`scripts/blender/skill_run.py`），每个 skill 产出 `build/<id>/contracts/skill-<id>.json` 契约并经 `assetctl skills doctor` 校验；`assetctl verify` 扩为 11 步（新增 skills 注册表与 gofmt 门禁）。**版本收敛为 Blender 5.2 单一目标**：不再探测/兼容 5.1 / 4.x 安装，`doctor` 与 `FindBlender` 只接受 5.2 系列。
>
> **v2.1 changelog（相对 v2）**：新增 §5A 阶段文件契约（10 个 bpy 模块各产出 `contracts/<stage>.json`，含 SHA-256 / 字节数 / FBX 七参数，`assetctl validate` 默认重算校验）；`assetctl verify`（Go 实现，9 步）与 3-OS CI；非 dry-run 六阶段 E2E 集成测试；阶段级 Python 测试（10 模块 + mock_bpy）；fixture GLB 改为含真实三角形；修复 resume manifest 降级、输出流数据竞争、env handoff 校验、Blender 5.x EEVEE API 与跨平台 Blender 发现。语言边界：系统 Python 仅测试 harness，交付逻辑全 Go + bpy。
>
> **v2 changelog（相对 v1）**：`tripo` 升级为**默认生产供应商（DEFAULT Supplier）**，取代原先默认的 `blender` 参数化策略；新增供应商交付机制 `handoff`（`manual | file | fixture`）与环境变量覆盖，让「人在回路」的默认供应商可脚本化、可测试；新增 `internal/runner` 的 `Launcher` 抽象与 `testing/fake_blender.py` 全链路无 Blender 自测；新增 `assetctl doctor` 与 `assetctl validate` 的 `manifest.json` 全 `ok` DoD 门禁；`assetctl init` 新增 `--handoff` / `--source` / `--supplier` 标志。§1/§2 硬约束、六阶段、FBX 七参数、ACT 映射、命名与许可保持不变。

## 1. 目标

把「一张参考图 / 一段参数」变成 Unity（URP）可直接使用的 3D 游戏资产，覆盖人形角色、道具与人形角色动画。

## 2. 硬约束

- **我们自己写的代码 = Go 1.27**：CLI、编排器、策略注册表、配置、校验、参考图处理，全部 Go 1.27，且**零第三方依赖**（仅标准库）。
- **Blender 内部代码 = Blender 5.2 内置 Python（`bpy`）**：通过 `blender --background --factory-startup --python <script> -- ...` 运行。Blender 原生自带什么 Python 就用什么，不假设系统 Python，也不在 Blender 里 `pip install`。（用户原话：「他原生代码用什么就用什么」）
- **Blender 版本 = 5.2，且只有 5.2**（单一目标版本）：`asset.json.blenderVersion` 必须为 `"5.2"`，`FindBlender` 只探测 5.2 安装路径/`PATH`，`CheckBlenderVersion` 要求 `blender --version` 报告 5.2 系列；不为 5.1 / 4.x 提供兼容分支（旧版本 API 差异只在文档中作为迁移提示）。**Tripo 入口 = https://studio.tripo3d.com/**。
- **参考插件 ACT**：https://github.com/mrven/Blender-Asset-Creation-Toolset （GPL-3.0，默认分支 `master`，release 2025.2）。只对齐其行为与参数，**不复制 / 不 vendoring 其源码**，避免 GPL 污染本仓库 MIT 许可。
- 系统 Python 3.14 只用于开发机上的辅助检查（`py_compile` / 测试），不进入交付流水线。

## 3. 多策略（Strategy）与默认供应商（Supplier）

「建模」阶段是**可替换策略**，其余阶段共享。

> **术语等价（仅此一次说明）**：本规范里「**供应商（Supplier）**」与「**建模策略（Strategy）**」指的是同一件事——阶段 1 由谁来产出原始网格。为兼容既有代码与配置，Go 类型仍叫 `Strategy`，`asset.json` 字段仍叫 `strategy`，不做重命名；文档里可自由使用「供应商」这一生产语义。

**`tripo` 是默认生产供应商**（`IsDefault() == true`，`config.Default().Strategy == "tripo"`，`assetctl init` 默认 `--strategy tripo`）。Tripo 为浏览器端半自动、人在回路的图生 3D / 文生 3D / 自动绑定服务，入口固定 **https://studio.tripo3d.com/**。`blender`（参数化 blockout）与 `import`（复制已有文件）是**一等公民的可选策略**，registry 是未来接入其它供应商（`meshy` / `rodin` / `scenario` / 摄影测量）的扩展点。

| id | 名称 | 角色 | 阶段 1 实现 | 自动化程度 | `IsDefault()` |
|----|------|------|-------------|-----------|---------------|
| `tripo` | 浏览器图生 3D（**默认供应商**） | **DEFAULT** | 人工在 https://studio.tripo3d.com/ 上传 → 下载 GLB 到 `raw/<id>.glb`；或经 `handoff` 脚本化交付 | 半自动（人在回路）+ 可脚本化 handoff | `true` |
| `blender` | 参数化 blockout | 可选策略 / CI 基线 | `scripts/blender/stage_01_modeling.py` | 全自动 | `false` |
| `import` | 导入已有模型 | 可选策略 | 复制 `config.source` 到 `raw/<id>.<ext>` | 全自动 | `false` |

扩展方式：实现 `internal/strategy/strategy.go` 的 `Strategy` 接口并 `Register()`。所有新供应商必须返回 `IsDefault() == false`（默认唯一，归 `tripo`）；产物统一落到 `raw/<id>.<ext>`。

```go
type Strategy interface {
    ID() string
    Description() string
    IsDefault() bool
    Acquire(ctx *RunContext) (*AcquireResult, error)
}
```

所有策略的产物统一落到 `raw/<id>.glb`（或 `.fbx` / `.obj`），随后进入**同一条**共享流水线。

## 3A. 供应商交付（Handoff）、可测试性与 DoD 门禁（v2 新增章节）

### 3A.1 Handoff 机制（让默认供应商可脚本化、可测试）

因为默认供应商 `tripo` 是人在回路的，生产环境需要定义良好的**交付（handoff）**机制，使整条流水线可自动化、可回归。`asset.json` 新增 `handoff` 对象：

```json
"handoff": { "mode": "manual", "source": "" }
```

| mode | 语义 | 典型场景 |
|------|------|----------|
| `manual`（默认） | 人工在 https://studio.tripo3d.com/ 上传参考图、下载 GLB 到 `raw/<id>.glb`，再 `assetctl run <id> --from modeling` 续跑。缺文件时打印操作指引并退出码 `3`（`ErrManualStepRequired`） | 正式生产 |
| `file` | 把 `handoff.source`（或环境变量 `ASSETCTL_TRIPO_SOURCE`）复制到 `raw/<id>.<ext>` | 上游 job / webhook 落下 Tripo 导出；`assetctl init <id> --handoff file --source <path>` |
| `fixture` | 生成一个确定性的、最小的合法 GLB（`raw/<id>.glb`） | CI / 测试的逃生舱，使**默认供应商也能完全无人值守**；环境变量 `ASSETCTL_TRIPO_HANDOFF=fixture` 亦可强制 |

环境变量覆盖：`ASSETCTL_TRIPO_HANDOFF`（覆盖 mode）、`ASSETCTL_TRIPO_SOURCE`（覆盖 source 路径）。

`Acquire` 优先级：**(1)** 已存在的 `raw/<id>.{glb,gltf,fbx,obj}` 优先胜出；**(2)** 否则按 handoff mode 处理（含 env 覆盖）；**(3)** 否则给出 manual 指引。

`--from` / `--only` 断点续跑保持不变，尤其 **`--from topology-uv`**（人工/上游补齐 GLB 后跳过 modeling 直接跑下游），是半自动与 handoff 场景的必需能力。

### 3A.2 Launcher 抽象与端到端可测试性

`internal/runner` 引入 `Launcher` 抽象，将「如何拉起一个阶段」与编排逻辑解耦：

| 实现 | 用途 | 运行时 |
|------|------|--------|
| `CommandLauncher` | 生产 | 真实 **Blender 5.2**（`blender --background --factory-startup --python <script> -- ...`，使用 Blender 自带 `bpy`） |
| `ScriptLauncher` | CI / 测试 | 运行 `testing/fake_blender.py`——纯标准库实现的 `bpy` + `mathutils` 模拟器 |

`testing/fake_blender.py` 通过写入 `.blend` / `.fbx` 文件的 JSON 快照在**多个阶段进程之间持久化场景状态**，并用标准库 `zlib` / `struct` 写出**真实 PNG**。因此 `go test ./...` 即可端到端跑完六阶段，**无需安装 Blender 5.2**；`go test ./...` 本身就是完整的自动化验证。

防静默失败：`CommandLauncher` 在每个真实阶段执行后，校验该阶段声明的 `--out` 文件确实存在，缺失即判失败。

### 3A.3 DoD / manifest 门禁与 `doctor`

- `assetctl run` 写 `build/<id>/manifest.json`（阶段状态 + DoD 结果）。
- `assetctl validate` 除既有检查外，额外校验 `build/<id>/manifest.json`：**六阶段必须全部为 `ok`**（DoD 门禁）。
- 新增 `assetctl doctor`：打印工具链报告——Go 版本、Blender 5.2 是否存在 / `ASSETCTL_BLENDER`、已注册供应商 + DEFAULT、Tripo URL、坐标轴契约、`scripts/blender` 阶段脚本是否齐全、退出码。用于生产环境 onboarding 自检。

**一键验证 = `assetctl verify`（Go 1.27 实现，`internal/verify/verify.go`）**，共 11 步：仓库布局（11 个 bpy 脚本 + harness）、skills 注册表一致性、`go.mod` 零第三方依赖校验、`gofmt -l` 格式门禁、`go build ./...`、`go vet ./...`、`go test ./... -count=1 -coverprofile=coverage.out`（Unix 额外 `-race`，Windows 因 cgo 限制不启用 race）、`py_compile` 所有 bpy/测试脚本、`python -m unittest discover -s testing`、`assetctl doctor`、CLI 冒烟。`scripts/verify.py` 只是调用它的兼容 shim（**验证逻辑不在系统 Python 中**），`make verify` 同样转调 Go 命令；`.github/workflows/ci.yml` 为 3-OS 矩阵（ubuntu / macos / windows）+ Go 1.27 + setup-python，并上传 `coverage.out`。

`go test ./...` 本身即完整自动化验证：包含 **非 dry-run 六阶段 E2E**（`ScriptLauncher` 驱动真实 bpy 脚本）、Go 全量单测与 CLI 测试；`python -m unittest discover -s testing` 覆盖 11 个 bpy 模块、28 个 skills、契约与 mock_bpy 自身。**全程不需要安装 Blender 5.2。**

## 4. 游戏美术全流程（六阶段，顺序固定）

| # | 阶段 id | 名称 | 实现 |
|---|---------|------|------|
| 1 | `modeling` | 建模 | 由所选 **Strategy / Supplier** 提供 |
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

退出码：`0` 成功；非 `0` 失败，错误写 stderr。所有脚本必须容忍 `--factory-startup`（不依赖用户偏好）。生产 `CommandLauncher` 在每个阶段后校验 `--out` 文件真实存在（见 §3A.2）。

| 阶段 | `--in` | `--out` |
|------|--------|---------|
| 1 modeling | 参考图 / `config.source` | `work/<id>/01-modeling.blend` |
| 2 topology-uv | `work/<id>/01-modeling.blend` | `work/<id>/02-topology-uv.blend` |
| 3 bake | `work/<id>/02-topology-uv.blend` | `work/<id>/03-bake.blend` + `build/<id>/*_*.png` |
| 4 pbr | `work/<id>/03-bake.blend` | `work/<id>/04-pbr.blend` |
| 5 unity-export | `work/<id>/04-pbr.blend` | `build/<id>/<id>.fbx` + PBR 贴图 |
| 6 render | `build/<id>/<id>.fbx` | `build/<id>/preview.png` |

人形门禁：阶段 5 之后，若 `humanoid: true`，额外跑 `scripts/blender/rig_check.py`。

## 5A. 阶段文件契约（v2.1 新增门禁）

`scripts/blender` 下 **11 个 Python 模块**都能在 `testing/fake_blender.py`（mock_bpy）下独立执行，并在 `build/<id>/contracts/` 产出契约文件：

| 契约 id | 生产者 |
|---------|--------|
| `modeling` / `topology-uv` / `bake` / `pbr` / `unity-export` / `render` | 六阶段脚本 |
| `rig-check` | `rig_check.py`（humanoid 门禁结果） |
| `tripo-to-unity` | `tripo_to_unity.py`（独立 shim） |
| `common` | `common.py`（常量 / FBX 导出参数快照） |
| `humanoid-rig` | `humanoid_rig.py`（独立 rig 构建） |
| `skill-<id>` | `skill_run.py`（28 个 skill 的统一调度器，见 §8A） |

契约 `schema = 1`，字段包含 `asset / stage / input / output / blendOut / tripoUrl / blenderVersion / unityEngine / fbxExport`（§9 七参数）与 `outputs` 数组（每项 `path` 为相对 repo root 的 POSIX 路径、`bytes`、`sha256`）。脚本同时以 `@@JSON@@ <json>` 行协议向 Go / 测试输出结构化摘要（沿用 blender-skills 约定）。

`assetctl validate --expect-contracts=true`（**CLI 默认**）逐份重读文件、重算 SHA-256；缺契约、schema 不符、stage 名不符、空 `outputs`、字节数或摘要不一致均判 DoD 失败。`--expect-contracts=false` 仅用于校验历史资产。该机制使「脚本静默成功但没有真正产物」不可能通过交付门禁。

## 6. 目录约定

```
assets/<id>/asset.json          资产配置（JSON，Go encoding/json）
raw/<id>.*                      策略产物（原始，保留，便于回溯）
work/<id>/NN-<stage>.blend      阶段中间 .blend
build/<id>/<id>.fbx             最终 Unity 资产
build/<id>/<id>_basecolor.png   PBR 贴图（normal/roughness/metallic/ao 同理）
build/<id>/preview.png          阶段 6 渲染合成图（QA 产物，豁免 PBR 后缀检查）
build/<id>/manifest.json        阶段状态 + DoD 结果（六阶段全 ok 才算完成）
build/<id>/contracts/*.json     阶段/skill 契约（schema 1，含 bytes + SHA-256）
work/<id>/skills/<id>.blend     skills 子系统的工作文件
skills/<id>/SKILL.md            照抄的 28 个 Agent Skill 文档（MIT）
skills/<id>/ADAPTATION.md       每个 skill 的 Go CLI / bpy 适配说明
skills/manifest.json            skill 注册表（Go 侧据此校验）
```

> `skills/` 是**知识参考 + 注册表**，不是交付产物；`internal/skills` 在编译期通过 `skills/manifest.json` 与 `scripts/blender/skill_run.py` 的存在性校验仓库完整性。

## 7. `asset.json` 字段

```json
{
  "id": "hero-knight",
  "strategy": "tripo",
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
  "stages": ["modeling", "topology-uv", "bake", "pbr", "unity-export", "render"],
  "handoff": { "mode": "manual", "source": "" }
}
```

> `strategy` 默认 `tripo`；`handoff` 为 v2 新增对象，字段语义见 §3A.1。

## 8. CLI（`assetctl`，Go 1.27）

```
assetctl version
assetctl strategies                          # 列出策略/供应商，标注 DEFAULT（tripo）
assetctl stages                              # 列出六阶段
assetctl init <asset-id> [--strategy tripo] [--supplier <id>] [--handoff <manual|file|fixture>] [--source <path>] [--kind character] [--humanoid]
assetctl run <asset-id> [--strategy <id>] [--from <stage>] [--only <stage,...>] [--dry-run]
assetctl reference prepare <in.png> <outdir> [--size 1024] [--bg 808080]
assetctl validate <build-dir> [--expect-humanoid] [--expect-textures] [--expect-manifest=true|false] [--expect-contracts=true|false]
assetctl blender <script> [-- --args...]     # 透传 Blender 5.2
assetctl doctor                              # 工具链报告（v2 新增）
assetctl skills list|show|run|doctor         # 28 个 Blender skills（v2.2 新增，Go 编排 + bpy 执行）
assetctl verify                              # 一键生产级验证（10 步，Go 实现，v2.2）
```

- `assetctl init` 默认 `--strategy tripo`；`--supplier` 是 `--strategy` 的别名；`--handoff` / `--source` 写入 `asset.json.handoff`（见 §3A.1）。
- `assetctl run` 依次执行六阶段，写 `build/<id>/manifest.json`；任一阶段失败即停止并记录状态。
- `--from` / `--only` 支持断点续跑（Tripo 半自动 / handoff 场景必需，尤其 `--from topology-uv`）。
- `assetctl run` 受 `ASSETCTL_TRIPO_HANDOFF` / `ASSETCTL_TRIPO_SOURCE` 环境变量覆盖；`assetctl doctor` 读取 `ASSETCTL_BLENDER` 定位 Blender 5.2。
- `assetctl validate` 增加 `build/<id>/manifest.json` 六阶段全 `ok` 的 DoD 门禁。

## 8A. Skills 子系统（v2.2 新增）

**定位**：六阶段流水线是「一条固定产线」；Skills 是可单独调用的**操作知识 + 可执行算子**（倒角、布尔、重拓扑、UV 展开、烘焙、Rigify、几何节点、渲染等）。两者共享同一套 Blender 5.2 / Go 1.27 / 契约约束。

- **知识来源**：28 个 `SKILL.md` **原样照抄**自 [EL4CTEO/blender-skills](https://github.com/EL4CTEO/blender-skills)（MIT，副本见 `skills/LICENSE-blender-skills.txt`），仅作规范/知识参考；**不复制其 Python 源码**，实际算子由本仓库用 bpy 重新实现。
- **Go 侧**（`internal/skills`）：`catalog.go` 登记 28 个 skill 的 `id / category / operation / stages`；`manifest.go` 校验 `skills/manifest.json` 与目录树（SKILL.md / ADAPTATION.md / 数量 / 命名）一致；`run.go` 校验参数、拼装 argv、启动 Blender 并**复核契约**（`schema=1`、`stage=skill-<id>`、`operation` 与登记一致）。
- **bpy 侧**（`scripts/blender/skill_run.py`）：单一入口按 `--skill <id>` 分派到 `operation` 实现，执行 2-5 个算子步骤，逐步记录 `steps[]`（`label / ok / detail`），最后落 `build/<id>/contracts/skill-<id>.json` 与 `work/<id>/skills/<id>.blend`。
- **CLI**：
  ```
  assetctl skills list                                  # 28 个 skill（id / category / summary）
  assetctl skills show <id>                             # 元数据 + ADAPTATION.md
  assetctl skills run <id> [--asset <id>] [--param k=v] [--dry-run]
  assetctl skills doctor                                # 目录树 / manifest / catalog / dispatcher 一致性
  ```
- **可测试性**：`testing/test_skills.py` 在 `mock_bpy` 下跑通 **全部 28 个 skill**（每个都产出非空 outputs 且 `result.ok=true`）；`internal/skills/skills_test.go` 覆盖注册表一致性、未知 skill、非法 asset id、契约缺失/串号与 arity 校验；`assetctl verify` 第 2 步即 `skills.ValidateCatalog`。

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

安装方式：Blender Extensions（4.2+，见 https://extensions.blender.org/add-ons/act-game-asset-creation-toolset/ ）或 https://github.com/mrven/Blender-Asset-Creation-Toolset/raw/master/release/act_2025_2_1.zip 。
ACT 2025.2 支持 Blender 3.0–5.0+，可与本机 **Blender 5.2** 并存；其 "Scale for models in Unity is 1" 与本仓库 `scale: 1.0` 一致。

## 11. 命名与 DoD

- 资产 ID：小写连字符 `hero-knight`；clip：`hero-knight-idle`。
- 贴图后缀：`_basecolor` / `_normal` / `_roughness` / `_metallic` / `_ao`；`preview.png` 为 QA 产物，豁免 PBR 后缀检查。
- Humanoid 骨骼（21 根）：`Hips` / `Spine` / `Chest` / `Neck` / `Head` / `Left…` / `Right…`，左右用 `Left`/`Right` 前缀；阶段 5 后 `rig_check.py` 门禁；clip `<id>-idle`。
- DoD：`assetctl validate` 通过（含 `manifest.json` 六阶段全 `ok`、六份阶段契约 SHA-256 一致、交付产物非零且 magic 正确）+ `assetctl doctor` 通过 + `assetctl skills doctor` 通过 + `assetctl verify` 全绿 + Unity 中无粉材质、无 T-pose、Avatar 映射无红项（SPEC.md §3A.3 / §5A / §8A）。

## 12. 许可证

本仓库 **MIT**（仓库地址 https://github.com/neko233-com/skill233-create-3d-game-asset-pipeline ）。ACT 为 **GPL-3.0**，仅作为可选外部插件在文档中引用，不纳入本仓库代码。提交遵循 **Conventional Commits**，仅在 **`main`** 分支开发。
