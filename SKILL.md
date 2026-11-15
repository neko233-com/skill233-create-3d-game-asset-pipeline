---
name: create-3d-game-asset-pipeline
description: 用浏览器 Tripo(https://studio.tripo3d.com/) 图生3D、本机 Blender 5.2 与 Python 清洗/绑定，产出可直接导入 Unity 的人形角色与道具游戏资产（含动画）。当用户要把图片/概念图变成 Unity 3D 资产、做图生3D、Blender 批处理、Unity 人形骨骼与动画重定向时使用。
---

# Create 3D Game Asset Pipeline

把一张参考图变成 Unity 里可用的 3D 游戏资产的端到端流水线。

## 何时使用

- 用 Tripo 做图生 3D / 文生 3D，导出 GLB/OBJ/FBX。
- 用本机 Blender 5.2 做减面、清理、UV/贴图整理、骨骼校验、导出 Unity FBX。
- 纯 Python 做参考图预处理与交付前校验。
- 人形角色自动绑定与动画重定向到 Unity Humanoid。

## 工具链（硬性）

- Blender **5.2**（本机），无头脚本用 `blender --background --python ... -- --args`。
- Tripo **https://studio.tripo3d.com/**（浏览器端）。
- Python **3.14.3**（工具脚本），Unity 目标为 URP。

## 工作流（顺序执行）

1. **准备参考图** — 主体单一、正面、干净背景、正方形。用 `scripts/python/prepare_reference.py` 归一化。
2. **Tripo 生成** — 打开 https://studio.tripo3d.com/ ，选 Image-to-3D，上传参考图，生成后下载 GLB。详见 `docs/01-tripo-image-to-3d.md`。
3. **Blender 清洗** — 用 `scripts/blender/tripo_to_unity.py` 无头导入 GLB、清理、按 Unity 坐标系导出 FBX + 贴图。详见 `docs/02-blender-cleanup.md`。
4. **人形动画** — 需要角色动画时，先跑 `scripts/blender/rig_check.py` 校验骨骼，再按 `docs/04-humanoid-animation.md` 重定向，Tripo 的自动绑定/动画能力也在 https://studio.tripo3d.com/ 完成。
5. **校验交付** — `scripts/python/validate_assets.py` 检查命名、贴图、骨骼，按 `docs/05-unity-import.md` 导入 Unity。

## 脚本速查

```bash
# 1) 参考图归一化（正方形、中性背景）
python scripts/python/prepare_reference.py in.png out/ --size 1024

# 2) Blender 无头：Tripo GLB -> Unity FBX
blender --background --python scripts/blender/tripo_to_unity.py -- \
  --input raw/hero.glb --output build/hero/hero.fbx --scale 1.0

# 3) 骨骼校验
blender --background --python scripts/blender/rig_check.py -- --input build/hero/hero.fbx

# 4) 交付校验
python scripts/python/validate_assets.py build/hero --expect-humanoid
```

## 约定

- 资产 ID 小写连字符；坐标系 `-Z forward / Y up`；FBX + PBR 贴图交付。
- 任何版本/URL 变更先改 `AGENTS.md`。
