# blender-weather-fx 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-weather-fx/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-weather-fx`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-weather-fx`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 几何节点循环雨 / 雪 / 尘 / 灰烬 / 火花及地面命中飞溅
2. 湿表面、水坑涟漪、墙面水痕与积雪
3. 闪电效果
4. 适配游戏 / 实时渲染

## 产出与测试
- 执行后产出契约文件 `build/blender-weather-fx/contracts/skill-blender-weather-fx.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
