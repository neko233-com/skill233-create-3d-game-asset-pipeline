# blender-core 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-core/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-core`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-core`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. bpy 的两种运行方式：MCP live 会话与 headless CLI
2. context overrides、locale-safe 节点查找与 enum discovery
3. 掌握 Blender 5.x 相对旧教程的 API 变更
4. 收尾前必须 Verify 验证，并路由到各专用 blender-* skill

## 产出与测试
- 执行后产出契约文件 `build/blender-core/contracts/skill-blender-core.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
