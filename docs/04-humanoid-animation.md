# 04 · 人形角色与动画

## 两条绑定路线

1. **Tripo 自动绑定**（推荐起步）：在 https://studio.tripo3d.com/ 生成人形后使用自动绑定，导出带骨骼模型。优点快；缺点骨骼命名可能与 Unity 不匹配，需要重定向。
2. **Blender 手动绑定**：导出静态 GLB → Blender 里绑定 → 按 Unity Humanoid 命名骨骼。适合对骨骼结构有精确要求。

## Unity Humanoid 骨骼命名

Unity 的 Humanoid Avatar 需要关键骨骼映射。命名时至少包含这些（大小写不敏感，但建议统一）：

```
Hips
 ├─ Spine
 │   └─ Chest
 │       ├─ Neck → Head
 │       ├─ LeftShoulder → LeftUpperArm → LeftLowerArm → LeftHand
 │       └─ RightShoulder → RightUpperArm → RightLowerArm → RightHand
 └─ LeftUpperLeg → LeftLowerLeg → LeftFoot → LeftToes
    RightUpperLeg → RightLowerLeg → RightFoot → RightToes
```

关键点：
- 以 `Hips` 为根，层级连续。
- 左右用 `Left` / `Right` 前缀，不要 `L_` / `R_`（Unity 自动映射更稳）。
- 手指可选，但若要手部动画，按 `LeftThumb/Index/Middle/Ring/Pinky...` 命名。

## 骨骼校验：`rig_check.py`

导出前在 Blender 无头跑：

```bash
blender --background --python scripts/blender/rig_check.py -- --input build/hero/hero.fbx
```

检查：
- 存在 Armature
- 关键骨骼（`Hips`, `Spine`, `Head`, 左右四肢）齐全
- 层级连续、无重复命名
- 输出每个骨骼的世界坐标，便于核对

## 动画制作

- **Tripo 侧**：https://studio.tripo3d.com/ 的动画功能可生成基础动作（idle/walk/attack）。
- **Blender 侧**：手动 K 帧或用 mixamo 等外部动作，重定向到本角色的骨骼。
- **Unity 侧**：导入 FBX，在 Import Settings → Animation 里为每个 clip 设置循环/时长，配置 Animator Controller。

## 动画导出清单

| Clip | 循环 | 说明 |
|------|------|------|
| idle | 是 | 待机 |
| walk | 是 | 行走 |
| run | 是 | 跑步（可选） |
| attack | 否 | 攻击 |
| hurt | 否 | 受击 |
| death | 否 | 死亡 |

每个 clip 单独导出或在 FBX 内分段时间，命名 `hero-idle`、`hero-walk` 等。

## 常见坑

- **Unity 里 T-pose**：骨骼命名没对齐 Humanoid，或 Avatar 未配置。用 `rig_check.py` 先查。
- **脚滑**：动画位移与位移速度不匹配，在 Unity 里调 root motion 或动画曲线。
- **骨骼错位**：坐标系没统一，回到 02 文档检查 FBX 导出参数。
