# blender-game-optimization 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-game-optimization/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-game-optimization`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-game-optimization`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 按平台（mobile / web / VR / console / PC）控制面数、材质、贴图与骨骼预算
2. 构建 LOD 链，选择合适的 Decimate 模式并清理网格
3. 关注顶点分裂等隐性开销
4. 降低 draw call 并合批

## 产出与测试
- 执行后产出契约文件 `build/blender-game-optimization/contracts/skill-blender-game-optimization.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
