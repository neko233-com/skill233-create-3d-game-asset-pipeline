# 80 · `assetctl` CLI 完整参考

> 命令表**严格照 [SPEC.md §8](../SPEC.md)**。`assetctl` 用 **Go 1.27** 编写，零第三方依赖（仅标准库），是整条流水线的唯一入口。
> 六阶段顺序固定：`modeling → topology-uv → bake → pbr → unity-export → render`（`assetctl stages` 可列出）。

## 1. 命令总表（SPEC.md §8，逐字对齐）

```
assetctl version
assetctl strategies                          # 列出策略，标注 DEFAULT
assetctl stages                              # 列出六阶段
assetctl init <asset-id> [--strategy blender] [--kind character] [--humanoid]
assetctl run <asset-id> [--strategy <id>] [--from <stage>] [--only <stage,...>] [--dry-run]
assetctl reference prepare <in.png> <outdir> [--size 1024] [--bg 808080]
assetctl validate <build-dir> [--expect-humanoid] [--expect-textures]
assetctl blender <script> [-- --args...]     # 透传 Blender 5.2
```

`assetctl run` 依次执行六阶段，写 `build/<id>/manifest.json`；任一阶段失败即停止并记录状态。`--from` / `--only` 支持断点续跑（Tripo 半自动场景必需）。

## 2. 全局约定

| 项 | 值 |
|----|----|
| 二进制 | `assetctl`（Go 1.27，单文件，标准库） |
| 默认策略 | `blender`（`assetctl strategies` 会标 `DEFAULT`） |
| 退出码 | `0` 成功；非 `0` 失败，错误写 stderr |
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
| `blender` | 参数化建模（**DEFAULT**，`IsDefault() == true`） |
| `tripo` | 浏览器图生 3D（半自动） |
| `import` | 导入已有模型 |

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
assetctl init hero-knight
assetctl init hero-knight --strategy tripo --kind character --humanoid
```

| 参数 | 默认 | 说明 |
|------|------|------|
| `<asset-id>` | 必填 | 小写连字符，如 `hero-knight`；不合法时 `config.Validate()` 拒绝 |
| `--strategy <id>` | `blender` | `blender` / `tripo` / `import` |
| `--kind <kind>` | `prop` | `character` / `prop` 等，决定面数预算与门禁 |
| `--humanoid` | `false` | 人形角色：阶段 5 后加跑 `rig_check.py` |

生成的 `asset.json` 含：`blenderVersion: "5.2"`、`tripoUrl: "https://studio.tripo3d.com/"`、`targetEngine: "unity-urp"`、`axes: {forward:"-Z", up:"Y", scale:1.0}`、`budget: {maxTriangles:30000, maxTextureSize:2048}` 等默认值。

### 3.5 `assetctl run <asset-id> [flags]`（核心）

依次执行六阶段，写 `build/<id>/manifest.json`。

```bash
assetctl run hero-knight
assetctl run hero-knight --strategy tripo
assetctl run hero-knight --strategy tripo --from modeling
assetctl run hero-knight --only topology-uv,bake
assetctl run hero-knight --from bake --dry-run
```

| 参数 | 说明 |
|------|------|
| `<asset-id>` | 必填，对应 `assets/<id>/asset.json` |
| `--strategy <id>` | 覆盖 `asset.json.strategy`（切换建模策略） |
| `--from <stage>` | **从指定阶段开始**执行到结尾（断点续跑） |
| `--only <stage,...>` | **只执行**列出的阶段（逗号分隔） |
| `--dry-run` | 演练：打印将执行的阶段与命令，不真正跑、不写产物 |

行为：
- 每阶段按 [SPEC.md §5](../SPEC.md) 的文件契约传 `--asset / --config / --in / --out` 给 `bpy` 脚本。
- 任一阶段非 0 退出即**停止**，并在 `manifest.json` 记录失败状态；修好后用 `--from <失败阶段>` 续跑。
- `humanoid: true` 时阶段 5 之后额外跑 `scripts/blender/rig_check.py`。

#### 断点续跑（`--from` / `--only` / `--dry-run`）

Tripo 是半自动策略，人工在 https://studio.tripo3d.com/ 下载 GLB 前阶段 1 无法完成。典型流程：

```bash
# ① 先跑一次：GLB 未就位 → 停在 modeling 并给指引
assetctl run hero-knight --strategy tripo

# ② 人工下载 GLB 到 raw/hero-knight.glb 后，从 modeling 续跑全程
assetctl run hero-knight --strategy tripo --from modeling

# ③ 只补跑建模这一阶段
assetctl run hero-knight --strategy tripo --only modeling

# ④ 续跑前先演练，确认阶段与参数
assetctl run hero-knight --strategy tripo --from modeling --dry-run
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

检查：`.fbx` 存在、资产 ID 小写连字符、贴图后缀（`_basecolor`/`_normal`/`_roughness`/`_metallic`/`_ao`）、`manifest.json` 状态一致。退出码 `0` 通过、非 `0` 列出问题。

### 3.8 `assetctl blender <script> [-- --args...]`

透传调用 **Blender 5.2**（`--background --factory-startup --python`）。用于单独跑 `rig_check.py` 或调试某个阶段脚本。

```bash
assetctl blender scripts/blender/rig_check.py -- --input build/hero-knight/hero-knight.fbx
assetctl blender scripts/blender/stage_02_topology_uv.py -- \
  --asset hero-knight --config assets/hero-knight/asset.json \
  --in work/hero-knight/01-modeling.blend --out work/hero-knight/02-topology-uv.blend
```

`--` 之后的参数原样传给脚本（脚本用 `sys.argv[sys.argv.index("--")+1:]` 取），与 Go runner 内部调用约定一致（[61](61-language-split.md)）。

## 4. 退出码

| 退出码 | 含义 |
|--------|------|
| `0` | 成功 |
| 非 `0` | 失败，错误写 stderr（`assetctl run` 同时记录到 `manifest.json`） |

所有阶段脚本同样遵循 `0` 成功 / 非 `0` 失败（[SPEC.md §5](../SPEC.md)）。CI 直接以退出码作门禁。

## 5. `manifest.json` 结构示例

`assetctl run` 写入 `build/<id>/manifest.json`，记录六阶段状态 + DoD 结果（[SPEC.md §6](../SPEC.md)）：

```json
{
  "asset": "hero-knight",
  "strategy": "blender",
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

`assetctl validate` 以退出码作门禁，适合 GitHub Actions / 任意 CI：

```yaml
# .github/workflows/asset-pipeline.yml（示意）
name: asset-pipeline
on: [push, pull_request]
jobs:
  build-and-validate:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
      - name: Build assetctl
        run: go build -o assetctl ./cmd/assetctl
      - name: Confirm toolchain baseline
        run: ./assetctl version
      - name: Run full pipeline (default blender strategy)
        run: ./assetctl run hero-knight
      - name: Validate DoD (humanoid + textures)
        run: ./assetctl validate build/hero-knight --expect-humanoid --expect-textures
```

纯命令行等价（本地 / 任意 shell）：

```bash
go build -o assetctl ./cmd/assetctl
./assetctl version
./assetctl strategies
./assetctl stages
./assetctl init hero-knight --strategy blender --kind character --humanoid
./assetctl reference prepare concept.png raw/ --size 1024 --bg 808080
./assetctl run hero-knight
./assetctl validate build/hero-knight --expect-humanoid --expect-textures || exit 1
```

要点：
- CI 用**默认 `blender` 策略**（全自动、无人在回路）才能无人值守；`tripo` 半自动不适合 CI 全量跑，可只跑 `--from` 续跑或单独校验。
- `validate` 放在最后一步，非 0 直接让 job 失败。
- 版本/URL/坐标基线由 `assetctl version` 与 `config.Validate()` 双重保证（[61](61-language-split.md)）。

## 7. 速查卡

| 我想… | 命令 |
|-------|------|
| 看版本 | `assetctl version` |
| 看有哪些策略 | `assetctl strategies` |
| 看六阶段 | `assetctl stages` |
| 新建资产配置 | `assetctl init hero-knight --humanoid` |
| 生成参考图 | `assetctl reference prepare in.png raw/ --size 1024` |
| 跑全流程 | `assetctl run hero-knight` |
| 切 Tripo 并续跑 | `assetctl run hero-knight --strategy tripo --from modeling` |
| 演练 | `assetctl run hero-knight --dry-run` |
| 单独跑脚本 | `assetctl blender scripts/blender/rig_check.py -- --input build/hero-knight/hero-knight.fbx` |
| 交付校验 | `assetctl validate build/hero-knight --expect-humanoid --expect-textures` |
