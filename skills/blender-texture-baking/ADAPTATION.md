# blender-texture-baking 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-texture-baking/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-texture-baking`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-texture-baking`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. Cycles high → low 法线烘焙（cage / extrusion）
2. 烘焙 AO / curvature / ID / thickness / base color 等贴图
3. ORM 打包、OpenGL 与 DirectX 法线方向、边距设置
4. 修复常见烘焙伪影，处理非 PBR 数据

## 产出与测试
- 执行后产出契约文件 `build/blender-texture-baking/contracts/skill-blender-texture-baking.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
