# 61 · 语言分工（Go / Blender 内置 Python / 系统 Python）

> 契约见 [SPEC.md §2](../SPEC.md)（硬约束）与 [AGENTS.md](../AGENTS.md)（版本约束）。本文解释**为什么这样分**，以及统一调用约定。

## 1. 三种语言、三种职责

| 语言 / 运行时 | 版本 | 用途 | 是否进交付流水线 |
|---------------|------|------|------------------|
| **Go** | **1.27** | 我们自己写的代码：CLI `assetctl`、编排器、策略注册表（`internal/strategy/`）、配置（`internal/config/` 的 `asset.json`）、校验、参考图处理 | ✅ **是，主体** |
| **Blender 内置 Python（`bpy`）** | 随 **Blender 5.2** 自带 | Blender 内部所有阶段的 `stage_*.py` / `rig_check.py` / `humanoid_rig.py` | ✅ **是，阶段实现** |
| **系统 Python** | **3.14**（本机 3.14.3） | **仅开发机上的辅助检查**：临时看 JSON、核对文件、跑一次性脚本 | ❌ 否，仅辅助 |

一句话：**Go 1.27 管编排与一切仓库自有逻辑，Blender 5.2 自带 Python 管 Blender 内部操作，系统 Python 3.14 只在开发机上帮忙看看，不进交付。**

## 2. Go 1.27：零第三方依赖

SPEC.md §2：我们自己写的代码 = Go 1.27，覆盖 CLI / 编排器 / 策略注册表 / 配置 / 校验 / 参考图处理，且 **零第三方依赖（仅标准库）**。

| 模块 | 职责 | 标准库 |
|------|------|--------|
| `assetctl`（CLI） | 解析子命令、参数、退出码 | `flag` / `os` |
| 编排器（runner） | 依次跑六阶段、断点续跑、写 `manifest.json` | `os/exec` / `path/filepath` |
| `internal/strategy/` | `Strategy` 接口、三个内置策略、`Register()` 注册表 | 标准库 |
| `internal/config/` | `asset.json` 读写（`encoding/json`）、`Validate()` | `encoding/json` / `regexp` |
| 校验 | `assetctl validate` 检查命名、贴图、骨骼标记 | `os` / `path/filepath` |
| 参考图处理 | `assetctl reference prepare`：正方形归一化、中性背景 | `image` / `image/png`（标准库） |

为什么 Go：
- **单二进制分发**：美术机 / CI 上丢一个 `assetctl` 就能跑，不用装 Python 环境、不用管依赖。
- **零第三方依赖**：`go.mod` 只有 `go 1.27`，供应链干净，便于 MIT 许可与离线构建。
- **编排天然**：`os/exec` 调 Blender、并发 / 超时 / 退出码 / 日志聚合都直接。
- **类型安全**：`asset.json` 的 schema（`Axes` / `Budget` / `Config`）编译期校验，`Validate()` 再拒绝非法坐标 / URL / 版本。

## 3. Blender 内部：原生内置 Python + `bpy`

SPEC.md §2 与用户原话：**「他原生代码用什么就用什么」**。Blender 5.2 自带什么 Python，就用它，不假设系统 Python，也不在 Blender 里 `pip install`。

| 约束 | 含义 |
|------|------|
| 用 Blender 5.2 **内置** Python | 阶段脚本只依赖 `bpy` + 该 Python 自带的标准库 |
| 不假设系统 Python 3.14 | Blender 进程里的 `python` 不是开发机那个；脚本不能 `import` 系统里装过的包 |
| 不在 Blender 里 `pip install` | 不引入 Pillow / numpy 等第三方；需要图像操作时用 `bpy` / 内置 `bpy.data.images` 或交给 Go 侧参考图处理 |
| `--factory-startup` | 所有脚本必须容忍工厂启动（不依赖用户偏好、插件、主题、已装 Add-on） |

为什么这样分：
- `bpy` **只在 Blender 进程里存在**，系统 Python 里没有；反过来 Blender 内置 Python 也未必有开发机装的那套包。两边强行打通只会制造「我本地能跑、CI 跑不了」。
- 阶段脚本（减面、UV、烘焙、导出、渲染）本质是**调用 Blender 算子**（`bpy.ops.*`）与读写 `bpy.data`，用 `bpy` 是最短路径。
- 图像预处理这类「不进 Blender 也行」的活交给 Go 标准库做，避免把 Pillow 之类的依赖塞进 Blender 环境。

阶段脚本清单（均在 Blender 5.2 内置 Python 中运行）：

```
scripts/blender/stage_01_modeling.py
scripts/blender/stage_02_topology_uv.py
scripts/blender/stage_03_bake.py
scripts/blender/stage_04_pbr.py
scripts/blender/stage_05_unity_export.py
scripts/blender/stage_06_render.py
scripts/blender/rig_check.py            # 人形门禁
scripts/blender/humanoid_rig.py         # 手动绑定路线（见 50）
```

## 4. 系统 Python 3.14：仅开发机辅助检查

| 项 | 说明 |
|----|------|
| 版本 | 本机 **3.14.3**，文档写「系统 Python 3.14」 |
| 用途 | 开发机上临时核对 `asset.json` / `manifest.json`、一次性文件清点、肉眼比对 |
| 边界 | **不进入交付流水线**：`assetctl run` 不会调用它，CI 不依赖它，脚本不写在 `scripts/` 里作为阶段实现 |

> 历史说明：旧仓库曾有 `scripts/python/prepare_reference.py`（用 Pillow）与 `scripts/python/validate_assets.py`，那是旧单体架构。新架构把参考图处理收敛到 `assetctl reference prepare`（Go 1.27 标准库）、把校验收敛到 `assetctl validate`（Go 1.27），系统 Python 只剩「开发机辅助」角色。本套文档只以 SPEC.md 为准，不要求安装 Pillow。

## 5. `blender --background --factory-startup --python` 调用约定

Go runner 统一按此约定拉起 Blender 5.2（SPEC.md §2/§5）：

```
blender --background --factory-startup --python <script> -- \
  --asset <id> --config <asset.json 路径> --in <上一阶段产物> --out <本阶段产物> [--blend-out <中间 .blend>]
```

拆解：

| 片段 | 作用 |
|------|------|
| `blender` | Blender **5.2** 可执行文件 |
| `--background` | 无头运行，不弹 GUI，适合 CI / 批量 |
| `--factory-startup` | 工厂启动，忽略用户偏好与插件，保证可重复（所有脚本必须容忍） |
| `--python <script>` | 在 Blender 内置 Python 里执行该 `bpy` 脚本 |
| `--` | **Blender 参数与脚本自有参数的分隔符**；脚本用 `sys.argv[sys.argv.index("--")+1:]` 取后半段 |
| `--asset / --config / --in / --out` | Go runner 传给每个阶段的统一文件契约（SPEC.md §5） |
| `[--blend-out]` | 可选中间 `.blend` |

退出码：`0` 成功；非 `0` 失败，错误写 stderr。脚本必须容忍 `--factory-startup`。

`assetctl` 也提供透传入口（[80](80-cli-reference.md)）：

```bash
assetctl blender scripts/blender/rig_check.py -- --input build/hero-knight/hero-knight.fbx
```

## 6. 为什么这样分（设计理由汇总）

| 决策 | 理由 |
|------|------|
| 编排 / CLI / 配置 / 校验 / 参考图 → **Go 1.27** | 单二进制、零第三方依赖、类型安全、适合 CI 与 MIT 许可；不碰 `bpy` |
| Blender 内部操作 → **Blender 5.2 内置 Python（`bpy`）** | `bpy` 只在 Blender 进程里存在；算子调用是最短路径；「他原生代码用什么就用什么」 |
| 不在 Blender 里 `pip install` / 不假设系统 Python | 避免两套 Python 环境耦合，保证 `--factory-startup` 可重复、CI 与本机一致 |
| 系统 Python 3.14 仅开发机辅助 | 不进交付，避免把 Pillow 等依赖带回流水线 |
| 统一 `--background --factory-startup --python ... --` 约定 | 一个调用形态覆盖六阶段 + rig 校验，Go 侧只管传 `--asset/--config/--in/--out` 与收退出码 |
| 版本硬锁 Go 1.27 / Blender 5.2 / Tripo https://studio.tripo3d.com/ | 见 AGENTS.md；升级先改约束文件再改脚本/文档 |

- [ ] `go.mod` 为 `go 1.27`，`internal/` 无第三方 import
- [ ] 阶段脚本只依赖 `bpy` + Blender 5.2 内置标准库
- [ ] 无 `pip install` 出现在交付流程
- [ ] 所有 Blender 调用带 `--background --factory-startup --python ... --`
- [ ] 系统 Python 3.14 只出现在「开发机辅助」描述里
