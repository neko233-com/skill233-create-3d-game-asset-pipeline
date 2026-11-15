# 05 · Unity 导入（URP）

## 导入前的资产结构

```
build/hero/
├── hero.fbx              # 模型 + 骨骼 + 动画
├── hero_basecolor.png
├── hero_normal.png
├── hero_roughness.png
└── hero_metallic.png
```

## 导入步骤

1. 把 `build/hero/` 整个目录拖进 Unity 的 `Assets/Characters/Hero/`。
2. 选中 `hero.fbx` → Inspector → **Model**：
   - Scale Factor: `1`
   - Convert Units: 按需（脚本已按 1.0 处理，一般关闭）
   - Read/Write Enabled: 需要脚本改网格时开启
3. **Rig** 页：
   - Animation Type: 人形选 `Humanoid`，道具选 `None`
   - Avatar Definition: `Create From This Model`
   - 点击 `Configure` 检查骨骼映射，无红色缺失项
4. **Animation** 页：
   - 每个 clip 设置名称、循环时间、Loop Time（idle/walk/run 勾选）
5. **Materials** 页：
   - 勾选 `Import Materials`，或用自己的 URP/Lit 材质覆盖

## 材质（URP）

- 用 `Universal Render Pipeline/Lit` 着色器。
- Basecolor 拖到 Base Map；Normal 图设 Texture Type = `Normal Map` 后拖到 Normal Map。
- Metallic / Roughness 按贴图或统一数值设置。

## 生成 Prefab

- 把模型拖进场景 → 调整 Transform → 挂上 Animator（绑定 Controller）→ 拖回 Project 存为 Prefab。
- Animator Controller：新建，拖入 idle 作为默认状态，walk/run/attack 按需加 Transitions。

## 验收

- [ ] 场景里模型朝向正确（面朝 +Z）
- [ ] 无粉材质（贴图丢失）
- [ ] Humanoid Avatar 配置通过（无骨骼映射错误）
- [ ] idle 动画正常循环，无 T-pose 残留
- [ ] Prefab 已保存，Animator Controller 已绑定
