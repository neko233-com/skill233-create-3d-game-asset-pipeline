# 50 · 人形角色动画（横切关注点）

> 本文是**横切**文档：Humanoid 绑定、rig 校验、clip 导出贯穿阶段 4（PBR 材质）与阶段 5（Unity 导出/同步）。
> 契约见 [SPEC.md §4/§5/§11](../SPEC.md)：`humanoid: true` 时，阶段 5 之后额外跑 `scripts/blender/rig_check.py`。工具：Blender **5.2**、Tripo **https://studio.tripo3d.com/**。

## 1. 两条绑定路线

| 路线 | 做法 | 优点 | 缺点 / 注意 |
|------|------|------|-------------|
| **A. Tripo 自动绑定** | 在 https://studio.tripo3d.com/ 生成人形后使用**自动绑定（auto-rig）与动画**，导出带骨骼模型 | 快，适合起步 / 原型 | 骨骼命名可能与 Unity Humanoid 不完全一致，需重定向或改名 |
| **B. Blender 手动绑定** | 导出静态 GLB → 在 Blender 5.2 里绑定 → 按 Unity Humanoid 命名骨骼（脚本 `scripts/blender/humanoid_rig.py`） | 骨骼结构精确可控，Avatar 映射稳 | 需要人工/脚本投入 |

> v2 起 `tripo` 是**默认生产供应商**，因此路线 A（Tripo 自动绑定）与默认建模来源一致；`blender` 参数化建模为可选策略，对应路线 B 的手动绑定。

选择建议：
- 要快速出可动原型 → 路线 A（默认供应商），再用 `rig_check.py` 暴露命名差异并重定向。
- 要正式入库、多人共用 Avatar / 动画重定向 → 路线 B，骨骼一次对齐，后续所有 clip 复用。

> 两条路线都最终落到同一套骨骼命名（§2）与同一套 clip 清单（§4）。自动绑定能力本身在 https://studio.tripo3d.com/ 内完成；本仓库不内嵌第三方绑定器。

## 2. Unity Humanoid 骨骼命名表

Unity Humanoid Avatar 按**骨骼名 + 层级**做自动映射。命名规则（SPEC.md §11）：根为 `Hips`，左右用 **`Left` / `Right` 前缀**（不要 `L_` / `R_`，Unity 自动映射更稳）。

```
Hips
 ├─ Spine
 │   └─ Chest
 │       ├─ Neck → Head
 │       ├─ LeftShoulder → LeftUpperArm → LeftLowerArm → LeftHand
 │       └─ RightShoulder → RightUpperArm → RightLowerArm → RightHand
 ├─ LeftUpperLeg → LeftLowerLeg → LeftFoot → LeftToes
 └─ RightUpperLeg → RightLowerLeg → RightFoot → RightToes
```

| 部位 | 必填骨骼 | 可选 / 手指 |
|------|----------|-------------|
| 躯干根 | `Hips` | — |
| 脊柱 | `Spine`、`Chest`、`Neck`、`Head` | `UpperChest` |
| 左臂 | `LeftShoulder`、`LeftUpperArm`、`LeftLowerArm`、`LeftHand` | `LeftThumb` / `LeftIndex` / `LeftMiddle` / `LeftRing` / `LeftPinky` 各节 |
| 右臂 | `RightShoulder`、`RightUpperArm`、`RightLowerArm`、`RightHand` | `RightThumb` / `RightIndex` / `RightMiddle` / `RightRing` / `RightPinky` 各节 |
| 左腿 | `LeftUpperLeg`、`LeftLowerLeg`、`LeftFoot`、`LeftToes` | — |
| 右腿 | `RightUpperLeg`、`RightLowerLeg`、`RightFoot`、`RightToes` | — |

关键点：
- 以 `Hips` 为根，**层级连续**，中间不要断链（Unity 映射靠层级推断）。
- 大小写不敏感但建议统一为表中写法；左右前缀固定 `Left` / `Right`。
- 手指可选；若要手部动画 / 握持武器，按 `LeftThumb/Index/Middle/Ring/Pinky...` 命名。
- 导出 FBX 时 `add_leaf_bones = False`（[SPEC.md §9](../SPEC.md)），避免末端多余骨骼污染层级。

## 3. `rig_check.py` 用法

导出前在 Blender 5.2 无头跑（SPEC.md §5 把它作为阶段 5 之后的**人形门禁**）：

```bash
blender --background --factory-startup --python scripts/blender/rig_check.py -- \
  --input build/hero-knight/hero-knight.fbx
```

也可按阶段统一传参约定调用：

```
blender --background --factory-startup --python scripts/blender/rig_check.py -- \
  --asset hero-knight --config assets/hero-knight/asset.json \
  --in build/hero-knight/hero-knight.fbx
```

检查项：
- 存在 **Armature**；
- 关键骨骼（`Hips`、`Spine`、`Head`、左右四肢）齐全，符合 §2 命名；
- 层级连续、无重复命名、无 `L_` / `R_` 之类非标前缀；
- 输出每个骨骼的世界坐标，便于核对 T-pose / 骨骼错位。

退出码：`0` 通过；非 `0` 列出缺失/错位骨骼并写 stderr。`rig_check.py` 通过是 DoD 的前置条件（[70](70-naming-and-dod.md)）。

## 4. Clip 清单与循环标记

`asset.json.animations` 声明本资产带哪些 clip（SPEC.md §7 示例：`["idle", "walk", "attack"]`）。完整清单：

| Clip | 循环（Loop Time） | 说明 | `asset.json` 示例名 |
|------|-------------------|------|---------------------|
| `idle` | **是** | 待机，Animator 默认状态 | `hero-knight-idle` |
| `walk` | **是** | 行走 | `hero-knight-walk` |
| `run` | **是** | 跑步（可选） | `hero-knight-run` |
| `attack` | 否 | 攻击，单次播放 | `hero-knight-attack` |
| `hurt` | 否 | 受击 | `hero-knight-hurt` |
| `death` | 否 | 死亡 | `hero-knight-death` |

约定：
- clip 命名 `{asset-id}-{action}`，资产 ID 小写连字符（[70](70-naming-and-dod.md)）。
- 循环 clip 在 Unity Import Settings → Animation 页勾 **Loop Time / Loop Pose**；单次 clip 不勾。
- 导出 FBX 时 `bake_anim = True (humanoid)`（SPEC.md §9）；每个 clip 可单独导出成独立 FBX，或在同一个 FBX 内分段并在 Unity 里切分。
- 路线 A 的基础动作可在 https://studio.tripo3d.com/ 生成；路线 B 在 Blender 5.2 手动 K 帧 / 重定向外部动作（如 Mixamo）到本角色骨骼。

## 5. 常见坑

| 现象 | 原因 / 处理 |
|------|-------------|
| **Unity 里 T-pose** | 骨骼命名没对齐 Humanoid（用了 `L_`/`R_`、缺 `Hips`/`Chest`），或 **Avatar 未配置 / Configure 有红项**；先跑 `rig_check.py` |
| **脚滑（foot sliding）** | 动画位移与速度不匹配、根运动处理不当；在 Unity 里调 **Root Motion** 或动画曲线，对齐步幅与位移 |
| **骨骼错位** | 坐标系没统一 / 用了 ACT "Unity (Pre-Rotated)"（不支持动画）；回查 [42](42-stage-05-unity-export.md) 的 `-Z forward / Y up / scale 1.0` 与 **Bake Axis Conversion** |
| **Avatar 配置** | Rig → Animation Type 必须为 `Humanoid`、Avatar Definition `Create From This Model`；点 **Configure** 直到无红色缺失项（DoD 之一） |
| **clip 导入后没动画** | FBX `bake_anim` 没开、或源 GLB 未带骨骼动画；阶段 5 确认 `bake_anim = True (humanoid)` |
| **手指 / 武器握持错** | 手指骨骼未按 `LeftThumb/Index/...` 命名，或缺 `LeftHand` 层级；按 §2 补全 |

## 6. 与六阶段的衔接

| 阶段 | 人形动画相关动作 |
|------|------------------|
| ① modeling | Tripo 路线可在这里选自动绑定（[20](20-stage-01-modeling.md)） |
| ④ pbr | 绑定 / 材质槽整理、`animations` 声明生效（[41](41-stage-04-pbr.md)） |
| ⑤ unity-export | `bake_anim = True` 导出 FBX + clip；**阶段 5 之后跑 `rig_check.py`**（[42](42-stage-05-unity-export.md)） |
| Unity | Rig → Humanoid、Avatar Configure、Animator Controller / Prefab（[42](42-stage-05-unity-export.md)、[70](70-naming-and-dod.md)） |

- [ ] 选定绑定路线（Tripo 自动绑定 / Blender `humanoid_rig.py`）
- [ ] 骨骼命名符合 §2，`Left`/`Right` 前缀、`Hips` 为根、层级连续
- [ ] `rig_check.py` 通过
- [ ] clip 循环标记正确、命名 `{asset-id}-{action}`
- [ ] Unity Avatar 映射无红项、无 T-pose、idle 正常循环
