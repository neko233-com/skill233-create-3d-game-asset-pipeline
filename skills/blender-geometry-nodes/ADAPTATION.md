# blender-geometry-nodes 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-geometry-nodes/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-geometry-nodes`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-geometry-nodes`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 理解 fields / domains / attributes / instancing 核心概念
2. 使用 zones（simulation / repeat / for-each）与 closures / bundles / lists
3. SDF grids、node tools 与 baking
4. 通过 JSON→节点树 builder 程序化搭建

## 产出与测试
- 执行后产出契约文件 `build/blender-geometry-nodes/contracts/skill-blender-geometry-nodes.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
