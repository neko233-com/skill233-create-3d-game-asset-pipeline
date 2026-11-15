# Asset Manifest · hero-knight

示例资产清单。复制为 `build/<asset-id>/manifest.md` 并按实际填写。

- Asset ID: `hero-knight`
- Type: character / humanoid / animated
- Source: Tripo Image-to-3D @ https://studio.tripo3d.com/
- Reference: `raw/hero-knight.png`
- Raw model: `raw/hero-knight.glb`
- Rig: Tripo auto-rig / Blender manual
- Triangles: 18432
- Textures: basecolor, normal, roughness, metallic (1024)
- Animations: idle(loop), walk(loop), attack(once)

## DoD

- [x] naming convention
- [x] PBR textures
- [x] -Z forward / Y up / scale 1.0
- [x] face budget
- [x] rig_check.py passed
- [x] validate_assets.py passed
- [x] Unity import verified (prefab + animator)
