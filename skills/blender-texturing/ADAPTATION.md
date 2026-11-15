# blender-texturing 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-texturing/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-texturing`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-texturing`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 智能材质层：边缘磨损、cavity 污垢、灰尘、锈迹、roughness 扰动
2. 基于曲率 / AO 遮罩做无 z-fight 的贴花放置
3. trim sheets、平铺与纹理绘制
4. 把材质正确交付到游戏引擎

## 产出与测试
- 执行后产出契约文件 `build/blender-texturing/contracts/skill-blender-texturing.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
