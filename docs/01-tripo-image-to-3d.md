# 01 · Tripo 图生 3D

> Tripo 入口：**https://studio.tripo3d.com/**（浏览器端，无需安装）

## 参考图要求（决定生成质量）

- **主体单一**：一张图只有一个物体/角色，避免遮挡与多主体。
- **正面视角**：人形用正面或 3/4 正面站姿，四肢尽量分开（避免 A/T-pose 粘连）。
- **干净背景**：纯色或透明背景，减少模型把背景"烤"进几何体。
- **光线均匀**：避免强阴影与反射高光。
- **正方形**：用 `prepare_reference.py` 归一化到 1024×1024。

## 生成步骤

1. 打开 https://studio.tripo3d.com/ 并登录。
2. 选择 **Image-to-3D**（图生 3D）。
3. 上传归一化后的参考图（来自 `raw/`）。
4. 关键参数（以当前 UI 为准）：
   - 生成模式：标准 / 高质量（角色建议高质量）
   - 是否需要纹理：开启（PBR 贴图）
   - 面数：默认即可，后续在 Blender 里减面
5. 生成完成后预览，检查：正面结构完整、无明显破面、贴图无严重拉伸。
6. 下载模型，优先选 **GLB**（单文件含贴图，Blender 导入最省事）。保存到 `raw/<asset-id>.glb`。

## 文生 3D（可选）

没有参考图时，用 Text-to-3D：写清主体、风格、姿态、材质。仍从 https://studio.tripo3d.com/ 进入。提示词模板见 [examples/tripo_prompts.md](../examples/tripo_prompts.md)。

## 自动绑定与动画（人形）

Tripo 支持对生成的人形做自动绑定（auto-rig）与动画。需要角色动画时，优先在 https://studio.tripo3d.com/ 内完成绑定并导出带骨骼的模型；也可导出静态 GLB 后到 Blender / Unity 侧绑定，详见 [04-humanoid-animation.md](04-humanoid-animation.md)。

## 产物

| 文件 | 位置 | 说明 |
|------|------|------|
| `<asset-id>.glb` | `raw/` | Tripo 原始导出，保留不要删（便于回溯） |
| 参考图 | `raw/` | `prepare_reference.py` 产物 |

## 常见坑

- **生成物带底座/背景** → 参考图背景不够干净；在 Blender 清洗阶段删除无关网格。
- **贴图糊/拉伸** → 参考图分辨率太低或视角太偏；重做参考图。
- **模型朝向不对** →  Blender 导入后按 02 文档统一坐标系，不要依赖 Tripo 的朝向。
