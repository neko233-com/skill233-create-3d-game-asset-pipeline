# blender-export-pipelines 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-export-pipelines/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-export-pipelines`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-export-pipelines`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 按引擎加载验证过的预设：Unity(FBX, Humanoid)、Unreal、Roblox、Godot/web、USD
2. 执行通用导出前检查清单并选择合适的格式
3. 使用 collection exporters（4.2+）一键重导出
4. 导出后必须 Verify 验证结果

## 产出与测试
- 执行后产出契约文件 `build/blender-export-pipelines/contracts/skill-blender-export-pipelines.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
