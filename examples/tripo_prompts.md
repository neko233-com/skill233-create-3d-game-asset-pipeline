# Tripo 提示词模板

> **`tripo` 现在就是默认生产供应商（DEFAULT Supplier）**（`assetctl init <id>` 默认 `--strategy tripo`）；`blender` / `import` 为可选策略。Tripo 入口：https://studio.tripo3d.com/ 。
> 默认 `handoff.mode = "manual"`：在 https://studio.tripo3d.com/ 上传参考图 / 用 Text-to-3D，下载 GLB 到 `raw/<id>.glb` 后 `assetctl run <id> --from modeling`；缺文件会打印指引并 exit `3`（`ErrManualStepRequired`）。
> CI / 无人值守用 `assetctl init <id> --handoff fixture`（生成确定性最小合法 GLB），或 `ASSETCTL_TRIPO_HANDOFF=fixture`；上游落盘用 `--handoff file --source <path>` / `ASSETCTL_TRIPO_SOURCE`。
> 见 [docs/10-strategies.md](../docs/10-strategies.md) 与 [docs/20-stage-01-modeling.md](../docs/20-stage-01-modeling.md)。

用于 https://studio.tripo3d.com/ 的 Text-to-3D，或作为图生 3D 的补充描述。

## 人形角色

```
A stylized low-poly knight character in full armor, standing in A-pose,
front view, clean untextured grey background, even lighting, game-ready,
symmetric, single character only.
```

中文版：
```
风格化低多边形骑士角色，全身盔甲，A-pose 站立，正面视角，纯灰背景，
光线均匀，游戏可用，左右对称，仅单个角色。
```

## 道具

```
A simple wooden sword low-poly game prop, centered, front view,
clean grey background, even lighting, PBR textures, single object only.
```

## 编写要点

- 主体单一 + 视角明确（front view）
- 干净背景（clean grey background）
- 姿态（A-pose / T-pose 站立）
- 风格与用途（stylized / low-poly / game-ready）
- 明确 `single ... only` 避免多主体
