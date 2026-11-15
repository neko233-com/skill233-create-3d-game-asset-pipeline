# blender-rendering 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-rendering/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-rendering`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-rendering`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. Cycles / EEVEE / Workbench 质量预设与 GPU 设置
2. 自适应采样、降噪与 light paths
3. 统一色彩管理（AgX / Khronos PBR Neutral / ACES）
4. PNG/EXR/MP4 输出、passes 与 compositor（5.0+）

## 产出与测试
- 执行后产出契约文件 `build/blender-rendering/contracts/skill-blender-rendering.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
