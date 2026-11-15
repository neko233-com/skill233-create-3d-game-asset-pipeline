# blender-sculpting 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-sculpting/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-sculpting`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-sculpting`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. base → detail 流程，voxel remesh、dyntopo 与 multires
2. 5.x 笔刷（Essentials 资产）、mask / face sets 与对称
3. 使用 headless-safe 的程序化操作（remesh / symmetrize）
4. 遵循雕刻经验法则

## 产出与测试
- 执行后产出契约文件 `build/blender-sculpting/contracts/skill-blender-sculpting.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
