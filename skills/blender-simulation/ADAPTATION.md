# blender-simulation 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-simulation/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-simulation`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-simulation`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 刚体：密度质量、碰撞形状、bake to keys
2. 经典布料预设与钉固、节点式 Cloth / Hair 动力学（XPBD）
3. Mantaflow 流体与 hair curves 动力学
4. 根据场景选择合适的求解器

## 产出与测试
- 执行后产出契约文件 `build/blender-simulation/contracts/skill-blender-simulation.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
