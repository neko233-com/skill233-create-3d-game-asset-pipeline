# 10 · 多策略详解（建模阶段可替换）

> 唯一契约：[SPEC.md §3](../SPEC.md)。本文所有版本/URL 以 [AGENTS.md](../AGENTS.md) 与 SPEC.md 为准：Go **1.27**、Blender **5.2**、Tripo **https://studio.tripo3d.com/**。

## 1. 核心概念

「建模」（阶段 1 `modeling`）是**可替换策略**，其余五个阶段（拓扑/UV、烘焙、PBR、Unity 导出/同步、渲染合成）**共享同一条流水线**。Tripo 只是策略之一，**Blender 是默认策略**。

- 策略只负责「把参考图 / 参数 / 已有模型，变成一个原始网格文件」。
- 所有策略的产物统一落到 `raw/<id>.glb`（或 `.fbx` / `.obj`），随后进入同一条阶段 2→6。
- 好处：换生成源（换 AI 平台、换外包模型、换参数化套件）不影响下游减面、UV、烘焙、导出、渲染逻辑；TA 只需维护一个 `Strategy` 实现。

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
| `ID() string` | 策略唯一 id，与 `asset.json.strategy` 对应（如 `blender` / `tripo`） |
| `Description() string` | 供 `assetctl strategies` 展示的一句话说明 |
| `IsDefault() bool` | 是否为默认策略；**仅 `blender` 返回 true** |
| `Acquire(ctx *RunContext) (*AcquireResult, error)` | 执行「获取原始模型」：产出/定位 `raw/<id>.<ext>`，返回产物路径；失败写错误，非 0 退出 |

`RunContext` 携带 `asset.json` 解析后的配置、工作目录、阶段上下文；`AcquireResult` 至少包含产物路径与元信息。实现约定：

- 我们自己写的代码一律 **Go 1.27**，**零第三方依赖（仅标准库）**。
- 策略**不直接调用 Blender `bpy`**；需要 Blender 能力时通过统一的 `blender --background --factory-startup --python` 约定调用（见 [61](61-language-split.md)）。
- 产物一律落到 `raw/` 并保留，便于回溯；不得改写上游参考图。

## 3. 三个内置策略对比表

| id | 名称 | 阶段 1 实现 | 自动化程度 | 输入 | 产物 | 适用场景 |
|----|------|-------------|-----------|------|------|----------|
| `blender` | 参数化建模（**默认**） | `scripts/blender/stage_01_modeling.py` | **全自动** | `asset.json.modeling`（`primitive` / `size` / `subdivisions`） | `work/<id>/01-modeling.blend` | 无外部依赖、可重复、CI 友好、块灰模/占位资产 |
| `tripo` | 浏览器图生 3D | 人工在 https://studio.tripo3d.com/ 上传 → 下载 GLB 到 `raw/<id>.glb` | **半自动**（Go 检测文件、给指引、可断点续跑） | 归一化参考图 `raw/<id>.png` | `raw/<id>.glb` | 有概念图、要 AI 生成带纹理的角色/道具 |
| `import` | 导入已有模型 | 复制 `config.source` 到 `raw/<id>.<ext>` | **全自动** | `asset.json.source` 指向已有 GLB/FBX/OBJ | `raw/<id>.<ext>` | 外包模型、已购资产、手工建模成品入库 |

> 注意：按 SPEC.md §5 阶段契约，策略产物统一落到 `raw/<id>.*`，而阶段 1 的 `--out` 是 `work/<id>/01-modeling.blend`。即策略先把原始文件放进 `raw/`，阶段 1 脚本再把它规整为 `.blend` 交给阶段 2。

## 4. 为什么 `blender` 是默认策略

1. **零外部依赖、完全可重复**：`blender` 参数化建模不需要账号、网络、浏览器、人工点击；`assetctl run` 一次跑通六阶段，最适合 CI 与回归测试。
2. **确定性强**：输入是 `asset.json` 里的 `primitive` / `size` / `subdivisions`，输出几何可预测，方便给下游减面/UV/烘焙写断言。
3. **Tripo 是半自动**：`tripo` 需要人工在 https://studio.tripo3d.com/ 上传、等待、下载，天然有「人在回路」，不适合做默认无人值守路径。
4. **默认策略是基线**：新接入的 AI 平台 / 摄影测量等都以 `blender` 产物的格式与命名作为对齐基准，下游无需分支。

`IsDefault() == true` 只应有 `blender` 一个；`assetctl init <asset-id>` 未显式指定 `--strategy` 时即采用它。

## 5. `tripo` 半自动流程（断点续跑）

`tripo` 是「人在回路」策略：Go 不直接调用 Tripo API，而是检测 `raw/` 下文件是否就绪，给出指引，并在文件到位后从断点继续。

### 5.1 参考图准备（Go 侧，全自动）

```bash
assetctl reference prepare concept.png raw/ --size 1024 --bg 808080
```

产出归一化正方形图 `raw/<id>.png`（主体单一、正面、干净背景、正方形，见 [20](20-stage-01-modeling.md)）。

### 5.2 人工在浏览器完成（Tripo 侧）

1. 打开 **https://studio.tripo3d.com/** 并登录。
2. 选择 **Image-to-3D**，上传 `raw/<id>.png`。
3. 生成完成后预览，检查正面结构完整、无破面、贴图无严重拉伸。
4. 下载 **GLB**（单文件含贴图，Blender 5.2 导入最省事）到 `raw/<id>.glb`。

### 5.3 Go 断点续跑

Go runner 检测到 `raw/<id>.glb` 存在后，阶段 1 完成；后续阶段照常跑。人工环节中断过、或只想重跑后半段，用 `--from` / `--only`：

```bash
# 先跑一次（若 GLB 未就位会提示并停在该阶段）
assetctl run hero-knight --strategy tripo

# 人工下载 GLB 到 raw/hero-knight.glb 后，从建模阶段续跑全程
assetctl run hero-knight --strategy tripo --from modeling

# 只补跑建模这一个阶段
assetctl run hero-knight --strategy tripo --only modeling

# 先演练，不真正执行
assetctl run hero-knight --strategy tripo --from modeling --dry-run
```

`--from` / `--only` 支持断点续跑（Tripo 半自动场景必需）；`assetctl run` 依次执行六阶段并写 `build/<id>/manifest.json`，任一阶段失败即停止并记录状态。详见 [80-cli-reference.md](80-cli-reference.md)。

## 6. 如何扩展新策略（scenario / meshy / rodin）

SPEC.md §3 列出的典型可扩展项：`scenario` / `meshy` / `rodin` / 摄影测量。步骤：

1. 在 `internal/strategy/` 新建 Go 1.27 文件，实现 `Strategy` 接口的四个方法（仅标准库，零第三方依赖）。
2. `ID()` 返回新 id（如 `meshy`），`IsDefault()` 返回 `false`（默认唯一，归 `blender`）。
3. `Acquire()` 负责获取模型并把产物落到 `raw/<id>.<ext>`；若需 Blender 处理，走统一无头调用约定，不在 Go 里内嵌 `bpy`。
4. 在包初始化处 `Register()` 注册。
5. `assetctl strategies` 应能列出它；`asset.json.strategy` 填该 id 即可切换。

| 候选 | 说明 | 注意 |
|------|------|------|
| `scenario` | 接 Scenario 平台的图生/文生 3D 能力 | 仍要把产物规整为 `raw/<id>.glb` |
| `meshy` | Meshy 图生/文生 3D、retexture、remesh | 保持产物命名与坐标系约定 |
| `rodin` | Rodin / 摄影测量类高模输入 | 高模正好作为阶段 3「高模→低模」烘焙源 |

> 扩展策略**不改变**下游五阶段，也**不改变** `asset.json` 里 `axes` / `budget` / `textures` 的契约。任何新策略都要产出 Unity 可用的 `-Z forward / Y up / scale 1.0` 基线的原始文件。

## 7. 在 `asset.json` 里切换策略

`asset.json` 位于 `assets/<id>/asset.json`，字段见 SPEC.md §7。切换只改 `strategy` 一个字段：

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
  "stages": ["modeling", "topology-uv", "bake", "pbr", "unity-export", "render"]
}
```

三种写法：

| 想用的策略 | `strategy` 值 | 还需设置的字段 |
|-----------|---------------|----------------|
| 默认参数化建模 | `"blender"`（或省略） | `modeling`（`primitive` / `size` / `subdivisions`） |
| Tripo 图生 3D | `"tripo"` | `referenceImage`（`raw/<id>.png`）；人工下载 GLB 到 `raw/<id>.glb` |
| 导入已有模型 | `"import"` | `source`（指向已有 GLB/FBX/OBJ） |

也可用 `assetctl init` 一次性生成带策略的配置：

```bash
assetctl init hero-knight --strategy tripo --kind character --humanoid
assetctl strategies          # 列出策略，DEFAULT 标注为 blender
```

切换策略后，`assetctl validate` 与 DoD 清单（[70](70-naming-and-dod.md)）不变——所有策略共用同一套交付门禁。
