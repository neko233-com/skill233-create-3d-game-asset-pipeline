# blender-hardsurface-modeling 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-hardsurface-modeling/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-hardsurface-modeling`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-hardsurface-modeling`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 遵循 blockout → detail 流程与非破坏性修改器栈
2. 组合 Mirror / Boolean Exact / Bevel / Weighted Normal / Smooth by Angle
3. 在 mid-poly 与 subdivision 之间选择工作流
4. 遵守拓扑规则并使用 4.0+ 命名的边 / 面属性

## 产出与测试
- 执行后产出契约文件 `build/blender-hardsurface-modeling/contracts/skill-blender-hardsurface-modeling.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
