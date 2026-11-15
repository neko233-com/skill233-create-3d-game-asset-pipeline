# 03 · Python 自动化工具

> 本机 Python **3.14.3**。脚本用标准库 + Pillow（`pip install pillow`），不引入重依赖。
> 这些脚本在 Blender 之外运行。

## 1. `prepare_reference.py` — 参考图归一化

把任意图片转成 Tripo 友好的正方形图（居中、留白、中性背景）。

```bash
python scripts/python/prepare_reference.py concept.png raw/ --size 1024
```

参数：
- 位置参数：输入图、输出目录
- `--size`：输出边长（默认 1024）
- `--bg`：背景色（默认中性灰 `#808080`）

流程：按比例缩放 → 居中放到正方形画布 → 保存 PNG。

## 2. `validate_assets.py` — 交付校验

在导入 Unity 前检查资产目录是否完整。

```bash
python scripts/python/validate_assets.py build/hero --expect-humanoid
```

检查项：
- 存在 `.fbx`（或 `.glb` / `.obj`）模型文件
- 存在至少一张贴图（若资产标注需要贴图）
- PBR 贴图命名符合规范（basecolor/normal/roughness/metallic）
- `--expect-humanoid` 时，模型文件名或清单标注含骨骼标记
- 资产 ID 符合小写连字符命名

退出码：0 = 全部通过；非 0 = 列出问题。

## 安装依赖

```bash
python -m pip install pillow
```

## 设计原则

- 只读 + 输出报告，不破坏原始文件。
- 失败时给出明确、可执行的错误信息。
- 不依赖 Blender 运行时（Blender 内逻辑在 `scripts/blender/`）。
