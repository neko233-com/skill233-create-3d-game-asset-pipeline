# blender-lighting-lookdev 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-lighting-lookdev/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-lighting-lookdev`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-lighting-lookdev`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 三点布光 / 产品布光、HDRI 环境与太阳天空外景
2. light linking 与物理光照单位
3. 相机焦距与 turntable 展示
4. 材质审批的 look-dev 流程

## 产出与测试
- 执行后产出契约文件 `build/blender-lighting-lookdev/contracts/skill-blender-lighting-lookdev.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
