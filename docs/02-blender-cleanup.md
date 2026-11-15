# 02 · Blender 5.2 清洗与导出

> 所有脚本以本机 **Blender 5.2** 为基准。无头调用：
> `blender --background --python scripts/blender/tripo_to_unity.py -- --input raw/x.glb --output build/x/x.fbx`

## 为什么用无头脚本

可重复、可进 CI、避免手动点导出参数出错。默认参数已按 Unity 坐标系配置。

## 主脚本：`scripts/blender/tripo_to_unity.py`

流程：

1. 清空默认场景 → 导入 GLB（Tripo 产物）。
2. 清理：删除空物体、非流形边可选项、合并同名材质冗余。
3. 减面（可选）：Decimate 修改器应用，按目标面数比例。
4. 坐标系统一：应用变换，导出 `-Z forward / Y up`，缩放 1.0。
5. 贴图导出：把 PBR 贴图拷到输出目录，供 Unity 关联。
6. 导出 FBX（含骨骼/动画，如果源模型有）。

## 用法

```bash
blender --background --python scripts/blender/tripo_to_unity.py -- \
  --input raw/hero.glb \
  --output build/hero/hero.fbx \
  --scale 1.0 \
  --decimate 0.6        # 可选：保留 60% 面
```

## 面数预算（游戏资产参考）

| 类型 | 三角面预算 | 说明 |
|------|-----------|------|
| 主要角色 | 15k–30k | 近景，可高一些 |
| 普通 NPC | 5k–15k | 中景 |
| 道具 | 500–5k | 视屏幕占比 |
| 远景物 | <500 | 可考虑 impostor |

## 坐标系（关键）

Unity 使用 **左手系，Y-up，Z-forward**。Blender 导出 FBX 时用：

- `axis_forward = '-Z'`
- `axis_up = 'Y'`
- `global_scale = 1.0`
- `apply_unit_scale = True`

脚本已设好默认值，一般不要改。若 Unity 中模型朝向错误，优先检查这里。

## 贴图

- 保持 PBR 命名：`*_basecolor.png`、`*_normal.png`、`*_roughness.png`、`*_metallic.png`。
- 法线图在 Unity 里设为 `Normal Map` 类型；脚本导出时不改贴图内容，只整理文件名与目录。
- 避免 >2048 的单张贴图（移动端用 1024）。

## 校验

导出后跑骨骼校验（人形）与交付校验：

```bash
blender --background --python scripts/blender/rig_check.py -- --input build/hero/hero.fbx
python scripts/python/validate_assets.py build/hero --expect-humanoid
```
