# 90 · Skills 子系统（28 个 Blender Agent Skills）

规范性契约：[SPEC.md §8A](../SPEC.md)。本文件是操作说明与清单。

## 1. 定位

六阶段流水线是**一条固定产线**；Skills 是**可单独调用的操作知识与可执行算子**（倒角、布尔、重拓扑、UV、烘焙、Rigify、几何节点、渲染……）。二者共享同一套硬约束：

| 维度 | 约束 |
|------|------|
| Blender | **仅 5.2**（`skill_run.py` 只用 5.2 存在的 API，不写 4.x 兼容分支） |
| 语言 | Go 1.27（注册表 / 参数校验 / 进程拉起 / 契约复核）+ Blender 5.2 内置 `bpy`（算子） |
| 依赖 | 零第三方依赖；bpy 侧只用 `bpy` / `mathutils` / 标准库 |
| 可测试 | `mock_bpy` + `fake_blender.py`，**无需安装 Blender** |

## 2. 知识来源与许可

- 28 个 `skills/<id>/SKILL.md` **原样照抄**自 [EL4CTEO/blender-skills](https://github.com/EL4CTEO/blender-skills)（MIT）。
- 原文副本：`skills/LICENSE-blender-skills.txt`（含 `Copyright (c) 2026 EL4CTEO`）。
- **不复制其 Python 源码**（其 `scripts/` 下的 `.py` 仅作为 `manifest.json` 的 `referenceScripts` 元数据记录）；本仓库的算子由 `scripts/blender/skill_run.py` 用 bpy 重新实现，避免引入未知许可证与版本假设。
- 每个 skill 另有 `skills/<id>/ADAPTATION.md`：参考来源、对应 Go 命令、bpy 入口、核心要点、产出契约与测试方式。

## 3. CLI

```bash
assetctl skills list                       # 28 个：id / category / summary
assetctl skills list --json                # 机器可读（count / skills[] / blenderVersion / license）
assetctl skills list --category rigging    # 按分组过滤
assetctl skills show blender-core          # 元数据 + ADAPTATION.md
assetctl skills doctor                     # 目录树 / manifest / catalog / dispatcher 一致性
assetctl skills run blender-retopology-uv --asset hero-knight
assetctl skills run blender-rendering --asset hero-knight --param samples=64,exposure=-1 --dry-run
```

`run` 的参数：

| 标志 | 说明 |
|------|------|
| `--asset <id>` | 资产 id（默认 `skill-probe`，仅用于探针） |
| `--param k1=v1,k2=v2` | 传给 bpy 的参数（**逗号分隔**，键按字典序排序后拼入 argv，保证可复现） |
| `--dry-run` | 只打印 blender 命令行，不执行 |
| `--root` / `--blender` | 仓库根 / Blender 5.2 可执行文件 |

退出码与主 CLI 一致：`0` 成功、`2` 用法/校验失败、`3` 需人工介入（handoff manual）。

## 4. 执行链

```
assetctl skills run <id>
  └─ internal/skills.Run
       ├─ ByID(<id>)              未知 id → 2
       ├─ 校验 asset id / 参数
       ├─ 拼 argv（排序 params）
       ├─ runner.FindBlender      仅 5.2
       └─ blender --background --factory-startup --python scripts/blender/skill_run.py
            └─ --skill <id> --asset <id> --config <asset.json>
               --out build/<id>/contracts/skill-<id>.json
               --blend-out work/<id>/skills/<id>.blend
```

bpy 侧 `skill_run.py`：

1. `--skill` → `operation`（catalog 与脚本共用同一套 key）；
2. 按 operation 取 2–5 个算子步骤，逐步 `try/except` 记录 `steps[] = {label, ok, detail}`；
3. 汇总 `outputs[]`（相对 repo root 的 POSIX 路径 + `bytes` + `sha256`）；
4. 保存 `.blend`，写契约 JSON，并打印 `@@JSON@@ <json>` 行协议摘要。

## 5. 契约

`build/<id>/contracts/skill-<id>.json`，`schema = 1`：

```json
{
  "schema": 1,
  "asset": "hero-knight",
  "stage": "skill-blender-retopology-uv",
  "skill": "blender-retopology-uv",
  "operation": "retopology_uv",
  "blenderVersion": "5.2",
  "tripoUrl": "https://studio.tripo3d.com/",
  "params": {"supplier": "tripo"},
  "steps": [{"label": "...", "ok": true, "detail": "..."}],
  "outputs": [{"path": "work/hero-knight/skills/blender-retopology-uv.blend", "bytes": 1234, "sha256": "..."}],
  "result": {"ok": true, "failed": 0},
  "startedAt": "...", "finishedAt": "..."
}
```

Go 侧读完契约后**必须**复核：`schema==1`、`stage == "skill-"+id`、`skill == id`、`operation` 与 catalog 一致；任一不符即失败（防止脚本"成功"但产物串号）。

## 6. 28 个 skill 清单

| # | id | category | operation | 六阶段关联 |
|---|----|----------|-----------|-----------|
| 1 | `blender-core` | foundation | `core` | — |
| 2 | `blender-scene-setup` | foundation | `scene_setup` | modeling |
| 3 | `blender-hardsurface-modeling` | modeling | `hardsurface_modeling` | modeling, topology-uv |
| 4 | `blender-sculpting` | modeling | `sculpting` | modeling |
| 5 | `blender-retopology-uv` | modeling | `retopology_uv` | topology-uv |
| 6 | `blender-geometry-nodes` | modeling | `geometry_nodes` | modeling |
| 7 | `blender-pbr-materials` | materials | `pbr_materials` | pbr |
| 8 | `blender-texturing` | materials | `texturing` | pbr |
| 9 | `blender-texture-baking` | materials | `texture_baking` | bake |
| 10 | `blender-character-rigging` | rigging | `character_rigging` | pbr, unity-export |
| 11 | `blender-facial-shapekeys` | rigging | `facial_shapekeys` | pbr |
| 12 | `blender-animation` | rigging | `animation` | unity-export |
| 13 | `blender-clothing` | rigging | `clothing` | topology-uv |
| 14 | `blender-hair-grooming` | rigging | `hair_grooming` | topology-uv |
| 15 | `blender-game-optimization` | pipeline | `game_optimization` | topology-uv |
| 16 | `blender-export-pipelines` | pipeline | `export_pipelines` | unity-export |
| 17 | `blender-pipeline-automation` | pipeline | `pipeline_automation` | — |
| 18 | `blender-asset-qa` | qa | `asset_qa` | unity-export, render |
| 19 | `blender-workflow-recipes` | qa | `workflow_recipes` | — |
| 20 | `blender-rendering` | rendering | `rendering` | render |
| 21 | `blender-lighting-lookdev` | rendering | `lighting_lookdev` | render |
| 22 | `blender-stylized-npr` | rendering | `stylized_npr` | pbr |
| 23 | `blender-environment-kits` | rendering | `environment_kits` | modeling |
| 24 | `blender-time-of-day` | rendering | `time_of_day` | render |
| 25 | `blender-weather-fx` | rendering | `weather_fx` | render |
| 26 | `blender-simulation` | rendering | `simulation` | — |
| 27 | `blender-video-editing` | rendering | `video_editing` | — |
| 28 | `blender-addon-development` | pipeline | `addon_development` | — |

## 7. 测试

| 层 | 位置 | 覆盖 |
|----|------|------|
| Go 注册表 | `internal/skills/skills_test.go` | catalog 与 `skills/manifest.json` 一一对应、operation 唯一、未知 skill / 非法 asset id、dry-run argv 排序、契约串号与缺失 |
| Go 校验器 | `assetctl skills doctor`（`skills.ValidateCatalog`） | 28 个 `SKILL.md` 与 `ADAPTATION.md` 存在且非空、`manifest.json` 字段完整、`skill_run.py` 存在 |
| bpy 全量 | `testing/test_skills.py` | 在 `mock_bpy` 下跑通**全部 28 个** skill：退出码 0、契约 schema/身份/operation 正确、`result.ok`、`outputs` 非空且摘要可重算 |
| 一键 | `assetctl verify` 第 2 步 | 注册表一致性随本地 `verify` 自动跑（无 CI 服务） |

## 8. 新增 skill 的步骤

1. `skills/<id>/SKILL.md`（frontmatter `name` / `description` + 正文）与 `ADAPTATION.md`；
2. `skills/manifest.json` 增加一项（`id/name/description/referenceScripts/goCommand/bpyScript`）；
3. `internal/skills/catalog.go` 的 `Catalog` 增加 `{id, category, summary, operation, stages}`；
4. `scripts/blender/skill_run.py` 的 `OPS` 增加该 `operation` 的步骤列表；
5. `assetctl skills doctor` + `assetctl verify` 必须全绿（缺任一项会直接失败，避免文档与执行漂移）。
