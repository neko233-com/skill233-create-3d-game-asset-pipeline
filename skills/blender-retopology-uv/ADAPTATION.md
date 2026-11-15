# blender-retopology-uv 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-retopology-uv/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-retopology-uv`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-retopology-uv`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 自动重拓扑（Quadriflow / Decimate / planar）
2. 变形网格的边流、seams 与 unwrap 方式
3. 按像素边距 packing，测量 texel density
4. 处理 UV 特殊情形

## 产出与测试
- 执行后产出契约文件 `build/blender-retopology-uv/contracts/skill-blender-retopology-uv.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
