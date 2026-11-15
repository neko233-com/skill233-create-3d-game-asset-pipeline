# blender-workflow-recipes 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-workflow-recipes/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-workflow-recipes`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-workflow-recipes`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 串联多个 blender-* skill 并插入 QA 门禁
2. 游戏道具手册：mid-poly 快速流程与 high → low 烘焙流程
3. 游戏角色手册：雕刻 / AI mesh → 绑定 → 动画导出
4. Roblox、清理 AI / 下载资产、程序化环境等端到端剧本

## 产出与测试
- 执行后产出契约文件 `build/blender-workflow-recipes/contracts/skill-blender-workflow-recipes.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
