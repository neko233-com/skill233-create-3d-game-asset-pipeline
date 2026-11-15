"""Stage 4 · PBR material (SPEC.md §4/§5, docs/41).

Builds a Principled BSDF node tree wired to the baked PNGs, sets Non-Color
colorspace on data maps, and cleans duplicate/unused materials (ACT Material
Tools alignment: https://github.com/mrven/Blender-Asset-Creation-Toolset).

Principled BSDF -> Unity URP/Lit mapping (docs/41 §3):
  Base Color     <- <id>_basecolor.png   (sRGB)
  Normal         <- <id>_normal.png       (Non-Color) via Normal Map node
  Roughness      <- <id>_roughness.png    (Non-Color)
  Metallic       <- <id>_metallic.png     (Non-Color)
  AO             <- <id>_ao.png           (Non-Color, multiplied into base)

Input : work/<id>/03-bake.blend
Output: work/<id>/04-pbr.blend
Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, save_blend, cleanup_scene,
    texture_path, set_colorspace, all_meshes,
)


def load_image(asset, suffix):
    p = texture_path(asset, suffix)
    if not os.path.isfile(p):
        warn("missing %s; skipping that channel" % p)
        return None
    img = bpy.data.images.load(os.path.abspath(p), check_existing=True)
    set_colorspace(img, suffix)
    return img


def make_material(asset, cfg):
    """One shared URP/Lit-equivalent material. Replace with per-slot materials
    for a real multi-material asset."""
    mat_name = asset + "_PBR"
    mat = bpy.data.materials.get(mat_name) or bpy.data.materials.new(name=mat_name)
    mat.use_nodes = True
    nt = mat.node_tree
    nt.nodes.clear()

    out = nt.nodes.new("ShaderNodeOutputMaterial")
    out.location = (600, 0)
    bsdf = nt.nodes.new("ShaderNodeBsdfPrincipled")
    bsdf.location = (300, 0)
    nt.links.new(bsdf.outputs["BSDF"], out.inputs["Surface"])

    def tex_node(suffix, loc):
        img = load_image(asset, suffix)
        if img is None:
            return None
        node = nt.nodes.new("ShaderNodeTexImage")
        node.image = img
        node.location = loc
        node.label = suffix
        return node

    base = tex_node("basecolor", (-400, 200))
    if base:
        nt.links.new(base.outputs["Color"], bsdf.inputs["Base Color"])

    nrm = tex_node("normal", (-400, -100))
    if nrm:
        nmap = nt.nodes.new("ShaderNodeNormalMap")
        nmap.location = (0, -100)
        nt.links.new(nrm.outputs["Color"], nmap.inputs["Color"])
        nt.links.new(nmap.outputs["Normal"], bsdf.inputs["Normal"])

    for suffix, loc, socket in (
        ("roughness", (-400, -350), "Roughness"),
        ("metallic", (-400, -550), "Metallic"),
    ):
        node = tex_node(suffix, loc)
        if node:
            nt.links.new(node.outputs["Color"], bsdf.inputs[socket])

    ao = tex_node("ao", (-400, 450))
    if ao and base:
        mix = nt.nodes.new("ShaderNodeMixRGB")
        mix.blend_type = "MULTIPLY"
        mix.inputs[0].default_value = 1.0
        mix.location = (-50, 200)
        nt.links.new(base.outputs["Color"], mix.inputs[1])
        nt.links.new(ao.outputs["Color"], mix.inputs[2])
        nt.links.new(mix.outputs["Color"], bsdf.inputs["Base Color"])

    log("built PBR material %s (basecolor/normal/roughness/metallic/ao)"
        % mat.name)
    return mat


def cleanup_materials(mat):
    """ACT Material Tools alignment: assign the unified PBR material everywhere
    and purge zero-user materials left behind by the Tripo/import GLB."""
    for obj in all_meshes():
        if not obj.data.materials:
            obj.data.materials.append(mat)
        else:
            for i in range(len(obj.data.materials)):
                obj.data.materials[i] = mat
    for m in list(bpy.data.materials):
        if m.users == 0 and m != mat:
            bpy.data.materials.remove(m)


def main():
    a = parse_args()
    cfg = load_config(a.config)
    open_input(a.inp)

    mat = make_material(a.asset, cfg)
    cleanup_materials(mat)

    cleanup_scene(cfg)
    save_blend(a)
    log("done -> %s" % a.out)


if __name__ == "__main__":
    main()
