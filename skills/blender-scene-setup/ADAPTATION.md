# blender-scene-setup 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-scene-setup/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-scene-setup`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-scene-setup`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 设置单位与真实世界尺度、规划 collection 布局
2. 遵循 SM_/SK_/T_/M_ 命名约定并归正原点
3. 区分 linking 与 appending，使用 library overrides
4. 建立 asset libraries

## 产出与测试
- 执行后产出契约文件 `build/blender-scene-setup/contracts/skill-blender-scene-setup.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
