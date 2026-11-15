# blender-environment-kits 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-environment-kits/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-environment-kits`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-environment-kits`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 由噪声或高度图生成地形（侵蚀、梯田、坡度 / 高度材质）
2. 导出 16-bit 高度图与 splat map 供 UE / Unity 使用
3. 按坡度 / 高度 / 生物群系进行散布
4. 模块化 kit 的一致性检查

## 产出与测试
- 执行后产出契约文件 `build/blender-environment-kits/contracts/skill-blender-environment-kits.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
