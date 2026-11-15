# blender-facial-shapekeys 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-facial-shapekeys/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-facial-shapekeys`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-facial-shapekeys`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. ARKit 52 blendshapes 与 lip-sync 用 visemes
2. 雕刻表情，把对称形态键拆分为 Left / Right 并镜像
3. 构建 corrective 与 combination 形态键
4. 对齐各引擎的形态键导入注意事项

## 产出与测试
- 执行后产出契约文件 `build/blender-facial-shapekeys/contracts/skill-blender-facial-shapekeys.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
