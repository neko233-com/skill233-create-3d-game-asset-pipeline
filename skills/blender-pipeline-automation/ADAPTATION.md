# blender-pipeline-automation 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-pipeline-automation/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-pipeline-automation`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-pipeline-automation`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 掌握 headless CLI 标志，批量转换 / 清点 / 运行脚本
2. 对 .blend/.fbx/.glb/.obj/.usd 批处理与分片
3. 把 bpy 当作 Python 模块，处理库链接与路径管理
4. 编写可复用的 CI recipe（GitHub Actions）

## 产出与测试
- 执行后产出契约文件 `build/blender-pipeline-automation/contracts/skill-blender-pipeline-automation.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
