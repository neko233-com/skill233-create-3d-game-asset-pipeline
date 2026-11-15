"""Stage 2 · topology + UV (SPEC.md §4/§5, docs/30).

Aligns with ACT (https://github.com/mrven/Blender-Asset-Creation-Toolset):
  - Geometry Tools: decimate to budget.maxTriangles
  - UV Tools: Smart UV project, second "LightmapUV" layer, set active UV
  - Origin/Cleanup already applied in stage 1; re-apply scale here defensively

Input : work/<id>/01-modeling.blend (via --in)
Output: work/<id>/02-topology-uv.blend (+ --blend-out)
Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, save_blend, cleanup_scene,
    all_meshes, set_active,
)


def _triangle_count(obj):
    """Best-effort triangle count (evaluated, then raw mesh fallback)."""
    try:
        dg = bpy.context.evaluated_depsgraph_get()
        ev = obj.evaluated_get(dg)
        mesh = ev.to_mesh()
        tri = sum(len(p.vertices) - 2 for p in mesh.polygons)
        ev.to_mesh_clear()
        return tri
    except Exception:
        return sum(len(p.vertices) - 2 for p in obj.data.polygons)


def decimate_to_budget(cfg):
    """ACT Geometry Tools alignment: keep <= budget.maxTriangles."""
    budget = cfg.get("budget") or {}
    max_tri = int(budget.get("maxTriangles", 30000) or 30000)
    if max_tri <= 0:
        log("no triangle budget set; skip decimate")
        return
    for obj in all_meshes():
        tri = _triangle_count(obj)
        if tri <= max_tri:
            log("%s within budget (%d <= %d tris)" % (obj.name, tri, max_tri))
            continue
        ratio = min(1.0, max_tri / float(tri))
        mod = obj.modifiers.new(name="BudgetDecimate", type="DECIMATE")
        mod.ratio = ratio
        try:
            set_active(obj)
            bpy.ops.object.modifier_apply(modifier=mod.name)
            log("decimated %s: %d -> <=%d tris (ratio=%.3f)"
                % (obj.name, tri, max_tri, ratio))
        except Exception as exc:
            warn("decimate apply skipped on %s: %s" % (obj.name, exc))


def ensure_uv(obj):
    """Smart UV project + a second UV layer reserved for lightmaps."""
    if not obj.data.uv_layers:
        try:
            set_active(obj)
            bpy.ops.object.mode_set(mode="EDIT")
            bpy.ops.mesh.select_all(action="SELECT")
            bpy.ops.uv.smart_project(island_margin=0.02)
            bpy.ops.object.mode_set(mode="OBJECT")
            log("smart UV projected on %s" % obj.name)
        except Exception as exc:
            warn("smart_project skipped on %s (%s); adding default UVMap"
                 % (obj.name, exc))
            try:
                bpy.ops.object.mode_set(mode="OBJECT")
            except Exception:
                pass
            obj.data.uv_layers.new(name="UVMap")

    names = [uv.name for uv in obj.data.uv_layers]
    if "LightmapUV" not in names:
        obj.data.uv_layers.new(name="LightmapUV")
        log("added LightmapUV (2nd channel) on %s" % obj.name)

    # ACT "Mark Seams from UV" equivalent (guarded: operator availability varies).
    try:
        set_active(obj)
        bpy.ops.object.mode_set(mode="EDIT")
        bpy.ops.mesh.select_all(action="SELECT")
        bpy.ops.uv.seams_from_islands()
        bpy.ops.object.mode_set(mode="OBJECT")
        log("marked seams from UV islands on %s" % obj.name)
    except Exception as exc:
        warn("seams_from_islands skipped: %s" % exc)
        try:
            bpy.ops.object.mode_set(mode="OBJECT")
        except Exception:
            pass


def main():
    a = parse_args()
    cfg = load_config(a.config)
    open_input(a.inp)

    decimate_to_budget(cfg)
    for obj in all_meshes():
        ensure_uv(obj)

    cleanup_scene(cfg)
    save_blend(a)
    log("done -> %s" % a.out)


if __name__ == "__main__":
    main()
