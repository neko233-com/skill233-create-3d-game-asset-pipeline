# 41 · 阶段 ④ PBR 材质（pbr）

> 阶段 id：`pbr`。实现：`scripts/blender/stage_04_pbr.py`（Blender **5.2** 内置 Python / `bpy`）。
> 输入 `work/<id>/03-bake.blend` → 输出 `work/<id>/04-pbr.blend`。契约见 [SPEC.md §4/§5/§10](../SPEC.md)。

## 1. 阶段职责

把阶段 3 烘焙出的贴图、以及 `asset.json.textures` 声明的通道，整理成一套**命名规范、通道正确、可被 Unity URP/Lit 直接消费**的 PBR 材质。横切的人形角色绑定 / 动画也在此阶段与阶段 5 贯穿（见 [50](50-humanoid-animation.md)）。

| 项 | 值 |
|----|----|
| `--in` | `work/<id>/03-bake.blend` |
| `--out` | `work/<id>/04-pbr.blend` |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
| 运行 | `blender --background --factory-startup --python scripts/blender/stage_04_pbr.py -- --asset <id> --config <asset.json> --in ... --out ...` |

阶段 5 导出时，`path_mode = "COPY"` 会把这些贴图随 FBX 一并复制到 `build/<id>/`，与 `<id>.fbx` 同目录交付 Unity。`assetctl run` 在真实 Blender 阶段结束后会校验本阶段 `--out` 文件确实存在（静默失败防护）；CI / `go test ./...` 则由 `ScriptLauncher` 运行 `testing/fake_blender.py` 模拟端到端，无需安装 Blender 5.2。

## 2. 五张 PBR 贴图

`asset.json.textures` 默认 `["basecolor", "normal", "roughness", "metallic"]`；`_ao` 为可选第五张。

| 贴图 | 后缀 | 内容 | 色彩空间 | Unity URP/Lit 槽位 |
|------|------|------|----------|--------------------|
| **Basecolor / Albedo** | `_basecolor` | 固有色，不含光影信息 | **sRGB** | Base Map |
| **Normal** | `_normal` | 切线空间法线，凹凸细节 | **Non-Color（线性）** | Normal Map（Texture Type = `Normal Map`） |
| **Roughness** | `_roughness` | 表面粗糙 / 光滑度（灰度） | **Non-Color（线性）** | Smoothness 的反相 / Metallic 的 Alpha（见下） |
| **Metallic** | `_metallic` | 金属 / 非金属（灰度，0/1 为主） | **Non-Color（线性）** | Metallic（或 Metallic-Smoothness 合图） |
| **AO** | `_ao` | 环境光遮蔽（可选） | **Non-Color（线性）** | Occlusion |

> 命名规范与 `assetctl validate --expect-textures` 的逐张核对见 [70](70-naming-and-dod.md) 与 [80](80-cli-reference.md)。

## 3. Principled BSDF → Unity URP/Lit 对应关系

Blender 5.2 的 **Principled BSDF** 是 PBR 材质的统一入口；Unity URP 用 **Universal Render Pipeline/Lit**。两者按金属度-粗糙度工作流对应：

| Principled BSDF 输入（Blender 5.2） | 本仓库贴图 | URP/Lit 对应 | 说明 |
|------------------------------------|-----------|--------------|------|
| `Base Color` | `_basecolor` | **Base Map** | sRGB 颜色，不带烘焙光影 |
| `Normal`（经 Normal Map 节点） | `_normal` | **Normal Map** | 切线空间；Unity 里必须设 Texture Type = `Normal Map` |
| `Roughness` | `_roughness` | **Smoothness = 1 − Roughness** | URP/Lit 用 Smoothness（光滑度），与 Roughness 互为反相；若用 Metallic-Smoothness 合图则放 Alpha |
| `Metallic` | `_metallic` | **Metallic** | 0 = 非金属，1 = 金属；也可与 Smoothness 合并为一张 `Metallic-Smoothness` 图（R=Metallic, A=Smoothness） |
| `Ambient Occlusion` | `_ao` | **Occlusion** | 可选；只压暗间接光，不影响直接光 |
| `Emission` | — | **Emission** | 自发光，按需（本流水线默认不产出） |

要点：

- Blender 里用 **Image Texture 节点**连到 Principled BSDF 对应输入；`_normal` 必须经 **Normal Map 节点**（不能直接连 Base Color）。
- 五张图**一张一用途一个后缀**，不强制做通道打包；若 TA 想省纹理数，可把 Metallic 与 Smoothness 打包，但必须同步改 `assetctl validate` 的期望与命名，避免门禁误判。
- Unity 导入后若勾选 `Import Materials`，会把 FBX 内材质转成对应槽位；更稳妥的做法是用项目里统一的 URP/Lit 材质覆盖（见 [42](42-stage-05-unity-export.md)）。

## 4. 法线图 Non-Color（关键）

法线图、AO、Roughness、Metallic 都是**数据贴图**，不是颜色：

| 端 | 设置 |
|----|------|
| Blender 5.2 | 对应 Image Texture 节点的图像 **Color Space = `Non-Color`**（`img.colorspace_settings.name = "Non-Color"`） |
| Unity | 该贴图 Texture Type 设为 **`Normal Map`**（法线图）/ 关闭 **sRGB (Color Texture)**（数据图） |

若把法线图当 sRGB 颜色读，法线向量会被 gamma 变换扭曲，Unity 里表现为**光照错误、表面凹凸发紫/发亮**；若 Unity 未把 `_normal` 标成 `Normal Map`，则会出现「材质有图但没凹凸」或整体偏紫。这是粉材质/怪光照之外的常见坑。

## 5. Palette / 低多边形材质

对齐 SPEC.md §10 的 **Low-Poly Art workflow** 与 ACT 的 **Palette Bake / Material Tools**：

- 低多边形资产用**调色板贴图 + 少量纯色材质槽**：阶段 3 的 Palette Bake 产出调色板图，阶段 4 把它连到 Principled BSDF 的 `Base Color`。
- `Roughness` / `Metallic` 用**常量值**（如 Roughness `0.8`、Metallic `0.0`）而非逐像素贴图，减少贴图数量与包体。
- 配合阶段 2 的面数预算（远景 <500、道具 500–5k）形成统一风格。
- 面数、贴图尺寸都由 `asset.json.budget` 驱动，TA 改配置即可切换「写实高模烘焙」与「低多边形 Palette」两套产出。

## 6. 清理重复材质（ACT Material Tools）

SPEC.md §10 把 ACT 的 **Material/Texture Tools（清理重复材质、删未用材质）** 映射到阶段 3/4：

| ACT 能力 | 本阶段落点 |
|----------|-----------|
| 清理重复材质 | 合并同名 / 同节点参数的材质，避免 FBX 带出 `Material.001`、`Material.002` 这类冗余 |
| 删除未用材质 | 去掉网格未引用的材质槽，减少 Unity 里的 SubMesh / draw call |
| Palette Bake | 阶段 3 烘焙、阶段 4 连线（见 §5） |

清理后应满足：一个网格只保留实际用到的材质，材质名与贴图后缀一致，Unity 导入后**无粉材质**（粉 = 材质或 Shader 丢失，URP 下常见原因是导入了 Built-in 的 Standard Shader 材质）。ACT 仅作行为参考，**不 vendoring 源码**（[60](60-act-integration.md)）。

## 7. 产物与检查清单

| 项 | 期望 |
|----|------|
| `work/<id>/04-pbr.blend` | 阶段 5 的 `--in` |
| `build/<id>/<id>_basecolor.png` 等 | 五张图按后缀命名（阶段 5 随 FBX 交付） |
| 材质节点 | Principled BSDF + 各 Image Texture 连到正确输入 |
| 色彩空间 | Basecolor = sRGB；Normal/AO/Roughness/Metallic = Non-Color |
| 重复材质 | 已清理、无未用材质槽 |

- [ ] 五张图齐全（或 `asset.json.textures` 声明的子集）
- [ ] Principled BSDF → URP/Lit 槽位对应正确
- [ ] `_normal` 在 Blender 与 Unity 两侧都按 Non-Color / Normal Map 处理
- [ ] 低多边形资产用 Palette + 常量 Roughness/Metallic
- [ ] 重复 / 未用材质已清理
- [ ] 横切：人形角色在此阶段/阶段 5 完成绑定与 clip 准备（[50](50-humanoid-animation.md)）

下一步：[42-stage-05-unity-export.md](42-stage-05-unity-export.md)（Unity 导出 / 同步）。
