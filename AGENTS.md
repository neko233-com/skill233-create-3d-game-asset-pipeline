# AGENTS.md

本仓库面向「用 Tripo + 本机 Blender + Python 制作 Unity 3D 游戏资产」的流水线。
任何在本仓库工作的 Agent / 贡献者都必须遵守以下事实与约定。

## 工具链版本（硬性约束）

- **Blender：`5.2`**（本机安装版本）。所有 Blender 脚本以 `bpy` / Blender 5.2 为基准编写与验证。
  - 无头调用示例：`blender --background --python scripts/blender/tripo_to_unity.py -- --input in.glb --output out.fbx`
  - 禁止在文档/脚本中假设其它 Blender 大版本；升级版本必须先改本文件再改脚本。
- **Tripo：`https://studio.tripo3d.com/`**（浏览器端，图生 3D / 文生 3D / 自动绑定与动画）。
  - 所有 Tripo 相关说明、链接、脚本注释统一使用该 URL，不得写成旧域名或镜像。
- **Python**：本机 `3.14.3`。纯工具脚本（图片预处理 / 校验）用标准库 + Pillow，不引入重依赖。
- **目标引擎**：Unity（URP 优先），资产以 FBX + PBR 贴图交付。

## 仓库约定

- 语言：文档用中文；代码标识符、文件名、CLI 参数用英文。
- 命名：资产 ID 全小写连字符 `hero-knight-idle`；目录按 `characters/`、`props/`、`animations/` 分类。
- 坐标系：Blender 导出 Unity 用 `axis-forward=-Z`、`axis-up=Y`、缩放 `1.0`（见脚本默认值）。
- 人形角色：骨骼命名对齐 Unity Humanoid（见 `docs/04-humanoid-animation.md`），导出前必须过 rig 校验脚本。
- 提交：Conventional Commits（`feat:`、`fix:`、`docs:`、`chore:`）。

## 目录职责

| 路径 | 作用 |
|------|------|
| `SKILL.md` | 可复用 Skill 入口，供其它项目引用本流水线 |
| `docs/` | 分阶段流水线文档（Tripo → Blender → Python → 动画 → Unity） |
| `scripts/blender/` | Blender 内运行的 `bpy` 脚本（导入、清理、绑定校验、导出） |
| `scripts/python/` | Blender 之外运行的纯 Python 工具（参考图预处理、资产校验） |
| `examples/` | 配置与清单示例 |

## 成功标准

一个资产「完成」意味着：Tripo 产出模型 → Blender 清理/减面/重定向 → Python 校验通过 →
Unity 导入后贴图/骨骼/动画均可用，且 `docs/` 中对应清单项已勾选。
