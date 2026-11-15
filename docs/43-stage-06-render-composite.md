# 43 · 阶段 ⑥ 渲染合成（render / render-composite）

> 阶段 id：`render`（SPEC.md §4 表中名称「渲染合成」）。实现：`scripts/blender/stage_06_render.py`（Blender **5.2** / `bpy`）。
> 输入 `build/<id>/<id>.fbx` → 输出 `build/<id>/preview.png`。契约见 [SPEC.md §5](../SPEC.md)。

## 1. 阶段职责

用 Blender 5.2 给最终交付的 FBX 拍一张**标准化预览图**，供 QA、缩略图、商店图使用。这是六阶段的最后一环，也是 `assetctl validate` 之前最直观的人工验收物。

| 项 | 值 |
|----|----|
| `--in` | `build/<id>/<id>.fbx`（阶段 5 的最终 FBX） |
| `--out` | `build/<id>/preview.png` |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
| 运行 | `blender --background --factory-startup --python scripts/blender/stage_06_render.py -- --asset <id> --config <asset.json> --in ... --out ...` |

> 注意：本阶段读的是**已导出的 FBX**，而不是 `work/<id>/04-pbr.blend`。这样预览图反映的是「Unity 实际拿到的那份几何 + 材质关联」，避免工作文件与交付文件不一致时 QA 误判。

## 2. 三点光照（Three-Point Lighting）

标准三点布光，保证预览图能看清形体、材质与凹凸：

| 灯 | 位置 / 作用 | 建议 |
|----|-------------|------|
| **Key（主光）** | 相机侧前方 30–45°，塑造主体体积与法线凹凸 | 面积光 / 太阳光，最强 |
| **Fill（补光）** | 主光对侧，提亮暗部、避免死黑 | 强度约 Key 的 1/3–1/2，柔和 |
| **Rim / Back（轮廓光）** | 主体后方，勾边、把角色从背景里分离 | 强度适中，突出肩/头轮廓 |

要点：
- 用**中性白光**，避免给 Basecolor 染色，预览图才能真实反映贴图。
- 低多边形 / Palette 资产同样吃三点光，轮廓光对硬边风格尤其有效。
- 灯光与相机都写在 `stage_06_render.py` 里，脚本用 `--factory-startup` 保证不依赖用户场景偏好。

## 3. 相机

- 用一台固定相机，正交（Orthographic）或长焦透视，避免广角畸变影响形体判断。
- 构图：主体居中、留白适度，人形角色取全身（头脚都入画），道具取能展示主要面的 3/4 视角。
- 朝向与 Unity 一致：模型面朝 +Z（见 [42](42-stage-05-unity-export.md) 的坐标系），相机从 +Z 侧略偏上看，避免只拍到背面。
- 分辨率：正方形或 4:3，够做缩略图/商店图即可（如 1024×1024）；由脚本固定，保证不同资产预览图可横向比较。

## 4. EEVEE 预览图

Blender 5.2 的 **EEVEE**（实时光栅化引擎）用于本阶段的预览渲染：

| 维度 | 选择 |
|------|------|
| 烘焙（阶段 3） | **Cycles**：离线精度，高模→低模 / AO / 法线投射 |
| 预览（阶段 6） | **EEVEE**：实时、快，适合无头批量出 `preview.png` 与人工看效果 |

理由见 [40](40-stage-03-bake.md)：Cycles 管精度、EEVEE 管速度。EEVEE 出图够用于 QA / 缩略图 / 商店图，且能与 `blender --background --factory-startup --python` 的无头批量流程配合。开启软阴影 / AO / 合理的色彩管理（Filmic / AgX，按项目统一）避免过曝。

## 5. 合成（世界 / 环境光）

用 Blender 5.2 的**世界（World）与合成（Compositor）**把布光补足、成图：

| 合成要素 | 作用 |
|----------|------|
| **World / 环境光** | 给一个中性灰或 HDRI 环境，提供全局间接光；避免暗部纯黑、金属/粗糙度看不出来 |
| **环境光遮蔽（AO）** | 叠加缝隙遮蔽，增强接地感；与阶段 3 烘焙的 `_ao` 贴图互补（预览里再强化一点） |
| **背景 / 接地** | 纯色或渐变背景、简单地面 + 接触阴影，让主体不「飘」 |
| **色彩管理 / 调色** | 统一 View Transform，轻度对比/锐化，输出 `preview.png` |
| **Alpha / 透明通道** | 商店图 / 缩略图可按需输出透明背景，便于叠到任意底图上 |

合成节点写在脚本里，所有资产走同一套世界/调色，保证预览图风格一致、可批量比较。

## 6. Turntable 首帧

可给相机（或模型）做一圈 **Turntable（转台）**动画，本阶段取**首帧**作为静态 `preview.png`：

- 首帧对应上文「3/4 正面、能展示主要面」的机位，作为对外展示的代表角度。
- 保留 turntable 相机路径便于以后扩展成 360° 序列 / GIF / 短视频，但本阶段产物契约只有一张 `preview.png`。
- 人形角色可让首帧略微偏向展示武器/面部的一侧。

## 7. `preview.png` 的用途

| 用途 | 说明 |
|------|------|
| **QA** | 主程 / TA 不开 Unity 就能快速核对：形体、朝向、贴图是否丢失（粉材质）、法线是否反了、有没有破面 |
| **缩略图** | 资产库 / 内容浏览器 / 版本对比的封面图 |
| **商店图** | 商店页、素材市场、外包验收附图的底图（配合透明背景/合成） |

`preview.png` 落在 `build/<id>/`，与 `<id>.fbx`、五张 PBR 贴图、`manifest.json` 同目录，构成完整交付包。

## 8. 与 Unity 内最终效果的关系（重要）

**`preview.png` 是参考，不是等价替代。** 它用 Blender 5.2 / EEVEE 渲染，而 Unity 用 URP（不同渲染管线、光照模型、色彩管理、Shader、后处理）。因此：

| 维度 | Blender `preview.png` | Unity 内最终效果 |
|------|-----------------------|------------------|
| 渲染器 | EEVEE | URP/Lit |
| 光照 | 脚本固定三点光 + World | 场景真实光源 / 光照烘焙 / 探针 |
| 材质 | Principled BSDF | Universal Render Pipeline/Lit |
| 后处理 | Blender 合成节点 | URP Volume / Tonemapping |

一致的部分：几何、UV、PBR 贴图、坐标系、命名——这些由六阶段契约保证，所以预览图能可靠反映「资产本身对不对」。
不一致的部分：灯光、色调、阴影、后处理——这些属于 Unity 场景，TA 必须在 Unity 里用真实场景光再验收一次，不能只看 `preview.png` 就判定「上线效果 OK」。

> DoD 仍以 [70](70-naming-and-dod.md) 为准：`assetctl validate` 通过 + Unity 中无粉材质、无 T-pose、Avatar 映射无红项。预览图是帮助快速发现问题的工具，不是验收的终点。

## 9. 产物与检查清单

| 产物 | 位置 |
|------|------|
| `preview.png` | `build/<id>/preview.png` |

- [ ] 输入是阶段 5 的最终 FBX，不是工作 `.blend`
- [ ] 三点光照（Key / Fill / Rim）+ 中性 World 环境光
- [ ] 相机机位能展示主体主要面、面朝 +Z
- [ ] 用 EEVEE 出预览图（Cycles 留给阶段 3 烘焙）
- [ ] 合成含 World / AO / 背景 / 统一色彩管理
- [ ] turntable 首帧作为代表角度
- [ ] `preview.png` 与 FBX / 贴图 / `manifest.json` 同目录交付
- [ ] 文档/评审中明确：预览图 ≠ Unity 内最终效果，最终以 Unity 场景验收
