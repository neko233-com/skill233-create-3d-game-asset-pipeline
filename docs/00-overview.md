# 00 · 流水线总览

## 目标

把「一张参考图」变成 Unity 中可直接使用的 3D 游戏资产（人形角色 / 道具 / 动画）。

## 工具链（版本锁定）

| 工具 | 版本 / 地址 | 角色 |
|------|-------------|------|
| Tripo | https://studio.tripo3d.com/ | 浏览器端图生 3D / 文生 3D / 自动绑定与动画 |
| Blender | 5.2（本机） | 导入、清理、减面、UV/贴图整理、骨骼校验、导出 |
| Python | 3.14.3（本机） | 参考图预处理、交付校验 |
| Unity | URP 优先 | 最终导入、Prefab、Animator |

版本和 URL 的唯一真相来源是 [AGENTS.md](../AGENTS.md)；任何变更先改该文件。

## 全流程

```
[参考图] ── prepare_reference.py ──> [归一化图]
                                        │
                                   Tripo Image-to-3D
                                   (studio.tripo3d.com)
                                        │
                                     [GLB]
                                        │
                          blender tripo_to_unity.py
                          (清理/减面/坐标系/贴图)
                                        │
                                  [FBX + PBR 贴图]
                                        │
                   rig_check.py (人形) + validate_assets.py
                                        │
                                   Unity 导入
                                        │
                              [Prefab + Animator + Clip]
```

## 三种资产类型

1. **道具 / 静态物** — Tripo 生成 → Blender 清理减面 → 导出 FBX。无骨骼。
2. **人形角色（无动画）** — Tripo 生成 → Blender 清理 → 自动/手动绑定 → Humanoid 骨骼校验 → 导出。
3. **人形角色（带动画）** — 在 2 的基础上，用 Tripo 自动绑定/动画或 Blender 重定向，导出 idle/walk/attack 等独立 AnimClip。

## 交付清单（DoD）

- [ ] 资产 ID 符合命名规范（小写连字符）
- [ ] 贴图齐全（basecolor / normal / roughness / metallic，按需）
- [ ] 坐标系 `-Z forward / Y up`，缩放 1.0
- [ ] 面数符合预算（角色 / 道具分别见 02 文档）
- [ ] 人形骨骼通过 `rig_check.py`
- [ ] `validate_assets.py` 通过
- [ ] Unity 中贴图/骨骼/动画均正常，无粉材质、无 T-pose 残留
