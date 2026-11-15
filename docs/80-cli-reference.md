# 80 · `assetctl` CLI 完整参考

> 命令表**严格照 [SPEC.md §8](../SPEC.md)**。`assetctl` 用 **Go 1.27** 编写，零第三方依赖（仅标准库），是整条流水线的唯一入口。
> 六阶段顺序固定：`modeling → topology-uv → bake → pbr → unity-export → render`（`assetctl stages` 可列出）。

## 1. 命令总表（SPEC.md §8，逐字对齐）

```
assetctl version
assetctl strategies                          # 列出策略 / 供应商，标注 DEFAULT（tripo）
assetctl stages                              # 列出六阶段
assetctl init <asset-id> [--strategy <id>] [--supplier <id>] [--handoff <manual|file|fixture>] [--source <path>] [--kind character] [--humanoid]
assetctl run <asset-id> [--strategy <id>] [--from <stage>] [--only <stage,...>] [--dry-run]
assetctl reference prepare <in.png> <outdir> [--size 1024] [--bg 808080]
assetctl validate <build-dir> [--expect-humanoid] [--expect-textures]
assetctl blender <script> [-- --args...]     # 透传 Blender 5.2
assetctl doctor                              # 工具链报告（v2 新增）
assetctl verify                              # 一键生产级验证（9 步，Go 1.27 实现，v2.1）
```

`assetctl run` 依次执行六阶段，写 `build/<id>/manifest.json`；任一阶段失败即停止并记录状态。`--from` / `--only` 支持断点续跑（Tripo 半自动场景必需，尤其 `--from topology-uv`）。`assetctl init` 默认 `strategy=tripo`、`handoff.mode=manual`；`--supplier` 为 `--strategy` 的别名。

## 2. 全局约定

| 项 | 值 |
|----|----|
| 二进制 | `assetctl`（Go 1.27，单文件，标准库） |
| 默认策略 / 供应商 | `tripo`（`IsDefault() == true`、`config.Default().Strategy == "tripo"`，`assetctl strategies` 会标 `DEFAULT`）；`assetctl init` 默认 `--strategy tripo` |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr；`manual` handoff 缺文件时为 `3`（`ErrManualStepRequired`） |
| 配置 | `assets/<id>/asset.json`（`encoding/json`，字段见 [SPEC.md §7](../SPEC.md)） |
| manifest | `build/<id>/manifest.json`（阶段状态 + DoD 结果） |
| Blender 调用 | `blender --background --factory-startup --python <script> -- --asset ... --config ... --in ... --out ...`（见 [61](61-language-split.md)） |

## 3. 子命令逐个详解

### 3.1 `assetctl version`

打印工具与锁定的工具链版本。

```bash
assetctl version
```

输出应包含 **Go 1.27**、**Blender 5.2**、Tripo **https://studio.tripo3d.com/** 等版本/URL 声明，便于排障时确认基线。

### 3.2 `assetctl strategies`

列出已注册的策略，并标注哪个是 `DEFAULT`。

```bash
assetctl strategies
```

| 输出项 | 含义 |
|--------|------|
| `tripo` | 浏览器图生 3D（**DEFAULT 默认供应商**，`IsDefault() == true`，半自动、人在回路） |
| `blender` | 参数化建模（可选策略，`IsDefault() == false`） |
| `import` | 导入已有模型（可选策略） |

> 「供应商（Supplier）== 建模策略（Strategy）」：这里列出的每一项既是策略也是供应商，Go `Strategy` 类型不改名。

新策略在 `internal/strategy/` 实现 `Strategy` 接口并 `Register()` 后会出现在这里（[10](10-strategies.md)）。

### 3.3 `assetctl stages`

列出固定六阶段及其顺序。

```bash
assetctl stages
# 1 modeling
# 2 topology-uv
# 3 bake
# 4 pbr
# 5 unity-export
# 6 render
```

### 3.4 `assetctl init <asset-id> [flags]`

生成一份带默认值的 `assets/<id>/asset.json`（[SPEC.md §7](../SPEC.md) 字段）。

```bash
assetctl init hero-knight                                        # 默认 strategy=tripo、handoff.mode=manual
assetctl init hero-knight --handoff fixture                        # CI 无人值守（默认供应商也全自动）
assetctl init hero-knight --handoff file --source D:/art/knight.glb  # 复制上游导出
assetctl init hero-knight --strategy blender --kind character --humanoid  # 切到可选策略
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `<asset-id>` | 必填 | 小写连字符，如 `hero-knight`；不合法时 `config.Validate()` 拒绝 |
| `--strategy <id>` | `tripo` | `tripo`（默认供应商） / `blender` / `import` |
| `--supplier <id>` | 同 `--strategy` | `--strategy` 的别名（供应商==策略） |
| `--handoff <mode>` | `manual` | `manual` / `file` / `fixture`，写入 `asset.json.handoff.mode` |
| `--source <path>` | `""` | 写入 `asset.json.handoff.source`（`file` 模式来源） / `import` 策略来源 |
| `--kind <kind>` | `prop` | `character` / `prop` 等，决定面数预算与门禁 |
| `--humanoid` | `false` | 人形角色：阶段 5 后加跑 `rig_check.py` |

生成的 `asset.json` 含：`blenderVersion: "5.2"`、`tripoUrl: "https://studio.tripo3d.com/"`、`targetEngine: "unity-urp"`、`axes: {forward:"-Z", up:"Y", scale:1.0}`、`budget: {maxTriangles:30000, maxTextureSize:2048}`、`handoff: {mode:"manual", source:""}` 等默认值（[SPEC.md §7](../SPEC.md)）。

### 3.5 `assetctl run <asset-id> [flags]`（核心）

依次执行六阶段，写 `build/<id>/manifest.json`。

```bash
assetctl run hero-knight                                  # 默认 strategy=tripo
assetctl run hero-knight --from modeling                   # 人工下载 GLB 后续跑
assetctl run hero-knight --from topology-uv                # 尤其常用：跳过建模，从拓扑/UV 续跑
assetctl run hero-knight --only topology-uv,bake
assetctl run hero-knight --from bake --dry-run
ASSETCTL_TRIPO_HANDOFF=fixture assetctl run hero-knight    # CI 强制无人值守
```

| 参数 | 说明 |
|------|------|
| `<asset-id>` | 必填，对应 `assets/<id>/asset.json` |
| `--strategy <id>` | 覆盖 `asset.json.strategy`（切换建模策略 / 供应商） |
| `--from <stage>` | **从指定阶段开始**执行到结尾（断点续跑，尤其 `--from topology-uv`） |
| `--only <stage,...>` | **只执行**列出的阶段（逗号分隔） |
| `--dry-run` | 演练：打印将执行的阶段与命令，不真正跑、不写产物 |

环境变量覆盖（handoff，见 [SPEC.md §3A](../SPEC.md)）：

| 变量 | 作用 |
|------|------|
| `ASSETCTL_TRIPO_HANDOFF` | 覆盖 `handoff.mode`（如设为 `fixture` 强制无人值守） |
| `ASSETCTL_TRIPO_SOURCE` | 覆盖 `handoff.source`（`file` 模式的来源路径） |

`Acquire` 优先级：已存在 `raw/<id>.{glb,gltf,fbx,obj}` > handoff mode > manual 指引（缺文件 exit `3` / `ErrManualStepRequired`）。

行为：
- 每阶段按 [SPEC.md §5](../SPEC.md) 的文件契约传 `--asset / --config / --in / --out` 给 `bpy` 脚本。
- 任一阶段非 0 退出即**停止**，并在 `manifest.json` 记录失败状态；修好后用 `--from <失败阶段>` 续跑。
- 每个真实阶段执行后，runner 校验该阶段声明的 `--out` 文件确实存在（防静默失败）。
- `humanoid: true` 时阶段 5 之后额外跑 `scripts/blender/rig_check.py`。

#### 断点续跑（`--from` / `--only` / `--dry-run`）

`tripo` 是默认供应商（半自动、人在回路）：`manual` 模式下人工在 https://studio.tripo3d.com/ 下载 GLB 前阶段 1 无法完成；`file` / `fixture` 模式则无需人工。典型流程：

```bash
# ① 先跑一次：manual 模式 GLB 未就位 → 停在 modeling 并给指引（exit 3）
assetctl run hero-knight

# ② 人工下载 GLB 到 raw/hero-knight.glb 后，从 modeling 续跑全程
assetctl run hero-knight --from modeling

# ③ 只补跑建模这一阶段
assetctl run hero-knight --only modeling

# ④ 续跑前先演练，确认阶段与参数
assetctl run hero-knight --from modeling --dry-run

# ⑤ 尤其常用：跳过建模，直接从拓扑/UV 续跑
assetctl run hero-knight --from topology-uv
```

`--from` 与 `--only` 可配合 `--dry-run`；断点续跑依赖 `work/<id>/NN-<stage>.blend` 中间产物与 `manifest.json` 里记录的已完成状态（[10](10-strategies.md)、[42](42-stage-05-unity-export.md)）。

### 3.6 `assetctl reference prepare <in.png> <outdir> [flags]`

参考图归一化（Go 1.27 标准库实现，零第三方依赖）：按比例缩放 → 居中放到正方形画布 → 保存 PNG。

```bash
assetctl reference prepare concept.png raw/ --size 1024 --bg 808080
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `<in.png>` | 必填 | 输入参考图 |
| `<outdir>` | 必填 | 输出目录（一般 `raw/`） |
| `--size <n>` | `1024` | 输出正方形边长 |
| `--bg <hex>` | `808080` | 中性背景色 |

产出 `raw/<id>.png`，供 Tripo **Image-to-3D** 上传（主体单一、正面、干净背景、正方形，见 [20](20-stage-01-modeling.md)）。

### 3.7 `assetctl validate <build-dir> [flags]`

交付前 DoD 机器校验（只读，不破坏产物），详见 [70](70-naming-and-dod.md)。

```bash
assetctl validate build/hero-knight
assetctl validate build/hero-knight --expect-humanoid
assetctl validate build/hero-knight --expect-textures
assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

| 参数 | 说明 |
|------|------|
| `<build-dir>` | 必填，如 `build/hero-knight` |
| `--expect-humanoid` | 额外要求人形标记 / `rig_check` 结果 |
| `--expect-textures` | 逐张核对 `asset.json.textures` 声明的贴图后缀 |

检查：`.fbx` 存在、资产 ID 小写连字符、贴图后缀（`_basecolor`/`_normal`/`_roughness`/`_metallic`/`_ao`）、`manifest.json` 状态一致，且**额外要求 `build/<id>/manifest.json` 六阶段全为 `ok`**（DoD 门禁）。退出码 `0` 通过、非 `0` 列出问题。

### 3.8 `assetctl blender <script> [-- --args...]`

透传调用 **Blender 5.2**（`--background --factory-startup --python`）。用于单独跑 `rig_check.py` 或调试某个阶段脚本。

```bash
assetctl blender scripts/blender/rig_check.py -- --input build/hero-knight/hero-knight.fbx
assetctl blender scripts/blender/stage_02_topology_uv.py -- \
  --asset hero-knight --config assets/hero-knight/asset.json \
  --in work/hero-knight/01-modeling.blend --out work/hero-knight/02-topology-uv.blend
```

`--` 之后的参数原样传给脚本（脚本用 `sys.argv[sys.argv.index("--")+1:]` 取），与 Go runner 内部调用约定一致（[61](61-language-split.md)）。

### 3.9 `assetctl doctor`（v2 新增）

生产 onboarding 的工具链自检报告，只读、不产出资产。打印：

- Go 版本（基线 **1.27**）；
- **Blender 5.2** 是否存在、以及 `ASSETCTL_BLENDER` 覆盖值；
- 已注册的供应商 / 策略 + 哪个是 **DEFAULT**（`tripo`）；
- Tripo URL **https://studio.tripo3d.com/**；
- 坐标轴契约 `-Z forward / Y up / scale 1.0`；
- `scripts/blender/` 六阶段脚本是否齐备；
- 综合退出码（用于 CI / 人工排查环境）。

```bash
assetctl doctor
```

### 3.10 `assetctl verify`（v2.1 新增，一键验证）

Go 1.27 实现（`internal/verify/verify.go`），**逻辑不在系统 Python 中**；`python scripts/verify.py` 与 `make verify` 都只是兼容转调。9 步按序执行、任一失败即停：

1. 仓库布局（10 个 `scripts/blender/*.py` + `testing/` harness）
2. `go.mod` 零第三方依赖 + `go 1.27` 校验
3. `go build ./...`
4. `go vet ./...`
5. `go test ./... -count=1 -coverprofile=coverage.out`（Unix 额外 `-race`；Windows 因 cgo 限制不启用 race；含非 dry-run 六阶段 E2E）
6. `py_compile` 全部 bpy / 测试脚本
7. `python -m unittest discover -s testing`（10 个 bpy 模块 + 契约 + mock_bpy）
8. `assetctl doctor`
9. CLI 冒烟（version / strategies / stages）

```bash
go run ./cmd/assetctl verify            # 推荐
make verify                            # 等价
python scripts/verify.py                # 兼容 shim
ASSETCTL_VERIFY_NO_RACE=1 make verify   # 跳过 race（无 gcc 的 Unix 环境）
```

`ASSETCTL_PYTHON` 可指定测试用解释器；CI（`.github/workflows/ci.yml`）在 ubuntu / macos / windows 三平台执行同一命令并上传覆盖率。

## 4. 退出码

| 退出码 | 含义 |
|--------|------|
| `0` | 成功 |
| `3` | `ErrManualStepRequired`：`manual` handoff 模式下 `raw/<id>.glb` 缺失，需要人工下载后 `--from modeling` 续跑 |
| 非 `0` | 失败，错误写 stderr（`assetctl run` 同时记录到 `manifest.json`） |

所有阶段脚本同样遵循 `0` 成功 / 非 `0` 失败（[SPEC.md §5](../SPEC.md)）。CI 直接以退出码作门禁。

## 5. `manifest.json` 结构示例

`assetctl run` 写入 `build/<id>/manifest.json`，记录六阶段状态 + DoD 结果（[SPEC.md §6](../SPEC.md)）：

```json
{
  "asset": "hero-knight",
  "strategy": "tripo",
  "handoff": { "mode": "fixture", "source": "" },
  "humanoid": true,
  "stages": {
    "modeling":      { "status": "ok", "out": "work/hero-knight/01-modeling.blend" },
    "topology-uv":   { "status": "ok", "out": "work/hero-knight/02-topology-uv.blend" },
    "bake":          { "status": "ok", "out": "work/hero-knight/03-bake.blend" },
    "pbr":           { "status": "ok", "out": "work/hero-knight/04-pbr.blend" },
    "unity-export":  { "status": "ok", "out": "build/hero-knight/hero-knight.fbx" },
    "render":        { "status": "ok", "out": "build/hero-knight/preview.png" }
  },
  "dod": {
    "validate": "pass",
    "rigCheck": "pass"
  }
}
```

- `stages.<id>.status`：`ok` / `failed` / `skipped`（`--only` / `--from` 时未跑的阶段）。
- `dod`：`assetctl validate` 与人形门禁 `rig_check.py` 的结果。
- 任一阶段失败：`status: "failed"`，`assetctl run` 停止，修好后 `--from <stage>` 续跑会更新该条目。

完整的 `build/<id>/` 交付目录：

```
build/hero-knight/
├── hero-knight.fbx
├── hero-knight_basecolor.png
├── hero-knight_normal.png
├── hero-knight_roughness.png
├── hero-knight_metallic.png
├── hero-knight_ao.png          # 可选
├── preview.png
└── manifest.json
```

## 6. CI 集成示例

**`go test ./...` 本身就是完整的自动化验证**：`internal/runner` 的 `Launcher` 抽象在生产用 `CommandLauncher`（真实 Blender 5.2），在 CI / 测试用 `ScriptLauncher` 运行 `testing/fake_blender.py`（纯标准库 `bpy` + `mathutils` 模拟器）。模拟器通过写入 `.blend` / `.fbx` 文件的 JSON 快照在多个阶段进程间持久化场景状态、并用标准库 `zlib` / `struct` 写出真实 PNG，因此六阶段端到端跑通，**无需安装 Blender 5.2**。`scripts/verify.py` 可一键跑 `go build` / `go vet` / `go test`（unix 上带 `-race -cover`，Windows 上普通）/ `py_compile` / `python -m unittest discover -s testing`；`Makefile` 的 `verify` target 调用它；`.github/workflows/ci.yml` 用 3-OS 矩阵（ubuntu / macos / windows）+ Go 1.27 + setup-python。

```yaml
# .github/workflows/ci.yml（示意，3-OS 矩阵）
name: ci
on: [push, pull_request]
jobs:
  verify:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
      - uses: actions/setup-python@v5
      - name: Full automated verification (no Blender needed)
        run: go test ./...
      - name: One-shot verify
        run: python scripts/verify.py
```

纯命令行等价（本地 / 任意 shell，用默认供应商 + `--handoff fixture` 无人值守）：

```bash
go test ./...                                                   # 完整自动化验证，无需 Blender
go build -o assetctl ./cmd/assetctl
./assetctl version
./assetctl doctor
./assetctl strategies
./assetctl init hero-knight --handoff fixture --kind character --humanoid   # 默认 strategy=tripo + fixture 无人值守
./assetctl run hero-knight
./assetctl validate build/hero-knight --expect-humanoid --expect-textures || exit 1
```

要点：
- **CI 用 `--handoff fixture`（或 `ASSETCTL_TRIPO_HANDOFF=fixture`）才能无人值守**——默认供应商 `tripo`（半自动）在 `manual` 模式下会缺 GLB 而 exit `3`；`fixture` 生成确定性最小合法 GLB，使默认供应商在 CI 中全自动。
- `validate` 放在最后一步，非 0 直接让 job 失败；它额外检查 `manifest.json` 六阶段全 `ok`（DoD 门禁）。
- 版本/URL/坐标基线由 `assetctl version` / `assetctl doctor` 与 `config.Validate()` 双重保证（[61](61-language-split.md)）。

## 7. 速查卡

| 我想… | 命令 |
|-------|------|
| 看版本 | `assetctl version` |
| 看有哪些策略 / 供应商 | `assetctl strategies` |
| 看六阶段 | `assetctl stages` |
| 工具链自检 | `assetctl doctor` |
| 新建资产配置 | `assetctl init hero-knight --humanoid` |
| 新建无人值守配置 | `assetctl init hero-knight --handoff fixture` |
| 生成参考图 | `assetctl reference prepare in.png raw/ --size 1024` |
| 跑全流程 | `assetctl run hero-knight` |
| 默认供应商断点续跑 | `assetctl run hero-knight --from modeling` / `--from topology-uv` |
| 演练 | `assetctl run hero-knight --dry-run` |
| 单独跑脚本 | `assetctl blender scripts/blender/rig_check.py -- --input build/hero-knight/hero-knight.fbx` |
| 交付校验 | `assetctl validate build/hero-knight --expect-humanoid --expect-textures` |
