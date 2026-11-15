"""Blender 5.2 · Tripo GLB -> Unity FBX.

Usage (headless):
    blender --background --python scripts/blender/tripo_to_unity.py -- \
        --input raw/hero.glb --output build/hero/hero.fbx [--scale 1.0] [--decimate 0.6]

Runs inside Blender's bundled Python (bpy available).
Tripo source: https://studio.tripo3d.com/
"""

import argparse
import os
import shutil
import sys

import bpy


def parse_args():
    argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
    p = argparse.ArgumentParser(description="Tripo GLB -> Unity FBX (Blender 5.2)")
    p.add_argument("--input", required=True, help="input GLB/OBJ/FBX from Tripo")
    p.add_argument("--output", required=True, help="output .fbx path")
    p.add_argument("--scale", type=float, default=1.0, help="global scale (default 1.0)")
    p.add_argument("--decimate", type=float, default=0.0,
                   help="keep ratio 0<r<=1, 0 disables decimation")
    p.add_argument("--export-anim", action="store_true", help="export animations if present")
    return p.parse_args(argv)


def clear_scene():
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.object.delete(use_global=False)
    for block in (bpy.data.meshes, bpy.data.armatures, bpy.data.materials, bpy.data.images):
        for item in list(block):
            if item.users == 0:
                block.remove(item)


def import_model(path):
    ext = os.path.splitext(path)[1].lower()
    if ext == ".glb" or ext == ".gltf":
        bpy.ops.import_scene.gltf(filepath=path)
    elif ext == ".fbx":
        bpy.ops.import_scene.fbx(filepath=path)
    elif ext == ".obj":
        bpy.ops.wm.obj_import(filepath=path)
    else:
        raise ValueError(f"Unsupported input format: {ext}")


def clean_scene():
    # Remove empty objects.
    for obj in list(bpy.data.objects):
        if obj.type == "EMPTY":
            bpy.data.objects.remove(obj, do_unlink=True)
    # Apply transforms on meshes & armatures so export scale is predictable.
    for obj in bpy.data.objects:
        if obj.type in {"MESH", "ARMATURE"}:
            bpy.context.view_layer.objects.active = obj
            obj.select_set(True)
            bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
            obj.select_set(False)


def decimate_meshes(ratio):
    if not 0 < ratio < 1:
        return
    for obj in bpy.data.objects:
        if obj.type != "MESH" or len(obj.data.polygons) < 100:
            continue
        bpy.context.view_layer.objects.active = obj
        mod = obj.modifiers.new(name="Decimate", type="DECIMATE")
        mod.ratio = ratio
        bpy.ops.object.modifier_apply(modifier=mod.name)


def copy_textures(output_fbx):
    """Copy referenced image textures next to the FBX for Unity association."""
    out_dir = os.path.dirname(output_fbx)
    os.makedirs(out_dir, exist_ok=True)
    for img in bpy.data.images:
        if img.source != "FILE" or not img.filepath:
            continue
        src = bpy.path.abspath(img.filepath)
        if not os.path.isfile(src):
            continue
        dst = os.path.join(out_dir, os.path.basename(src))
        if os.path.abspath(src) != os.path.abspath(dst):
            shutil.copy2(src, dst)


def export_fbx(output, scale, export_anim):
    os.makedirs(os.path.dirname(output) or ".", exist_ok=True)
    bpy.ops.object.select_all(action="DESELECT")
    for obj in bpy.data.objects:
        if obj.type in {"MESH", "ARMATURE"}:
            obj.select_set(True)
    bpy.ops.export_scene.fbx(
        filepath=output,
        use_selection=True,
        axis_forward="-Z",     # Unity: -Z forward
        axis_up="Y",           # Unity: Y up
        global_scale=scale,
        apply_unit_scale=True,
        add_leaf_bones=False,
        bake_anim=export_anim,
        path_mode="COPY",
        embed_textures=False,
    )


def main():
    args = parse_args()
    args.input = os.path.abspath(args.input)
    args.output = os.path.abspath(args.output)
    print(f"[tripo_to_unity] input={args.input}")
    clear_scene()
    import_model(args.input)
    clean_scene()
    decimate_meshes(args.decimate)
    copy_textures(args.output)
    export_fbx(args.output, args.scale, args.export_anim)
    print(f"[tripo_to_unity] done -> {args.output}")


if __name__ == "__main__":
    main()
