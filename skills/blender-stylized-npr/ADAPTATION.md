# blender-stylized-npr 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-stylized-npr/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-stylized-npr`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-stylized-npr`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. toon / cel 着色：色带、阴影染色、rim、硬高光
2. 动漫脸的自定义法线处理
3. inverted-hull 描边与平滑描边法线
4. Grease Pencil（5.x API）与导出到游戏引擎

## 产出与测试
- 执行后产出契约文件 `build/blender-stylized-npr/contracts/skill-blender-stylized-npr.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
