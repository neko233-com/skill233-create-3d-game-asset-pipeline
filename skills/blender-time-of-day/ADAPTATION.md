# blender-time-of-day 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-time-of-day/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-time-of-day`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-time-of-day`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 实测天空 + 太阳：黎明 / 正午 / 黄金时刻 / 日落 / 夜晚
2. 曝光平衡、高度雾与体积效果
3. 阴天、月光星空与城市辉光
4. 保证物理正确性并遵循规则

## 产出与测试
- 执行后产出契约文件 `build/blender-time-of-day/contracts/skill-blender-time-of-day.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
