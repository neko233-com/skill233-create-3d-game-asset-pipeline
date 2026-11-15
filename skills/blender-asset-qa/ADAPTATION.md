# blender-asset-qa 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-asset-qa/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-asset-qa`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-asset-qa`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 任何网格 / 资产编辑后的强制审查门禁
2. 检测 z-fighting、悬浮或游离部件、法线翻转与不一致
3. 发现非流形 / 开放几何、退化面等拓扑问题
4. 补充脚本无法完全判断的人工检查项

## 产出与测试
- 执行后产出契约文件 `build/blender-asset-qa/contracts/skill-blender-asset-qa.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
