"""Stage 1 · modeling (SPEC.md §3/§4/§5).

Strategy-dependent entry point:
  - strategy=tripo (DEFAULT production supplier): --in is raw/<id>.glb
    downloaded from https://studio.tripo3d.com/ (image-to-3D), imported as-is.
  - strategy=blender (alternative / CI baseline): --in is EMPTY -> procedural
    blockout from asset.json.modeling (primitive / size / subdivisions).
  - strategy=import (alternative): --in is a copied raw/<id>.{glb,fbx,obj,...}.

If an imported file yields no mesh (empty GLB / upstream glitch), a procedural
blockout is generated so downstream stages never operate on an empty scene.

Output: work/<id>/01-modeling.blend (also saved to --blend-out).
Runs inside Blender 5.2 native Python (bpy + stdlib only), headless and
--factory-startup safe.
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402  (provided by Blender 5.2's bundled Python)

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, save_blend, cleanup_scene,
    all_meshes, set_active,
)

PRIMITIVES = {"cube", "sphere", "cylinder", "plane"}


def build_procedural(cfg, asset):
    """Deterministic blockout mesh (CI-friendly, no external service)."""
    m = cfg.get("modeling", {}) or {}
    primitive = str(m.get("primitive", "cube")).lower()
    if primitive not in PRIMITIVES:
        primitive = "cube"
    size = float(m.get("size", 1.0)) or 1.0
    subs = max(0, min(int(m.get("subdivisions", 0)), 3))

    if primitive == "cube":
        bpy.ops.mesh.primitive_cube_add(size=size)
    elif primitive == "sphere":
        bpy.ops.mesh.primitive_uv_sphere_add(radius=size, segments=32, ring_count=16)
    elif primitive == "cylinder":
        bpy.ops.mesh.primitive_cylinder_add(radius=size, depth=size * 2.0, vertices=32)
    else:  # plane
        bpy.ops.mesh.primitive_plane_add(size=size * 2.0)

    obj = bpy.context.active_object
    obj.name = asset

    if subs > 0:
        mod = obj.modifiers.new(name="BlockoutSubdiv", type="SUBSURF")
        mod.levels = subs
        mod.render_levels = subs
        try:
            set_active(obj)
            bpy.ops.object.modifier_apply(modifier=mod.name)
        except Exception as exc:  # pragma: no cover - bpy context dependent
            warn("subsurf apply skipped: %s" % exc)

    # ACT Origin Align Tool equivalent (SPEC.md §10): centre origin on bounds.
    try:
        set_active(obj)
        bpy.ops.object.origin_set(type="ORIGIN_GEOMETRY", center="BOUNDS")
    except Exception as exc:
        warn("origin_set skipped: %s" % exc)

    log("procedural %s (size=%.2f, subdivisions=%d)" % (primitive, size, subs))
    return obj


def main():
    a = parse_args()
    cfg = load_config(a.config)
    open_input(a.inp)  # clears scene / imports raw file, or opens a .blend

    meshes = all_meshes()
    if not a.inp or not meshes:
        # Empty --in = procedural `blender` strategy; imported-but-empty = a
        # Tripo/import drop without geometry. Both must yield a mesh.
        if a.inp and not meshes:
            warn("imported %s produced no mesh; falling back to procedural blockout" % a.inp)
        meshes = [build_procedural(cfg, a.asset)]

    # Humanoid assets need a Unity-Humanoid skeleton from day one so the
    # stage-5 rig_check.py gate and bake_anim=True clip export work.
    if cfg.get("humanoid") and not any(o.type == "ARMATURE" for o in bpy.data.objects):
        try:
            from humanoid_rig import create_humanoid_rig, attach_mesh, create_clips
            arm = create_humanoid_rig(a.asset, cfg.get("animations", []))
            for obj in meshes:
                attach_mesh(obj, arm)
            create_clips(arm, a.asset, cfg.get("animations", []))
            log("humanoid rig + %d clip(s) created" % len(cfg.get("animations", [])))
        except Exception as exc:  # keep pipeline alive; rig_check reports details
            warn("humanoid rig creation failed: %s" % exc)

    cleanup_scene(cfg)
    save_blend(a)
    log("done -> %s" % a.out)


if __name__ == "__main__":
    main()
