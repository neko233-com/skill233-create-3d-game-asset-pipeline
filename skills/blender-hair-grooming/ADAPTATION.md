# blender-hair-grooming 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-hair-grooming/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-hair-grooming`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-hair-grooming`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 在曲面上建立 hair curves，guides 加插值子发
2. 使用内置毛发节点资产（clump / curl / noise / droop）与密度遮罩
3. Principled Hair BSDF（melanin）着色
4. 生成游戏用 hair cards 并导出 groom

## 产出与测试
- 执行后产出契约文件 `build/blender-hair-grooming/contracts/skill-blender-hair-grooming.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
