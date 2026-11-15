# blender-video-editing 适配说明

- **参考来源**：照抄自 EL4CTEO/blender-skills（MIT）的 `skills/blender-video-editing/SKILL.md`，原文未改动，仅作知识/规范参考。
- **对应 Go CLI**：`assetctl skills run blender-video-editing`（Go 1.27 编排）。
- **统一 bpy 入口**：`scripts/blender/skill_run.py --skill blender-video-editing`（Blender 5.2 内置 bpy）。

## 核心操作要点（提炼自 SKILL.md）
1. VSE 剪辑：片段、图像序列、交叉淡化、字幕、音乐淡入淡出
2. proxies、色彩管理与 H.264/AAC 输出
3. 运动跟踪：特征检测与 camera solve
4. 遵循让解算成功的拍摄素材技巧

## 产出与测试
- 执行后产出契约文件 `build/blender-video-editing/contracts/skill-blender-video-editing.json`。
- 由 `testing/test_skills.py` 在 mock_bpy 下自动测试。
