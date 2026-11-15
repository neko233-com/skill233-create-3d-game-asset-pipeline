# 3D Game Asset Pipeline — 游戏美术管线向

[![Go 1.27](https://img.shields.io/badge/Go-1.27-00ADD8)](https://go.dev/)
[![Blender 5.2](https://img.shields.io/badge/Blender-5.2-orange)](https://www.blender.org/)
[![Tripo](https://img.shields.io/badge/Tripo-studio.tripo3d.com-blue)](https://studio.tripo3d.com/)
[![License MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)

**多策略 · 六阶段 · Go 1.27 编排**的 3D 游戏资产生产流水线：把参考图 / 参数变成 **Unity（URP）** 可直接使用的 FBX + PBR 贴图，覆盖人形角色、道具与人形角色动画。

> - Tripo 入口：**https://studio.tripo3d.com/**（建模策略之一，**非默认**）
> - 默认建模策略：**Blender 5.2 参数化建模**
> - 参考插件：[ACT — Blender Asset Creation Toolset](https://github.com/mrven/Blender-Asset-Creation-Toolset)（GPL-3.0，只对齐行为与参数、不 vendoring 源码）
> - 📐 规范性契约：[SPEC.md](SPEC.md) · 硬性版本 / 语言分工：[AGENTS.md](AGENTS.md) · 可复用 Skill：[SKILL.md](SKILL.md)

## 为什么存在

把「一张图 / 一段参数 → Unity 里能跑的角色」标准化为**游戏美术管线**顺序，适合 Unity 主程 + TA 管线：固定工具版本、固定坐标系、固定命名与校验、可断点续跑，避免每次手动踩坑。

```
                ┌─────────────── 多策略（阶段 1 可替换）───────────────┐
                │  blender(默认,全自动)  /  tripo(半自动)  /  import    │
                └──────────────────────┬───────────────────────────────┘
                                       ▼
  参考图 ──> ① 建模 ──> ② 拓扑/UV ──> ③ 烘焙 ──> ④ PBR 材质 ──> ⑤ Unity 导出/同步 ──> ⑥ 渲染合成 ──> Unity (URP)
              ▲                                                     │                    │
              │                                          humanoid: rig_check.py 门禁     preview.png
        assets/<id>/asset.json                            FBX + PBR 贴图 + manifest.json
        （Go 1.27 assetctl 编排，Blender 5.2 内置 bpy 执行六阶段脚本）
```

## 语言分工（硬性，AGENTS.md）

| 运行位置 | 语言 |
|----------|------|
| CLI / 编排器 / 策略 / 配置 / 校验 / 参考图处理 | **Go 1.27**（零第三方依赖，仅标准库） |
| Blender 内部六阶段脚本 / rig 校验 | **Blender 5.2 原生内置 `bpy`** |
| 系统 Python 3.14.3 | 仅开发辅助检查，不进交付流水线 |

## 快速开始

```bash
# 0. 构建 + 自测（Go 1.27）
go build ./... && go vet ./... && go test ./...

# 1. 查看策略与六阶段
go run ./cmd/assetctl strategies
go run ./cmd/assetctl stages

# 2. 初始化资产配置（默认策略 blender；人形角色加 --humanoid）
go run ./cmd/assetctl init hero-knight --humanoid --kind character

# 3. 预演（只打印六条 Blender 命令，不执行、不需要 Blender）
go run ./cmd/assetctl run hero-knight --dry-run

# 4. 真正执行（需本机 Blender 5.2；找不到时用 --blender <path> 指定）
go run ./cmd/assetctl run hero-knight

# 5. 交付前校验
go run ./cmd/assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

### 切到 Tripo 策略（半自动）

```bash
go run ./cmd/assetctl init hero-knight --strategy tripo
# 人工步骤：打开 https://studio.tripo3d.com/ ，Image-to-3D 上传参考图，下载 GLB 到 raw/hero-knight.glb
go run ./cmd/assetctl run hero-knight --from modeling   # 缺 GLB 时 Go 会打印 4 步指引并 exit 3
```

参考图可用纯 Go 归一化：`go run ./cmd/assetctl reference prepare concept.png raw/ --size 1024`。

## CLI 一览（SPEC.md §8）

| 命令 | 作用 |
|------|------|
| `assetctl version` | 版本与工具链 |
| `assetctl strategies` | 列出建模策略，标注 DEFAULT |
| `assetctl stages` | 列出六阶段 |
| `assetctl init <id>` | 创建 `assets/<id>/asset.json` |
| `assetctl run <id>` | 执行六阶段，写 `build/<id>/manifest.json` |
| `assetctl reference prepare <in> <out>` | 参考图归一化（纯 Go） |
| `assetctl validate <build-dir>` | 交付前校验（DoD） |
| `assetctl blender <script> [-- --args]` | 透传 Blender 5.2 |

`run` 支持 `--strategy`（覆盖配置）、`--from` / `--only`（断点续跑）、`--dry-run`（只打印命令）。

## 六阶段与文档索引

| # | 阶段 | 实现 | 文档 |
|---|------|------|------|
| 1 | 建模（策略） | `internal/strategy/*` + `scripts/blender/stage_01_modeling.py` | [docs/10-strategies.md](docs/10-strategies.md) · [docs/20-stage-01-modeling.md](docs/20-stage-01-modeling.md) |
| 2 | 拓扑 / UV | `scripts/blender/stage_02_topology_uv.py` | [docs/30-stage-02-topology-uv.md](docs/30-stage-02-topology-uv.md) |
| 3 | 烘焙 | `scripts/blender/stage_03_bake.py` | [docs/40-stage-03-bake.md](docs/40-stage-03-bake.md) |
| 4 | PBR 材质 | `scripts/blender/stage_04_pbr.py` | [docs/41-stage-04-pbr.md](docs/41-stage-04-pbr.md) |
| 5 | Unity 导出 / 同步 | `scripts/blender/stage_05_unity_export.py` | [docs/42-stage-05-unity-export.md](docs/42-stage-05-unity-export.md) |
| 6 | 渲染合成 | `scripts/blender/stage_06_render.py` | [docs/43-stage-06-render-composite.md](docs/43-stage-06-render-composite.md) |

横切与参考文档：[总览](docs/00-overview.md) · [人形动画](docs/50-humanoid-animation.md) · [ACT 整合](docs/60-act-integration.md) · [语言分工](docs/61-language-split.md) · [命名与 DoD](docs/70-naming-and-dod.md) · [CLI 参考](docs/80-cli-reference.md)。

## 目录结构

```
SPEC.md / AGENTS.md / SKILL.md     契约 + 硬性事实 + 可复用 Skill
cmd/assetctl/                      Go 1.27 CLI 入口
internal/config/                   asset.json 配置与校验
internal/strategy/                 多策略注册表（blender/tripo/import）
internal/runner/                   六阶段执行 + manifest.json + rig 门禁
internal/reference/                参考图归一化（纯 Go）
internal/validate/                 交付校验（DoD）
scripts/blender/                   Blender 5.2 内置 bpy 脚本（stage_01~06 + rig_check + humanoid_rig + tripo_to_unity）
docs/                              分阶段文档（00/10/20…80）
examples/                          asset.json / 提示词 / 清单示例
assets/<id>/asset.json             每个资产的配置
raw/ work/ build/                  策略产物 / 阶段中间 .blend / 最终交付
```

## 导出参数（对齐 Unity + ACT，SPEC.md §9）

`axis_forward=-Z` · `axis_up=Y` · `global_scale=1.0` · `apply_unit_scale=True` · `add_leaf_bones=False` · `bake_anim=True` · `path_mode=COPY`。
Unity Import Settings 勾选 **Bake Axis Conversion**（采用 ACT `"Unity"` 档；`"Unity (Pre-Rotated)"` 不支持动画故不采用）。

## DoD

`assetctl validate` 通过 + Unity 中无粉材质、无 T-pose、Avatar 映射无红项 + `manifest.json` 六阶段全 `ok`。

## License

本仓库 **MIT**；ACT 为 **GPL-3.0**，仅作为可选外部插件在文档中引用，不纳入本仓库代码。
