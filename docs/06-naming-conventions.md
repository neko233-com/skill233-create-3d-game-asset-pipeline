# 06 · 命名规范

## 资产 ID

- 全小写，单词用连字符：`hero-knight`、`props-sword`、`npc-villager`。
- 动画 clip：`{asset-id}-{action}`，如 `hero-knight-idle`。

## 目录

```
raw/                        # Tripo 原始导出，保留
build/{asset-id}/           # Blender 导出、待导入 Unity
  {asset-id}.fbx
  {asset-id}_basecolor.png
  {asset-id}_normal.png
  {asset-id}_roughness.png
  {asset-id}_metallic.png
scripts/
  blender/
  python/
docs/
examples/
```

## 贴图后缀

| 后缀 | 贴图 |
|------|------|
| `_basecolor.png` | 基础色 / Albedo |
| `_normal.png` | 法线 |
| `_roughness.png` | 粗糙度 |
| `_metallic.png` | 金属度 |
| `_ao.png` | 环境光遮蔽（可选） |

## 骨骼

- 对齐 Unity Humanoid 命名（见 [04](04-humanoid-animation.md)）。
- 左右用 `Left` / `Right` 前缀。

## 提交信息

Conventional Commits：`feat:`、`fix:`、`docs:`、`chore:`、`refactor:`。
