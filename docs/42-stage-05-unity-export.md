# 42 · 阶段 ⑤ Unity 导出 / 同步（unity-export）

> 阶段 id：`unity-export`。实现：`scripts/blender/stage_05_unity_export.py`（Blender **5.2** / `bpy`）**+ Go 1.27 同步/校验**。
> 输入 `work/<id>/04-pbr.blend` → 输出 `build/<id>/<id>.fbx` **+ PBR 贴图**。契约见 [SPEC.md §5/§9](../SPEC.md)。
> **FBX 导出参数表严格照抄 SPEC.md §9，不得改默认值。**

## 1. 阶段职责

把阶段 4 的 PBR `.blend` 导出成 Unity 直接可用的 FBX + 同目录 PBR 贴图，并由 Go 侧同步资产、写 `manifest.json`、跑校验。人形角色在此阶段之后额外跑 `rig_check.py`（SPEC.md §5 门禁）。

| 项 | 值 |
|----|----|
| `--in` | `work/<id>/04-pbr.blend` |
| `--out` | `build/<id>/<id>.fbx` + PBR 贴图 |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
| 人形门禁 | 阶段 5 之后，若 `humanoid: true`，额外跑 `scripts/blender/rig_check.py` |
| 运行 | `blender --background --factory-startup --python scripts/blender/stage_05_unity_export.py -- --asset <id> --config <asset.json> --in ... --out ...` |

## 2. FBX 导出参数表（严格照 SPEC.md §9）

`bpy.ops.export_scene.fbx` 的参数必须与 `asset.json.axes`（`-Z / Y / 1.0`）及下列默认值一致：

| 参数 | 值 | 作用 |
|------|----|------|
| `axis_forward` | `"-Z"` | Blender → Unity 的前向轴（对齐 `axes.forward`） |
| `axis_up` | `"Y"` | Unity 的上轴（对齐 `axes.up`） |
| `global_scale` | `1.0` | 缩放（对齐 `axes.scale`，ACT 的 "Scale for models in Unity is 1"） |
| `apply_unit_scale` | `True` | 应用单位缩放，避免 Unity 里 100× / 0.01× 的尺寸异常 |
| `add_leaf_bones` | `False` | 不生成多余末端骨骼，保持 Humanoid 骨骼层级干净 |
| `bake_anim` | `True (humanoid)` | 人形角色导出动画；非人形/无动画按实际关闭 |
| `path_mode` | `"COPY"` | 把贴图复制到 FBX 同目录（配合 `embed_textures` 取舍），Unity 才能自动关联 |

> 这 7 项是唯一契约。Unity Import Settings 里勾选 **Bake Axis Conversion**（见 §4）与这套导出值配套；任何一项改动都必须先改 [SPEC.md §9](../SPEC.md)。

## 3. 坐标系说明

| 引擎 | 坐标系 | 说明 |
|------|--------|------|
| Blender 5.2 | 右手系，**Z-up**，前向 -Y | 建模/脚本的原生空间 |
| Unity | 左手系，**Y-up**，**Z-forward** | 目标运行时空间 |

导出时用 `axis_forward = "-Z"` / `axis_up = "Y"` 做轴转换，把 Blender 的 Z-up 转到 Unity 的 Y-up；`global_scale = 1.0` + `apply_unit_scale = True` 保证 1 Blender 单位 = 1 Unity 单位。`asset.json.axes` 固定为 `{ "forward": "-Z", "up": "Y", "scale": 1.0 }`，Go `config.Validate()` 会拒绝其它值。

Unity 侧不要再叠加额外旋转/缩放：配合勾选 **Bake Axis Conversion**，让 Unity 在导入时把轴转换「烤」进网格，动画与 Humanoid Avatar 才不会错位。脚本默认已设好，一般不要改；若 Unity 中模型朝向错误（面朝 -Z 而非 +Z），优先回查这里。

## 4. Unity Import Settings

选中 `build/<id>/<id>.fbx` 拖入 Unity 后，在 Inspector 配置：

### 4.1 Model 页
- **Scale Factor**：`1`
- **Convert Units**：按需（脚本已按 `1.0` 处理，一般关闭）
- **Bake Axis Conversion**：**勾选**（关键，见 §6）
- **Read/Write Enabled**：需要脚本改网格时开启

### 4.2 Rig 页
- **Animation Type**：人形选 **`Humanoid`**，道具选 `None`
- **Avatar Definition**：`Create From This Model`
- 点 **Configure** 检查骨骼映射，**无红色缺失项**（DoD 之一，见 [70](70-naming-and-dod.md)）

### 4.3 Materials 页
- 勾选 **Import Materials**，或用项目统一的 **Universal Render Pipeline/Lit** 材质覆盖（[41](41-stage-04-pbr.md)）。
- 五张图按后缀关联：Base Map / Normal Map / Metallic / Smoothness / Occlusion；`_normal` 的 Texture Type 设为 `Normal Map`。

### 4.4 Animation 页（clip 设置）
- 每个 clip 设名称、起止帧、**Loop Time**（`idle` / `walk` / `run` 勾选循环；`attack` / `hurt` / `death` 不勾）。
- clip 命名 `hero-knight-idle` 等（[70](70-naming-and-dod.md)）；完整清单见 [50](50-humanoid-animation.md)。

## 5. 为什么不用 ACT 的 "Unity (Pre-Rotated)" 档

ACT 提供两个导出档：**"Unity"** 与 **"Unity (Pre-Rotated)"**。本仓库选前者，并在 Unity 勾 **Bake Axis Conversion**：

| 档 | 做法 | 动画支持 | 我们的选择 |
|----|------|----------|-----------|
| **"Unity"** | 导出时按 `-Z forward / Y up / scale 1.0`，把轴转换交给 Unity 的 **Bake Axis Conversion** 在导入时处理 | **支持动画 / Humanoid** | ✅ 采用，对齐 SPEC.md §9 |
| **"Unity (Pre-Rotated)"** | 在 Blender 侧预先把网格/骨架旋转好再导出 | **不支持动画** | ❌ 不采用 |

原因（SPEC.md §9 明确）：**"Unity (Pre-Rotated)" 不支持动画**。人形角色必须带 clip 与 Humanoid Avatar，预旋转会破坏动画曲线与骨骼映射，导致 Unity 里 T-pose / 骨骼错位。因此统一用 "Unity" 档 + Bake Axis Conversion。详见 [60-act-integration.md](60-act-integration.md)。

## 6. 同步与 `manifest.json`

Go 1.27 runner 在阶段 5 做「同步」：把 Blender 导出的 `<id>.fbx` 与 PBR 贴图放置/更新到 `build/<id>/`，并写 **`build/<id>/manifest.json`** 记录各阶段状态与 DoD 结果。

`assetctl run` 依次执行六阶段、写 `manifest.json`；任一阶段失败即停止并记录状态，便于 `--from` 断点续跑（[80](80-cli-reference.md)）。`manifest.json` 结构示例：

```json
{
  "asset": "hero-knight",
  "strategy": "blender",
  "humanoid": true,
  "stages": {
    "modeling":      { "status": "ok", "out": "work/hero-knight/01-modeling.blend" },
    "topology-uv":   { "status": "ok", "out": "work/hero-knight/02-topology-uv.blend" },
    "bake":          { "status": "ok", "out": "work/hero-knight/03-bake.blend" },
    "pbr":           { "status": "ok", "out": "work/hero-knight/04-pbr.blend" },
    "unity-export":  { "status": "ok", "out": "build/hero-knight/hero-knight.fbx" },
    "render":        { "status": "ok", "out": "build/hero-knight/preview.png" }
  },
  "dod": {
    "validate": "pass",
    "rigCheck": "pass"
  }
}
```

> `manifest.json` 是机器可读的阶段状态 + DoD 结果（SPEC.md §6）；人可读的资产清单示例另见 `examples/asset-manifest.example.md`。

## 7. Prefab / Animator

交付到 Unity 后的落地步骤：

1. 把 `build/<id>/` 整个目录放进 `Assets/...`（如 `Assets/Characters/Hero/`）。
2. 把模型拖进场景 → 调整 Transform（面朝 +Z、脚底贴地）。
3. 挂上 **Animator**，绑定 **Animator Controller**：拖入 `idle` 作默认状态，`walk` / `run` / `attack` 按需加 Transitions（[50](50-humanoid-animation.md)）。
4. 拖回 Project 存为 **Prefab**。
5. 验收：无粉材质、无 T-pose、Avatar 映射无红项、idle 正常循环（[70](70-naming-and-dod.md)）。

## 8. 产物与检查清单

| 产物 | 位置 |
|------|------|
| `<id>.fbx` | `build/<id>/` |
| 五张 PBR 贴图 | `build/<id>/<id>_*.png`（与 FBX 同目录） |
| `manifest.json` | `build/<id>/manifest.json` |
| rig 校验结果 | `humanoid: true` 时由 `rig_check.py` 产出 |

- [ ] FBX 参数严格等于 SPEC.md §9 的 7 项
- [ ] 坐标系 `-Z forward / Y up / scale 1.0`，Unity 勾 **Bake Axis Conversion**
- [ ] Rig → Humanoid、Avatar Configure 无红项
- [ ] Materials 用 URP/Lit、`_normal` 为 Normal Map
- [ ] Animation clip 循环标记正确
- [ ] 不用 ACT "Unity (Pre-Rotated)"（不支持动画）
- [ ] `manifest.json` 记录阶段状态与 DoD
- [ ] `humanoid: true` 已跑 `rig_check.py`

下一步：[43-stage-06-render-composite.md](43-stage-06-render-composite.md)（渲染合成）。
