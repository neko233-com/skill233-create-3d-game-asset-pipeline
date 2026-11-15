# 70 · 命名规范 + Definition of Done

> 契约见 [SPEC.md §6/§7/§11](../SPEC.md)（目录、`asset.json`、命名与 DoD）与 [AGENTS.md](../AGENTS.md)。本文是提交前**逐项勾选**的清单。

## 1. 资产 ID（小写连字符）

| 项 | 规则 | 示例 |
|----|------|------|
| 资产 ID | **全小写，单词用连字符** `[a-z0-9]+(-[a-z0-9]+)*` | `hero-knight`、`props-sword`、`npc-villager` |
| 动画 clip | `{asset-id}-{action}` | `hero-knight-idle`、`hero-knight-walk` |
| 禁止 | 大写、下划线、空格、中文、连续连字符 | `HeroKnight` ✗、`hero_knight` ✗ |

Go 侧 `config.Validate()` 用正则 `^[a-z0-9]+(?:-[a-z0-9]+)*$` 强校验，不合规直接报错。目录按 `characters/`、`props/`、`animations/` 分类（AGENTS.md）。

## 2. 目录约定（SPEC.md §6）

```
assets/<id>/asset.json          资产配置（JSON，Go encoding/json）
raw/<id>.*                      策略产物（原始，保留，便于回溯）
raw/<id>.png                    归一化参考图
work/<id>/NN-<stage>.blend      阶段中间 .blend（01–06）
build/<id>/<id>.fbx             最终 Unity 资产
build/<id>/<id>_basecolor.png   PBR 贴图（normal/roughness/metallic/ao 同理）
build/<id>/preview.png          阶段 6 渲染合成图
build/<id>/manifest.json        阶段状态 + DoD 结果（机器可读）
scripts/blender/                Blender 5.2 内置 Python（bpy）阶段脚本
cmd/assetctl/                   Go 1.27 CLI 入口
internal/strategy/              Go 1.27 策略接口与注册表
internal/config/                Go 1.27 asset.json 解析与校验
docs/                           本套中文文档
examples/                       配置与清单示例
```

`manifest.json`（机器可读，阶段状态 + DoD）与 `examples/asset-manifest.example.md`（人可读示例）分工见 [42](42-stage-05-unity-export.md) / [80](80-cli-reference.md)。

## 3. 贴图后缀表（SPEC.md §11）

| 后缀 | 贴图 | 色彩空间 | Unity URP/Lit 槽位 |
|------|------|----------|--------------------|
| `_basecolor.png` | 基础色 / Albedo | sRGB | Base Map |
| `_normal.png` | 切线空间法线 | **Non-Color** | Normal Map（Texture Type = `Normal Map`） |
| `_roughness.png` | 粗糙度 | **Non-Color** | Smoothness（= 1 − Roughness） |
| `_metallic.png` | 金属度 | **Non-Color** | Metallic |
| `_ao.png` | 环境光遮蔽（可选） | **Non-Color** | Occlusion |

文件形如 `build/hero-knight/hero-knight_basecolor.png`。`asset.json.textures` 默认 `["basecolor", "normal", "roughness", "metallic"]`；`_ao` 按需追加（详见 [41](41-stage-04-pbr.md)）。

## 4. 骨骼命名（Unity Humanoid，SPEC.md §11）

- 根骨骼 `Hips`，脊柱 `Spine` / `Chest` / `Neck` / `Head`。
- 左右用 **`Left` / `Right` 前缀**：`LeftUpperArm`、`RightUpperLeg`……不要 `L_` / `R_`。
- 完整层级表、clip 清单与常见坑见 [50-humanoid-animation.md](50-humanoid-animation.md)。

## 5. 提交信息（Conventional Commits）

AGENTS.md 规定使用 Conventional Commits：

| 类型 | 用途 | 示例 |
|------|------|------|
| `feat:` | 新能力 / 新策略 / 新阶段 | `feat(strategy): add meshy strategy` |
| `fix:` | 修 bug | `fix(export): keep add_leaf_bones=False` |
| `docs:` | 文档变更 | `docs: rewrite pipeline docs per SPEC.md` |
| `chore:` | 构建 / 配置 / 杂务 | `chore: bump go.mod to 1.27` |
| `refactor:` | 重构（不改行为） | `refactor(config): extract Validate` |

格式：`<type>(<scope>): <subject>`，subject 用小写祈使句。本轮文档任务即 `docs:` 类。**不要 git commit**（按任务约束）。

## 6. Definition of Done（SPEC.md §11）

总门禁一句话：**`assetctl validate` 通过（含 `manifest.json` 六阶段全 `ok`）+ `assetctl doctor` 通过 + Unity 中无粉材质、无 T-pose、Avatar 映射无红项。**

### 6.1 逐项 DoD 勾选清单

**命名与结构**
- [ ] 资产 ID 小写连字符；clip 为 `{asset-id}-{action}`
- [ ] 目录符合 §2：`assets/<id>/asset.json`、`raw/`、`work/`、`build/` 齐全
- [ ] 贴图后缀为 `_basecolor` / `_normal` / `_roughness` / `_metallic` / `_ao`
- [ ] 骨骼 `Hips` 为根、`Left`/`Right` 前缀、层级连续
- [ ] 提交信息为 Conventional Commits

**资产内容**
- [ ] `asset.json` 字段合法：`strategy` ∈ `blender|tripo|import`，`blenderVersion` = `5.2`，`tripoUrl` = https://studio.tripo3d.com/ ，`axes` = `-Z / Y / 1.0`，`targetEngine` = `unity-urp`
- [ ] 面数落在预算区间且 ≤ `budget.maxTriangles`（[30](30-stage-02-topology-uv.md)）
- [ ] 贴图齐全、尺寸 ≤ `budget.maxTextureSize`，`_normal` 为 Non-Color / Normal Map
- [ ] 主 UV 无重叠、第二套 `LightmapUV` 独立（[30](30-stage-02-topology-uv.md)）
- [ ] FBX 导出参数严格等于 [SPEC.md §9](../SPEC.md) 的 7 项（[42](42-stage-05-unity-export.md)）
- [ ] `humanoid: true` 时 `rig_check.py` 通过（[50](50-humanoid-animation.md)）
- [ ] clip 循环标记正确（idle/walk/run 循环，attack/hurt/death 单次）

**交付与引擎内验收**
- [ ] `build/<id>/manifest.json` 记录各阶段状态 + DoD 结果，且**六阶段全为 `ok`**（`assetctl validate` 的 DoD 门禁）
- [ ] `preview.png` 已生成（[43](43-stage-06-render-composite.md)）
- [ ] **`assetctl doctor` 工具链报告通过**（Go 1.27 / Blender 5.2 或 `ASSETCTL_BLENDER` / 供应商+DEFAULT / Tripo URL / axes / 六阶段脚本齐备）
- [ ] **`assetctl validate` 通过**
- [ ] Unity 中**无粉材质**（贴图 / Shader 丢失）
- [ ] Unity 中**无 T-pose**（Rig → Humanoid、Avatar 配置正确）
- [ ] **Avatar 映射无红项**（Configure 里无缺失骨骼）
- [ ] Prefab 已保存、Animator Controller 已绑定、idle 正常循环
- [ ] 本文对应清单项已勾选

## 7. `assetctl validate` 用法

`assetctl validate` 是 DoD 的机器入口（命令表见 [SPEC.md §8](../SPEC.md) 与 [80](80-cli-reference.md)）：

```bash
# 基本校验：命名、FBX、贴图后缀、build 目录完整性
assetctl validate build/hero-knight

# 人形资产：额外要求骨骼标记 / rig 结果
assetctl validate build/hero-knight --expect-humanoid

# 要求贴图齐全（逐张核对 basecolor/normal/roughness/metallic）
assetctl validate build/hero-knight --expect-textures

# 全量（CI / 提交前推荐）
assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

检查项：
- `build/<id>/<id>.fbx` 存在；
- 资产 ID 符合小写连字符正则；
- 贴图文件存在且后缀符合 §3（`--expect-textures` 时逐张核对 `asset.json.textures`）；
- `--expect-humanoid` 时核对人形标记 / `rig_check` 结果；
- `manifest.json` 记录的阶段状态与 DoD 结果一致，且**六阶段全部为 `ok`**（任一 `failed` / `skipped` 都不算完成）。

生产 onboarding 先用 `assetctl doctor` 自检工具链；`go test ./...` 是无需 Blender 5.2 的完整自动化验证（[80](80-cli-reference.md)）。退出码：`0` = 全部通过；非 `0` = 列出问题并写 stderr。CI 直接把它当门禁（[80](80-cli-reference.md)）。

> 校验是**只读**的，不破坏 `build/` 里任何文件；失败时按输出逐项回到对应阶段文档修正，再重跑 `assetctl run --from <stage>` 与 `validate`。
