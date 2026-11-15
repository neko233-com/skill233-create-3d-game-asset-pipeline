# 10 · 多策略详解（建模阶段可替换）

> 唯一契约：[SPEC.md §3](../SPEC.md)。本文所有版本/URL 以 [AGENTS.md](../AGENTS.md) 与 SPEC.md 为准：Go **1.27**、Blender **5.2**、Tripo **https://studio.tripo3d.com/**。

## 1. 核心概念

「建模」（阶段 1 `modeling`）是**可替换策略**，其余五个阶段（拓扑/UV、烘焙、PBR、Unity 导出/同步、渲染合成）**共享同一条流水线**。**`tripo` 是默认生产供应商（DEFAULT Supplier）**（`IsDefault() == true`、`config.Default().Strategy == "tripo"`、`assetctl init` 默认 `--strategy tripo`），半自动、人在回路；`blender`（参数化 blockout）与 `import`（复制已有文件）仍是一等公民的可选策略。

> **术语等价（仅在此说明一次）**：文档里的「**供应商（Supplier）**」与代码里的「**建模策略（Strategy）**」指的是同一件事——都是阶段 1 负责产出原始网格文件的实现。Go 的 `Strategy` 类型**保持原名**，不因为文档改叫「供应商」而重命名。`assetctl strategies` 列出的每一项，既是策略也是供应商。

- 策略只负责「把参考图 / 参数 / 已有模型，变成一个原始网格文件」。
- 所有策略的产物统一落到 `raw/<id>.glb`（或 `.fbx` / `.obj`），随后进入同一条阶段 2→6。
- 好处：换生成源（换 AI 平台、换外包模型、换参数化套件）不影响下游减面、UV、烘焙、导出、渲染逻辑；TA 只需维护一个 `Strategy` 实现。
- **默认供应商是人在回路的**，因此 v2 引入 `asset.json.handoff` 交付机制（`manual | file | fixture` + 环境变量覆盖），把「人工环节」补齐为可脚本化、可 CI 无人值守的路径，详见 §5。

## 2. `Strategy` 接口（Go 1.27）

扩展点在 `internal/strategy/strategy.go`，实现接口并 `Register()` 即可被 `assetctl strategies` 列出。SPEC.md §3 给出的签名：

```go
type Strategy interface {
    ID() string
    Description() string
    IsDefault() bool
    Acquire(ctx *RunContext) (*AcquireResult, error)
}
```

| 方法 | 语义 |
|------|------|
| `ID() string` | 策略唯一 id，与 `asset.json.strategy` 对应（如 `tripo` / `blender` / `import`） |
| `Description() string` | 供 `assetctl strategies` 展示的一句话说明 |
| `IsDefault() bool` | 是否为默认供应商；**仅 `tripo` 返回 `true`** |
| `Acquire(ctx *RunContext) (*AcquireResult, error)` | 执行「获取原始模型」：按 §5 的优先级产出/定位 `raw/<id>.<ext>`，返回产物路径；失败写错误，非 0 退出 |

`RunContext` 携带 `asset.json` 解析后的配置、工作目录、阶段上下文；`AcquireResult` 至少包含产物路径与元信息。实现约定：

- 我们自己写的代码一律 **Go 1.27**，**零第三方依赖（仅标准库）**。
- 策略**不直接调用 Blender `bpy`**；需要 Blender 能力时通过统一的 `blender --background --factory-startup --python` 约定调用（见 [61](61-language-split.md)）。
- 产物一律落到 `raw/` 并保留，便于回溯；不得改写上游参考图。

## 3. 三个内置策略对比表

| id | 名称 | 阶段 1 实现 | 自动化程度 | 输入 | 产物 | 适用场景 |
|----|------|-------------|-----------|------|------|----------|
| `tripo` | 浏览器图生 3D（**默认供应商 / DEFAULT**，`IsDefault()==true`） | 人工在 https://studio.tripo3d.com/ 上传 → 下载 GLB 到 `raw/<id>.glb`；或经 `handoff` 三模式交付 | **半自动**（人在回路；`fixture` 模式可完全无人值守） | 归一化参考图 `raw/<id>.png` | `raw/<id>.glb` | 有概念图、要 AI 生成带纹理的角色/道具（生产默认） |
| `blender` | 参数化建模（可选策略） | `scripts/blender/stage_01_modeling.py` | **全自动** | `asset.json.modeling`（`primitive` / `size` / `subdivisions`） | `work/<id>/01-modeling.blend` | 无外部依赖、可重复、CI 基线、块灰模/占位资产 |
| `import` | 导入已有模型（可选策略） | 复制 `config.source` 到 `raw/<id>.<ext>` | **全自动** | `asset.json.source` 指向已有 GLB/FBX/OBJ | `raw/<id>.<ext>` | 外包模型、已购资产、手工建模成品入库 |

> 注意：按 SPEC.md §5 阶段契约，策略产物统一落到 `raw/<id>.*`，而阶段 1 的 `--out` 是 `work/<id>/01-modeling.blend`。即策略先把原始文件放进 `raw/`，阶段 1 脚本再把它规整为 `.blend` 交给阶段 2。

## 4. 为什么 `tripo` 是默认生产供应商

1. **生产价值最高**：交付目标是「能进 Unity 的游戏资产」，Tripo 的 Image-to-3D / Text-to-3D 能从一张概念图直接产出带纹理、可 auto-rig 的角色 / 道具，比参数化 blockout 更接近成品美术，因此把它定为**默认生产供应商**。
2. **供应商可替换、下游不分支**：默认不是「独占」，只是 `IsDefault()==true` 的注册项。`blender` / `import` 仍是可选策略，registry 是 `meshy` / `rodin` / `scenario` / 摄影测量等未来供应商的扩展点，切换只改 `asset.json.strategy` 一个字段。
3. **半自动被 handoff 补齐**：`tripo` 天然有「人在回路」（浏览器上传、等待、下载），看似不适合自动化；v2 用 `handoff`（`manual | file | fixture` + 环境变量）把这个环节显式契约化，使**默认供应商也能脚本化、可无人值守测试**（`fixture` 是 CI 逃生舱）。
4. **CI 仍可全量回归**：`go test ./...` 借助 `ScriptLauncher` + `testing/fake_blender.py`（纯标准库 `bpy` 模拟器）端到端跑通六阶段，**无需安装 Blender 5.2**；见 [80-cli-reference.md](80-cli-reference.md) 与 SPEC.md §3A。

`IsDefault() == true` 只应有 `tripo` 一个；`assetctl init <asset-id>` 未显式指定 `--strategy` 时即采用它。`assetctl strategies` 会把 `tripo` 标注为 `DEFAULT`。

## 5. `handoff` 交付机制（默认供应商可脚本化的关键）

因为默认供应商是人在回路的，生产需要定义良好的「交接」点，让流水线可自动化、可测试。`asset.json` 新增 `handoff` 对象（SPEC.md §7）：

```json
"handoff": { "mode": "manual", "source": "" }
```

### 5.1 三种模式

| `mode` | 行为 | 典型场景 |
|--------|------|----------|
| `manual`（**默认**） | 人工在 https://studio.tripo3d.com/ 上传参考图，下载 GLB 到 `raw/<id>.glb`，然后 `assetctl run <id> --from modeling` 续跑；**缺文件时打印指引并以退出码 `3`（`ErrManualStepRequired`）停止** | 生产环境人工审片后交付 |
| `file` | 把 `handoff.source`（或环境变量 `ASSETCTL_TRIPO_SOURCE`）指向的文件复制到 `raw/<id>.<ext>`：`assetctl init <id> --handoff file --source <path>` | 上游 job / webhook 落盘了 Tripo 导出 |
| `fixture` | 生成一个**确定性最小合法 GLB**（`raw/<id>.glb`），使默认供应商完全无人值守；环境变量 `ASSETCTL_TRIPO_HANDOFF=fixture` 也可强制 | CI / 测试逃生舱 |

### 5.2 环境变量覆盖

| 变量 | 作用 |
|------|------|
| `ASSETCTL_TRIPO_HANDOFF` | 覆盖 `handoff.mode`（如设为 `fixture` 强制无人值守） |
| `ASSETCTL_TRIPO_SOURCE` | 覆盖 `handoff.source`（`file` 模式的来源路径） |

### 5.3 `Acquire` 优先级（固定）

1. **已存在** `raw/<id>.{glb,gltf,fbx,obj}` —— 直接采用（人工下载 / 上游落盘的结果优先，最便于回溯）；
2. 否则按 **handoff mode** 执行（`manual` 给指引并 exit `3` / `file` 复制 / `fixture` 生成确定性 GLB）；
3. 否则回落到 **manual 指引**。

> **CI 用 `--handoff fixture`（或 `ASSETCTL_TRIPO_HANDOFF=fixture`）才能无人值守**——这是默认供应商在 CI 里跑通的唯一推荐方式。`--from` / `--only` 断点续跑依然适用（尤其 `--from topology-uv`），见 §6。

## 6. `tripo` 半自动流程（断点续跑）

`tripo` 是「人在回路」策略：Go 不直接调用 Tripo API，而是检测 `raw/` 下文件是否就绪（或按 `handoff` 生成/复制），给出指引，并在文件到位后从断点继续。

### 6.1 参考图准备（Go 侧，全自动）

```bash
assetctl reference prepare concept.png raw/ --size 1024 --bg 808080
```

产出归一化正方形图 `raw/<id>.png`（主体单一、正面、干净背景、正方形，见 [20](20-stage-01-modeling.md)）。

### 6.2 人工在浏览器完成（`mode: manual`，Tripo 侧）

1. 打开 **https://studio.tripo3d.com/** 并登录。
2. 选择 **Image-to-3D**，上传 `raw/<id>.png`。
3. 生成完成后预览，检查正面结构完整、无破面、贴图无严重拉伸。
4. 下载 **GLB**（单文件含贴图，Blender 5.2 导入最省事）到 `raw/<id>.glb`。

`mode: file` / `mode: fixture` 则跳过浏览器：`file` 复制上游导出，`fixture` 生成确定性 GLB，二者都不需要人工点击。

### 6.3 Go 断点续跑

Go runner 检测到 `raw/<id>.glb` 存在（或 `handoff` 已产出）后，阶段 1 完成；后续阶段照常跑。人工环节中断过、或只想重跑后半段，用 `--from` / `--only`：

```bash
# 先跑一次（manual 模式下若 GLB 未就位会提示并 exit 3）
assetctl run hero-knight --strategy tripo

# 人工下载 GLB 到 raw/hero-knight.glb 后，从建模阶段续跑全程
assetctl run hero-knight --strategy tripo --from modeling

# 只补跑建模这一个阶段
assetctl run hero-knight --strategy tripo --only modeling

# 先演练，不真正执行
assetctl run hero-knight --strategy tripo --from modeling --dry-run

# 尤其常用：跳过建模，直接从拓扑/UV 续跑
assetctl run hero-knight --from topology-uv
```

`--from` / `--only` 支持断点续跑（Tripo 半自动场景必需）；`assetctl run` 依次执行六阶段并写 `build/<id>/manifest.json`，任一阶段失败即停止并记录状态，每个真实阶段后 runner 还会校验声明的 `--out` 文件确实存在（防静默失败）。详见 [80-cli-reference.md](80-cli-reference.md)。

## 7. 如何扩展新策略（scenario / meshy / rodin）

SPEC.md §3 列出的典型可扩展项：`scenario` / `meshy` / `rodin` / 摄影测量。步骤：

1. 在 `internal/strategy/` 新建 Go 1.27 文件，实现 `Strategy` 接口的四个方法（仅标准库，零第三方依赖）。
2. `ID()` 返回新 id（如 `meshy`），**`IsDefault()` 返回 `false`**（默认唯一，归 `tripo`）。
3. `Acquire()` 负责获取模型并把产物落到 `raw/<id>.<ext>`；若需 Blender 处理，走统一无头调用约定，不在 Go 里内嵌 `bpy`。
4. 在包初始化处 `Register()` 注册。
5. `assetctl strategies` 应能列出它；`asset.json.strategy` 填该 id 即可切换。

| 候选 | 说明 | 注意 |
|------|------|------|
| `scenario` | 接 Scenario 平台的图生/文生 3D 能力 | 仍要把产物规整为 `raw/<id>.glb` |
| `meshy` | Meshy 图生/文生 3D、retexture、remesh | 保持产物命名与坐标系约定 |
| `rodin` | Rodin / 摄影测量类高模输入 | 高模正好作为阶段 3「高模→低模」烘焙源 |

> 扩展策略**不改变**下游五阶段，也**不改变** `asset.json` 里 `axes` / `budget` / `textures` 的契约。任何新策略都要产出 Unity 可用的 `-Z forward / Y up / scale 1.0` 基线的原始文件。

## 8. 在 `asset.json` 里切换策略

`asset.json` 位于 `assets/<id>/asset.json`，字段见 SPEC.md §7。切换只改 `strategy` 一个字段；默认供应商还要按需配置 `handoff`：

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
  "handoff": { "mode": "manual", "source": "" },
  "stages": ["modeling", "topology-uv", "bake", "pbr", "unity-export", "render"]
}
```

三种写法：

| 想用的策略 | `strategy` 值 | 还需设置的字段 |
|-----------|---------------|----------------|
| 默认供应商 Tripo | `"tripo"`（或省略，即默认值） | `referenceImage`（`raw/<id>.png`）；`handoff`（`manual` 人工下载 GLB / `file` 指向上游导出 / `fixture` 生成确定性 GLB） |
| 参数化建模（可选） | `"blender"` | `modeling`（`primitive` / `size` / `subdivisions`） |
| 导入已有模型（可选） | `"import"` | `source`（指向已有 GLB/FBX/OBJ） |

也可用 `assetctl init` 一次性生成带策略与 handoff 的配置：

```bash
assetctl init hero-knight --kind character --humanoid                       # 默认 strategy=tripo、handoff.mode=manual
assetctl init hero-knight --handoff file --source D:/art/knight.glb         # 上游导出落盘
assetctl init hero-knight --handoff fixture                                 # CI 无人值守
assetctl strategies          # 列出供应商，DEFAULT 标注为 tripo
```

切换策略后，`assetctl validate` 与 DoD 清单（[70](70-naming-and-dod.md)）不变——所有策略共用同一套交付门禁；`assetctl validate` 还会检查 `build/<id>/manifest.json` 六阶段全 `ok`。
