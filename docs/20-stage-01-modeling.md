# 20 · 阶段 ① 建模（modeling）

> 阶段 id：`modeling`。由所选 **Strategy** 提供实现，其余阶段共享。
> 硬约束：Blender **5.2**、Tripo **https://studio.tripo3d.com/**、编排代码 Go **1.27**。契约见 [SPEC.md §3/§4/§5](../SPEC.md)。

## 1. 阶段职责与产物契约

建模阶段把「参考图 / 参数 / 已有模型」变成**原始网格**，是六阶段里唯一可替换策略的阶段。

| 项 | 值 |
|----|----|
| Go runner 传参 | `--asset <id> --config <asset.json> --in <参考图/config.source> --out work/<id>/01-modeling.blend` |
| 策略原始产物 | `raw/<id>.glb`（或 `.fbx` / `.obj`），**保留不删**，便于回溯 |
| 阶段 `--out` | `work/<id>/01-modeling.blend` |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
| 运行环境 | Blender 5.2 内置 Python（`bpy`），`--factory-startup` 容忍 |

三种策略 / 供应商（「供应商（Supplier）== 建模策略（Strategy）」，详见 [10-strategies.md](10-strategies.md)）：

| 策略 | 本节对应小节 | 自动化 |
|------|-------------|--------|
| `tripo`（**默认供应商 / DEFAULT**，半自动、人在回路） | §3 图生 3D + §3.0 handoff 三模式 | 半自动（经 `handoff` 可脚本化 / `fixture` 无人值守） |
| `blender`（可选策略 / CI 基线） | §2 参数化建模 | 全自动 |
| `import`（可选策略） | §4 导入已有模型 | 全自动 |

**v2 起 `tripo` 是默认生产供应商**（`IsDefault() == true`、`config.Default().Strategy == "tripo"`、`assetctl init` 默认 `--strategy tripo`）；`blender` 参数化建模退居为全自动的可选策略 / CI 基线。默认供应商是半自动的，靠 `asset.json.handoff` 交付机制补齐脚本化能力（见 §3.0）。

## 2. `blender` 策略：Blender 5.2 参数化建模（可选策略 / CI 基线）

一等公民的**可选策略**。`scripts/blender/stage_01_modeling.py` 在 Blender 5.2 内置 Python 中运行，按 `asset.json.modeling` 生成基础体块，再规整为 `.blend`。

配置字段（SPEC.md §7）：

```json
"modeling": { "primitive": "cube", "size": 1.0, "subdivisions": 2 }
```

| 字段 | 含义 | 示例 |
|------|------|------|
| `primitive` | 基础几何体类型 | `cube` / `sphere` / `cylinder` / `plane` |
| `size` | 尺寸（Blender 单位） | `1.0` |
| `subdivisions` | 细分级别（给后续 Decimate / 高模烘焙留余量） | `2` |

典型用途：块灰模（blockout）、占位资产、CI 回归基线、面数预算的下界测试。它是全自动、不需要账号/网络/浏览器的可选策略（`IsDefault() == false`）。

调用约定（由 `assetctl run` 统一发起，不需手工敲）：

```
blender --background --factory-startup --python scripts/blender/stage_01_modeling.py -- \
  --asset <id> --config assets/<id>/asset.json \
  --in raw/<id>.png --out work/<id>/01-modeling.blend
```

> 脚本必须容忍 `--factory-startup`（不依赖用户偏好、插件、主题）。

## 3. `tripo` 策略：Tripo 图生 3D（**默认生产供应商 / DEFAULT**）

Tripo 入口统一为 **https://studio.tripo3d.com/**（浏览器端，无需安装，Image-to-3D / Text-to-3D / auto-rig）。它是**默认供应商**：`IsDefault() == true`、`config.Default().Strategy == "tripo"`、`assetctl init` 默认 `--strategy tripo`。Go 侧负责参考图归一化、文件检测与 handoff；**`manual` 模式下生成动作仍在浏览器里人工完成**（半自动、人在回路），但 `file` / `fixture` 模式可跳过人工。

### 3.0 三条 handoff 交付路径（默认供应商可脚本化的关键）

`asset.json` 的 `handoff` 对象：`"handoff": { "mode": "manual", "source": "" }`。`Acquire` 优先级固定：

1. **已存在** `raw/<id>.{glb,gltf,fbx,obj}` 直接采用；
2. 否则按 **handoff mode** 执行；
3. 否则回落 **manual 指引**。

| `mode` | 行为 | 命令 / 场景 |
|--------|------|-------------|
| `manual`（**默认**） | 人工上传参考图、下载 GLB 到 `raw/<id>.glb`，再 `assetctl run <id> --from modeling` 续跑；**缺文件打印指引并 exit `3`（`ErrManualStepRequired`）** | 生产人工审片 |
| `file` | 把 `handoff.source`（或环境变量 `ASSETCTL_TRIPO_SOURCE`）复制到 `raw/<id>.<ext>` | `assetctl init <id> --handoff file --source <path>`；上游 job / webhook 落盘了 Tripo 导出 |
| `fixture` | 生成**确定性最小合法 GLB**（`raw/<id>.glb`），使默认供应商完全无人值守 | `assetctl init <id> --handoff fixture`；环境变量 `ASSETCTL_TRIPO_HANDOFF=fixture` 亦可强制；**CI 用 `--handoff fixture` 才能无人值守** |

环境变量覆盖：`ASSETCTL_TRIPO_HANDOFF`（覆盖 mode）、`ASSETCTL_TRIPO_SOURCE`（覆盖 source）。完整契约见 [10-strategies.md](10-strategies.md) §5 与 [SPEC.md §3A](../SPEC.md)。

### 3.1 参考图要求（决定生成质量）

| 要求 | 说明 |
|------|------|
| **主体单一** | 一张图只有一个物体/角色，避免遮挡与多主体（提示词里写 `single ... only`） |
| **正面视角** | 人形用正面或 3/4 正面站姿，四肢尽量分开（避免 A/T-pose 粘连） |
| **干净背景** | 纯色或透明背景，减少模型把背景「烤」进几何体 |
| **正方形** | 归一化到 1024×1024，光线均匀、避免强阴影与反射高光 |

参考图归一化（Go 1.27 实现，零第三方依赖）：

```bash
assetctl reference prepare concept.png raw/ --size 1024 --bg 808080
```

产出 `raw/<id>.png`。提示词模板见 `examples/tripo_prompts.md`。

### 3.2 具体 UI 步骤

1. 打开 **https://studio.tripo3d.com/** 并登录。
2. 选择 **Image-to-3D**（图生 3D）；无参考图时可选 **Text-to-3D**。
3. 上传归一化后的参考图 `raw/<id>.png`。
4. 关键参数（以当前 UI 为准）：
   - 生成模式：标准 / 高质量（角色建议高质量）
   - 是否需要纹理：开启（PBR 贴图）
   - 面数：默认即可，后续在 Blender 阶段 2 减面
5. 生成完成后预览，检查：正面结构完整、无明显破面、贴图无严重拉伸。
6. **下载 GLB**（单文件含贴图，Blender 5.2 导入最省事），保存到 `raw/<id>.glb`。

### 3.3 自动绑定与动画（人形，可选）

需要角色动画时，可优先在 https://studio.tripo3d.com/ 内完成**自动绑定（auto-rig）与动画**，导出带骨骼的模型；其骨骼命名可能与 Unity Humanoid 不完全一致，需要重定向。详见 [50-humanoid-animation.md](50-humanoid-animation.md)。也可导出静态 GLB 后在 Blender 手动绑定。

### 3.4 半自动断点续跑

`manual` 模式下 Go 检测 `raw/<id>.glb` 是否就位；`file` / `fixture` 模式则先复制 / 生成产物再照常跑：

```bash
assetctl run <id>                                        # 默认 strategy=tripo；manual 缺文件时给指引并 exit 3
assetctl run <id> --from modeling                        # 人工下载 GLB（或从 topology-uv 等断点）后续跑
assetctl run <id> --from topology-uv                     # 尤其常用：跳过建模，从拓扑/UV 续跑
assetctl run <id> --only modeling                        # 只补跑阶段 1
assetctl run <id> --from modeling --dry-run              # 演练
ASSETCTL_TRIPO_HANDOFF=fixture assetctl run <id>         # CI 强制无人值守
```

每个真实阶段执行后 runner 还会校验声明的 `--out` 文件确实存在（防静默失败）。`--from` / `--only` 断点续跑详见 [80-cli-reference.md](80-cli-reference.md)。

### 3.5 常见坑

| 现象 | 原因 / 处理 |
|------|-------------|
| 生成物带底座 / 背景 | 参考图背景不够干净；阶段 2 清理步骤删除无关网格 |
| 贴图糊 / 拉伸 | 参考图分辨率太低或视角太偏；重做参考图 |
| 模型朝向不对 | 不要依赖 Tripo 朝向；阶段 5 统一按 `-Z forward / Y up` 导出（见 [42](42-stage-05-unity-export.md)） |

## 4. `import` 策略：导入已有模型

全自动。把 `asset.json.source` 指向的已有模型（GLB / FBX / OBJ）复制到 `raw/<id>.<ext>`，随后进入共享流水线。

```json
"strategy": "import",
"source": "D:/art/outsource/hero-knight.fbx"
```

适用于外包模型、已购资产、手工建模成品入库。注意：

- 复制而非移动，原始文件保持不动。
- 导入后仍要过阶段 2 的清理、减面、UV 与阶段 5 的坐标系规整——**不要假设外部模型的缩放/朝向/命名已合规**。

## 5. 产物契约汇总

| 产物 | 位置 | 谁产生 | 说明 |
|------|------|--------|------|
| 归一化参考图 | `raw/<id>.png` | `assetctl reference prepare`（Go 1.27） | 正方形、中性背景 |
| 策略原始模型 | `raw/<id>.glb` / `.fbx` / `.obj` | `Strategy.Acquire()` | **保留，不删**，便于回溯 |
| 阶段 1 工作文件 | `work/<id>/01-modeling.blend` | `stage_01_modeling.py`（Blender 5.2 / `bpy`） | 交给阶段 2 的 `--in` |

下一步：[30-stage-02-topology-uv.md](30-stage-02-topology-uv.md)（拓扑 / UV）。
