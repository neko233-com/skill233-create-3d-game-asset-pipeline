# blender-animation 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-animation/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-animation`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-animation`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 理解 5.x 分层 / 槽位 Action 与 channelbag 数据模型
2. Python 关键帧、F-Curve、NLA clips 与 pose library 用法
3. 游戏循环动画需做 pop / hitch 检查，控制骨 bake 到导出骨架
4. Mocap / BVH / Mixamo 重定向与自动映射

## 产出与测试
- 执行后产出契约文件 `build/blender-animation/contracts/skill-blender-animation.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
