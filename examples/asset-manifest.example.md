# Asset Manifest 示例 · hero-knight

示例资产清单。真正运行时由 **`assetctl run`** 自动生成 `build/<asset-id>/manifest.json`（Go 编排器写出，见 [docs/80-cli-reference.md](../docs/80-cli-reference.md)）；本文件是人工可读的 DoD 勾选板，复制到 `build/<asset-id>/manifest.md` 按实际填写。

> 本仓库的规范以 [SPEC.md](../SPEC.md) 为准。多策略 / 供应商：`tripo`(**DEFAULT 默认供应商**) / `blender` / `import`（可选）；六阶段顺序固定。「供应商 == 建模策略」，Go `Strategy` 类型不改名。
> 可直接复制的 `asset.json` 配置见 [examples/asset.json](asset.json)。

## 资产信息

- Asset ID: `hero-knight`（小写连字符）
- Kind: `character` / `humanoid` / animated
- Strategy / Supplier: `tripo`（**默认**，https://studio.tripo3d.com/；可切 `blender` / `import` 可选策略）
- Handoff: `{ "mode": "manual", "source": "" }`（生产人工下载 GLB；CI 用 `fixture`，上游落盘用 `file`；`ASSETCTL_TRIPO_HANDOFF` / `ASSETCTL_TRIPO_SOURCE` 可覆盖）
- Config: `assets/hero-knight/asset.json`
- Reference: `raw/hero-knight.png`（`assetctl reference prepare` 生成）
- Raw model: `raw/hero-knight.glb`（tripo/import 策略产物）
- Rig: Blender `humanoid_rig.py` / Tripo 自动绑定
- Triangles: 18432（预算 `maxTriangles: 30000`）
- Textures: `<id>_basecolor / _normal / _roughness / _metallic / _ao`（≤ 2048）
- Animations: `hero-knight-idle`(loop)、`hero-knight-walk`(loop)、`hero-knight-attack`(once)

## 交付物（build/hero-knight/）

- `hero-knight.fbx` — 阶段 5 导出（`-Z forward / Y up / scale 1.0`）
- `hero-knight_basecolor.png` 等 PBR 贴图 — 阶段 3/4
- `preview.png` — 阶段 6 渲染合成
- `manifest.json` — 六阶段状态（全 `ok`）
- `.humanoid` — 人形标记（供 `assetctl validate --expect-humanoid`）

## DoD 勾选

- [x] 命名约定（小写连字符、贴图后缀 `_basecolor/_normal/...`）
- [x] 六阶段全部跑完，`manifest.json` 六项全 `ok`
- [x] PBR 贴图齐全（basecolor / normal / roughness / metallic / ao）
- [x] `-Z forward / Y up / scale 1.0`（`asset.json.axes`）
- [x] 面数预算达标（≤ `budget.maxTriangles`）
- [x] `rig_check.py` 门禁通过（humanoid=true 时由 runner 在阶段 5 后自动跑）
- [x] `build/<id>/manifest.json` 六阶段全 `ok`（`assetctl validate` 的 DoD 门禁）
- [x] `assetctl validate build/hero-knight --expect-humanoid --expect-textures` 通过
- [x] `assetctl doctor` 工具链报告通过
- [x] Unity 导入验证：无粉材质、无 T-pose、Avatar 映射无红项、Prefab + Animator 可用
- [x] Unity Import Settings 勾选 **Bake Axis Conversion**（ACT `"Unity"` 档）

## 测试 / CI

`go test ./...` 本身就是完整的自动化验证（无需安装 Blender 5.2）：`internal/runner` 的 `ScriptLauncher` 运行 `testing/fake_blender.py`（纯标准库 `bpy` / `mathutils` 模拟器，写 JSON 快照 + 真实 PNG）端到端跑通六阶段。`scripts/verify.py` 一键跑 `go build` / `go vet` / `go test` / `py_compile` / `python -m unittest discover -s testing`；本仓库不使用 CI 服务，`assetctl verify` 在本地跑。默认供应商在无人值守场景用 `--handoff fixture`。

## 命令回放

```bash
go run ./cmd/assetctl doctor
go run ./cmd/assetctl init hero-knight --humanoid --kind character   # 默认 strategy=tripo、handoff.mode=manual
# 生产：在 https://studio.tripo3d.com/ 下载 GLB 到 raw/hero-knight.glb 后：
go run ./cmd/assetctl run hero-knight --from modeling
# CI 无人值守：go run ./cmd/assetctl init hero-knight --handoff fixture && go run ./cmd/assetctl run hero-knight
go run ./cmd/assetctl validate build/hero-knight --expect-humanoid --expect-textures
go test ./...
```
