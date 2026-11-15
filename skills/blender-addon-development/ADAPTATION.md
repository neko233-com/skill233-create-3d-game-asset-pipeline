# blender-addon-development 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-addon-development/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-addon-development`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-addon-development`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. 以扩展（extension）形式编写插件，核心是 blender_manifest.toml 清单
2. 实现 Operator / Panel / Menu / 属性组 / 偏好设置，遵循 5.x 的 bpy.props 规则
3. 处理 undo/redo、modal 与 timer 模式，GPU 绘制不再使用 bgl
4. 发布到 extensions.blender.org，参考 assets/addon_template 模板

## 产出与测试
- 执行后产出契约文件 `build/blender-addon-development/contracts/skill-blender-addon-development.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
