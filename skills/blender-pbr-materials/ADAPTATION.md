# blender-pbr-materials 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-pbr-materials/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-pbr-materials`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-pbr-materials`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. metal / roughness 规则与 Principled BSDF 输入接线
2. 区分 sRGB 与 Non-Color 色彩空间，使用实测 albedo / F0
3. 贴图集自动接线与 ORM 通道打包
4. 处理引擎导出、alpha 与 displacement

## 产出与测试
- 执行后产出契约文件 `build/blender-pbr-materials/contracts/skill-blender-pbr-materials.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
