"""Stage 3 · bake (SPEC.md §4/§5, docs/40).

Bakes Normal / AO (and curvature if requested) textures into
build/<id>/<id>_normal.png etc.

Two paths:
  - deterministic placeholder path (default `blender` strategy): flat PBR PNGs
    written by common.ensure_textures — headless/CI friendly, no GPU needed.
  - real bake path (tripo/import high-poly): when `bake.engine == "cycles"` is
    set we attempt a Cycles AO/Normal bake; failures fall back to placeholders.

Input : work/<id>/02-topology-uv.blend
Output: work/<id>/03-bake.blend + build/<id>/*_*.png
Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, save_blend, cleanup_scene,
    ensure_textures,
)


def real_bake(cfg, asset):
    """Optional Cycles AO bake. Guarded so a missing GPU/addon never blocks.

    High-poly -> low-poly normal baking requires selected-to-active setup and
    real geometry from Tripo/import; here we only bake AO onto active meshes.
    """
    bake_cfg = cfg.get("bake") or {}
    if str(bake_cfg.get("engine", "placeholder")).lower() != "cycles":
        return False
    try:
        scene = bpy.context.scene
        scene.render.engine = "CYCLES"
        scene.cycles.bake_type = "AO"
        scene.render.bake.margin = 4
        for obj in [o for o in bpy.data.objects if o.type == "MESH"]:
            bpy.ops.object.select_all(action="DESELECT")
            obj.select_set(True)
            bpy.context.view_layer.objects.active = obj
            if not obj.data.uv_layers:
                obj.data.uv_layers.new(name="UVMap")
            size = int(bake_cfg.get("size", 512))
            img = bpy.data.images.new(name=asset + "_ao_bake",
                                      width=size, height=size)
            img.filepath_raw = os.path.abspath(
                os.path.join("build", asset, asset + "_ao.png"))
            img.file_format = "PNG"
            bpy.ops.object.bake(type="AO")
            img.save()
            bpy.data.images.remove(img)
            log("Cycles AO bake written for %s" % obj.name)
        return True
    except Exception as exc:
        warn("Cycles AO bake unavailable (%s); using placeholders" % exc)
        return False


def main():
    a = parse_args()
    cfg = load_config(a.config)
    open_input(a.inp)

    baked = real_bake(cfg, a.asset)
    # Idempotent: never overwrite a real bake with a placeholder.
    ensure_textures(cfg, a.asset)
    log("texture outputs %s" % ("Cycles bake" if baked else "placeholders"))

    cleanup_scene(cfg)
    save_blend(a)
    log("done -> %s" % a.out)


if __name__ == "__main__":
    main()
