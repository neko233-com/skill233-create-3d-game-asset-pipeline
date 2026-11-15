# blender-clothing 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-clothing/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-clothing`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-clothing`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 从身体区域生成衣物（ease offset、厚度）与蒙皮权重转移
2. 隐藏被衣物覆盖的身体面，避免穿模
3. 布料模拟生成褶皱，身体 morph 驱动衣物 shape keys
4. 处理分层衣物的层叠与穿插

## 产出与测试
- 执行后产出契约文件 `build/blender-clothing/contracts/skill-blender-clothing.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
