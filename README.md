# 3D Game Asset Pipeline (Tripo → Blender → Python → Unity)

[![Blender 5.2](https://img.shields.io/badge/Blender-5.2-orange)](https://www.blender.org/)
[![Tripo](https://img.shields.io/badge/Tripo-studio.tripo3d.com-blue)](https://studio.tripo3d.com/)
[![Python 3.14](https://img.shields.io/badge/Python-3.14-yellow)](https://www.python.org/)

用 **浏览器 Tripo**（图生 3D）+ **本机 Blender 5.2** + **Python** 制作提供给 **Unity** 的 3D 游戏资产，覆盖人形角色、道具，以及角色动画（自动绑定 / 重定向）。

> Tripo 入口：**https://studio.tripo3d.com/**

## 为什么存在

把「一张图 → Unity 里能跑的角色」这件事标准化：固定工具版本、固定坐标系、固定命名与校验，避免每次手动踩坑。

## 流水线一览

```
参考图 ──Python 预处理──> Tripo 图生3D ──GLB──> Blender 5.2 清洗/减面/绑定校验 ──FBX+PBR──> Unity
                                                          │
                                                   动画：Tripo 自动绑定 / Blender 重定向 Humanoid
```

| 阶段 | 工具 | 产物 | 文档 |
|------|------|------|------|
| 0 总览 | — | — | [docs/00-overview.md](docs/00-overview.md) |
| 1 参考图 + 生成 | Python + Tripo | 归一化图 + GLB | [docs/01-tripo-image-to-3d.md](docs/01-tripo-image-to-3d.md) |
| 2 清洗导出 | Blender 5.2 | FBX + 贴图 | [docs/02-blender-cleanup.md](docs/02-blender-cleanup.md) |
| 3 Python 工具 | Python 3.14 | 预处理 / 校验结果 | [docs/03-python-automation.md](docs/03-python-automation.md) |
| 4 人形动画 | Tripo + Blender | 绑定 + AnimClip | [docs/04-humanoid-animation.md](docs/04-humanoid-animation.md) |
| 5 Unity 导入 | Unity (URP) | Prefab + Animator | [docs/05-unity-import.md](docs/05-unity-import.md) |
| 命名规范 | — | — | [docs/06-naming-conventions.md](docs/06-naming-conventions.md) |

## 快速开始

```bash
# 1. 归一化参考图（主体单一、正面、干净背景）
python scripts/python/prepare_reference.py concept.png raw/ --size 1024

# 2. 打开 https://studio.tripo3d.com/ ，Image-to-3D 上传 raw/concept.png，下载 GLB 到 raw/hero.glb

# 3. Blender 5.2 无头导出 Unity FBX
blender --background --python scripts/blender/tripo_to_unity.py -- \
  --input raw/hero.glb --output build/hero/hero.fbx

# 4. 校验并导入 Unity
python scripts/python/validate_assets.py build/hero --expect-humanoid
```

## 仓库结构

- `AGENTS.md` — 版本与 URL 的硬性约束（**Blender 5.2 / Tripo URL** 记在这里）。
- `SKILL.md` — 可复用 Skill，其它项目可直接引用。
- `docs/` — 分阶段流水线文档。
- `scripts/blender/` — Blender `bpy` 脚本。
- `scripts/python/` — 纯 Python 工具。
- `examples/` — 配置与清单示例。

## License

MIT
