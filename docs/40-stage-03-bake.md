# 40 · 阶段 ③ 烘焙（bake）

> 阶段 id：`bake`。实现：`scripts/blender/stage_03_bake.py`（Blender **5.2** 内置 Python / `bpy`）。
> 输入 `work/<id>/02-topology-uv.blend` → 输出 `work/<id>/03-bake.blend` **+ `build/<id>/*_*.png`**。契约见 [SPEC.md §5/§10](../SPEC.md)。

## 1. 阶段职责

把高模细节 / 表面信息「烤」成贴图，供低模在 Unity URP 里实时使用。这是六阶段里**唯一直接向 `build/` 目录写 PNG** 的阶段（后续 PBR 阶段整理、Unity 导出阶段随 FBX 一并交付）。

| 项 | 值 |
|----|----|
| `--in` | `work/<id>/02-topology-uv.blend` |
| `--out` | `work/<id>/03-bake.blend` |
| 附加产物 | `build/<id>/*_*.png`（烘焙贴图） |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
| 运行 | `blender --background --factory-startup --python scripts/blender/stage_03_bake.py -- --asset <id> --config <asset.json> --in ... --out ...` |

## 2. 烘焙什么

| 烘焙类型 | 内容 | 用途 |
|----------|------|------|
| **Normal（法线）** | 高模表面法线 → 切线空间法线图 | 低模保留高模的凹凸细节，Unity URP/Lit 的 Normal Map |
| **AO（环境光遮蔽）** | 缝隙/角落的自遮蔽 | 叠加到 Basecolor 或 URP/Lit 的 Occlusion，增强体积感 |
| **Curvature（曲率）** | 凸边/凹边信息 | 做边缘磨损、脏迹、遮罩（mask），给阶段 4 的 PBR / Palette 用 |
| **高模 → 低模（Detail Transfer）** | 把高模（AI 生成原始网格 / 摄影测量 / 细分体）的细节投到阶段 2 减面后的低模 | 游戏运行时只看低模 + 烘焙图，兼顾面数预算与观感 |

典型来源：
- `tripo` / `import` 策略的原始高模作为「High」，阶段 2 Decimate 后的低模作为「Low」。
- `blender` 默认策略的 `subdivisions` 细分体可作为高模源，低模为应用 Decimate 后的结果。

> 烘焙依赖阶段 2 的**主 UV 无重叠**与**足够的 Island Margin / padding**，否则会串色、漏细节。低模与高模需对齐到同一空间（原点/缩放已应用）。

## 3. 贴图尺寸与通道

| 项 | 约定 |
|----|------|
| **尺寸** | 由 `asset.json.budget.maxTextureSize` 控制（默认 **2048**）；移动端建议 1024；必须为 2 的幂 |
| **格式** | PNG，输出到 `build/<id>/` |
| **Normal** | RGB 三通道切线空间法线，**线性数据**；Blender 里图像色彩空间设为 `Non-Color`（Unity 端设为 `Normal Map`，见 [41](41-stage-04-pbr.md)） |
| **AO / Curvature / Roughness / Metallic** | 灰度 / 单通道信息，按 **Non-Color**（线性）处理，不做 sRGB 转换 |
| **Basecolor** | sRGB（颜色）通道 |
| **Padding / Dilation** | 烘焙时设置 Margin（如 16–32 px）做边缘扩张，避免 Unity 里 mipmap 下采样出现黑边 |

> 通道打包（把 AO / Roughness / Metallic 塞进一张图的 R/G/B）不做强制要求；本仓库默认**一张图一个用途、一个后缀**，便于 `assetctl validate` 逐张检查与 Unity 自动关联。

## 4. 命名规范

烘焙产物落到 `build/<id>/`，统一使用 SPEC.md §11 的后缀（详见 [70](70-naming-and-dod.md)）：

| 文件 | 后缀 | 内容 |
|------|------|------|
| `<id>_normal.png` | `_normal` | 切线空间法线 |
| `<id>_ao.png` | `_ao` | 环境光遮蔽（可选） |
| `<id>_basecolor.png` | `_basecolor` | 基础色 / Albedo（阶段 4 整理） |
| `<id>_roughness.png` | `_roughness` | 粗糙度（阶段 4 整理） |
| `<id>_metallic.png` | `_metallic` | 金属度（阶段 4 整理） |
| 其它烘焙中间图 | 如 `_curvature` | 曲率 / 遮罩，供阶段 4 生成 PBR 用，可不随 FBX 交付 |

命名必须让 `assetctl validate --expect-textures`（[80](80-cli-reference.md)）能逐张核对。

## 5. Cycles vs EEVEE

Blender 5.2 的烘焙主要在 **Cycles** 下做；EEVEE 用于实时预览（阶段 6）。

| 引擎 | 烘焙适用性 | 说明 |
|------|-----------|------|
| **Cycles** | **推荐 / 默认** | `bpy.ops.object.bake` 的完整支持：Normal / AO / Diffuse / Glossy / 高模→低模（Selected to Active）+ Ray Distance / Cage；结果确定、可进 CI |
| **EEVEE** | 预览为主 | 实时光栅化，速度快，适合阶段 6 出 `preview.png` 与人工看效果；不依赖它做正式烘焙的精度 |

选择理由：
- 烘焙是**离线、一次性的精度活**，Cycles 的光线追踪对 AO / 法线投射更准确，尤其 Selected-to-Active 高模→低模。
- EEVEE 是**实时、交互式**的，适合 TA 在视口里快速核对材质与光照，对应阶段 6 的预览图。
- 两者都在 Blender 5.2 内置，无需额外渲染器；脚本用 `--factory-startup` 保证引擎设置不被用户偏好污染。

## 6. 对齐 ACT 的 Palette Bake

SPEC.md §10 把 ACT 的 **Palette Bake** 映射到阶段 3/4：

| ACT 能力 | 本阶段落点 |
|----------|-----------|
| **Palette Bake** | 把低多边形 / 手绘风格的多色块材质「烤」成一张调色板贴图（Palette Texture），用少量颜色 + UV 岛对应色块，替代逐像素高模贴图 |
| Material/Texture Tools（清理重复材质、删未用材质） | 阶段 3/4 共用：烘焙前合并重复材质、删除未引用材质，避免烘焙出多余图 |
| Low-Poly Art workflow | 阶段 2 面数预算 + 阶段 4 Palette 材质；Palette Bake 是这套风格的贴图侧实现 |

低多边形资产可跳过「高模→低模」的 Normal 烘焙，直接用 **Palette Bake** 产出 Basecolor 调色板图 + 统一 Roughness/Metallic 常量值，减少贴图数量与包体。ACT 仅作行为参考，**不 vendoring 源码**（[60](60-act-integration.md)）。

## 7. 常见坑

| 现象 | 原因 / 处理 |
|------|-------------|
| 法线图紫一块 / 反了 | 色彩空间没设 `Non-Color`；或 Unity 未把贴图设为 `Normal Map`、切线/手性（OpenGL vs DirectX）不一致 |
| 烘焙出黑边 / 串色 | UV Island Margin / 烘焙 Margin 不足，阶段 2 先加大岛间距 |
| 高模细节没投上 | High/Low 未对齐、Ray Distance / Cage 太小或 Selected-to-Active 没勾 |
| 贴图糊 | `budget.maxTextureSize` 偏低或参考图分辨率不足 |

## 8. 产物与检查清单

| 产物 | 位置 |
|------|------|
| `work/<id>/03-bake.blend` | 阶段 4 的 `--in` |
| `build/<id>/<id>_normal.png` | 法线图 |
| `build/<id>/<id>_ao.png` | AO（可选） |
| 曲率 / 遮罩 / Palette 中间图 | `build/<id>/`，按后缀命名 |

- [ ] Normal / AO / 曲率 / 高模→低模按需烘焙完成
- [ ] 尺寸 ≤ `budget.maxTextureSize`，2 的幂
- [ ] Normal / AO / Curvature 按 Non-Color（线性）处理
- [ ] 命名符合 `_normal` / `_ao` / `_basecolor` / `_roughness` / `_metallic`
- [ ] 用 Cycles 做正式烘焙、EEVEE 仅预览
- [ ] 重复 / 未用材质已清理，Palette Bake 适用低多边形资产

下一步：[41-stage-04-pbr.md](41-stage-04-pbr.md)（PBR 材质）。
