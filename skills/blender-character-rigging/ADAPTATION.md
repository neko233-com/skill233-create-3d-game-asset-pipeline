# blender-character-rigging 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-character-rigging/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-character-rigging`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-character-rigging`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. Rigify metarig 拟合 / 生成，骨骼命名、roll 与 collections 规则
2. DEF / MCH / control 分层与 IK / FK 控制
3. 稳健蒙皮（heat 权重 + voxel-proxy 回退）与权重绘制规则
4. 对齐 Unity Humanoid 游戏骨架，rig 校验与形变压力测试

## 产出与测试
- 执行后产出契约文件 `build/blender-character-rigging/contracts/skill-blender-character-rigging.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
