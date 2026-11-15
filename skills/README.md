# skills/ —— Blender Agent Skills 知识参考集

本目录收录 **28 个** Blender Agent Skills 文档，作为本仓库「游戏美术管线」的知识与规范参考，配合阶段三的 `assetctl skills` 使用。

## 来源与许可

- 全部 `SKILL.md` **原样照抄**自公开仓库 [EL4CTEO/blender-skills](https://github.com/EL4CTEO/blender-skills)（`main` 分支的 `skills/<name>/SKILL.md`），保留原始 YAML frontmatter 与正文，未改写、未翻译、未删减。
- 该仓库采用 **MIT 许可**，LICENSE 原文副本保存在 [`LICENSE-blender-skills.txt`](LICENSE-blender-skills.txt)（含 `Copyright (c) 2026 EL4CTEO` 版权声明），满足 MIT 合规要求。
- **仅作知识 / 规范参考**：本仓库**不复制、不 vendoring** 参考仓库的任何 Python 源码（`scripts/`、`assets/` 下的 `.py`）。`manifest.json` 的 `referenceScripts` 只是从参考仓库文件树提取的脚本文件名，用于标注知识出处，并非本仓库的可执行代码。

## 目录结构

| 文件 | 作用 |
|---|---|
| `manifest.json` | 全部 skill 的元数据索引：来源、许可、`blenderTarget`，以及每个 skill 的 `name` / `description` / `referenceScripts` / `goCommand` / `bpyScript` |
| `<name>/SKILL.md` | 照抄的原始 skill 文档（勿改动） |
| `<name>/ADAPTATION.md` | 中文适配说明：来源、Go CLI 与 bpy 入口、核心操作要点、产出契约、测试方式 |
| `LICENSE-blender-skills.txt` | 参考仓库 MIT LICENSE 原文 |

## 执行模型（本仓库）

参考仓库的 skill 是给通用 Agent 的知识文档；在本仓库里，**实际执行全部由自研代码完成**：

- **编排**：Go 1.27 CLI `assetctl skills`（零第三方依赖，仅标准库）。
- **执行**：Blender 5.2 内置 `bpy` 脚本，统一入口 `scripts/blender/skill_run.py --skill <id>`。
- 每个 skill 的命令映射见 `manifest.json` 的 `goCommand` / `bpyScript` 字段。
- 执行后产出契约文件 `build/<id>/contracts/skill-<id>.json`，并由 `testing/test_skills.py` 在 `mock_bpy` 下自动测试。

## 与 AGENTS.md 硬性约束的对齐

| 约束（[AGENTS.md](../AGENTS.md)） | 本目录如何对齐 |
|---|---|
| **Blender 5.2** | `manifest.json.blenderTarget = "5.2"`；所有 `bpy` 脚本以 5.2 内置 Python 为基准、必须能在 `--factory-startup` 下运行。参考文档中出现的 5.3 / 5.2 LTS 字样仅作知识参考，不作为本仓库版本依据 |
| **Go 1.27 零第三方依赖** | 编排全部走 `assetctl`（Go 1.27 标准库），不引入参考仓库的 Python 运行时或任何第三方包；系统 Python 仅开发辅助 |
| **Tripo 默认供应商** | `https://studio.tripo3d.com/` 仍是默认建模 / 绑定供应商（`assetctl init` 默认 `--strategy tripo`）；本目录 skills 只提供 Blender 侧加工知识，不改变策略选择 |
| **Unity URP** | 资产最终以 FBX + PBR 贴图交付 Unity（URP）；`blender-export-pipelines` 等 skill 的知识对齐 Unity 导出档，导入仍按 AGENTS.md 勾 **Bake Axis Conversion** |
| **语言分工** | 文档用中文（本 README、`ADAPTATION.md`），代码标识符、文件名、CLI 参数用英文 |

> 参考插件 ACT（GPL-3.0）与参考仓库（MIT）都只对齐行为与参数、不复制源码，避免许可证污染本仓库的 MIT 许可。

## 用法示例

`assetctl skills list` 读取 `skills/manifest.json` 输出 skill 索引：

```bash
# 列出全部 28 个 skill（id / name / 简要说明）
go run ./cmd/assetctl skills list

# 查看单个 skill 的详情与命令映射
go run ./cmd/assetctl skills list blender-character-rigging

# 运行某个 skill：由 Go 编排，拉起 Blender 5.2 的统一 bpy 入口
go run ./cmd/assetctl skills run blender-character-rigging
```

具体子命令与参数以阶段三实现为准。

## 28 个 skill 一览（按用途分组）

- **基础与流程**：`blender-core`、`blender-workflow-recipes`、`blender-pipeline-automation`、`blender-asset-qa`
- **建模与几何**：`blender-hardsurface-modeling`、`blender-sculpting`、`blender-retopology-uv`、`blender-geometry-nodes`、`blender-scene-setup`
- **材质与贴图**：`blender-pbr-materials`、`blender-texturing`、`blender-texture-baking`
- **角色与动画**：`blender-character-rigging`、`blender-facial-shapekeys`、`blender-animation`、`blender-clothing`、`blender-hair-grooming`
- **场景与氛围**：`blender-environment-kits`、`blender-lighting-lookdev`、`blender-time-of-day`、`blender-weather-fx`、`blender-simulation`、`blender-stylized-npr`
- **渲染与导出**：`blender-rendering`、`blender-video-editing`、`blender-export-pipelines`、`blender-game-optimization`
- **扩展开发**：`blender-addon-development`
