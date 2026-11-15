"""Stage 6 · render + composite (SPEC.md §4/§5, docs/43).

Reads the ALREADY-EXPORTED build/<id>/<id>.fbx (not 04-pbr.blend) so the
preview reflects exactly what Unity will import. Sets up a deterministic
three-point light rig + camera, renders with EEVEE to build/<id>/preview.png.

Note: preview.png is a QA artefact, NOT a PBR texture; `assetctl validate`
exempts it. The deliverable remains FBX + PBR PNGs.

Input : build/<id>/<id>.fbx
Output: build/<id>/preview.png
Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402
from mathutils import Vector  # noqa: E402

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, save_blend, build_dir,
    all_meshes,
)


def frame_objects():
    """Compute the scene bounding box centre + radius for camera framing."""
    objs = [o for o in bpy.data.objects if o.type in ("MESH", "ARMATURE")]
    if not objs:
        return Vector((0, 0, 0)), 2.0
    pts = []
    for o in objs:
        for c in o.bound_box:
            pts.append(o.matrix_world @ Vector(c))
    lo = Vector((min(p.x for p in pts), min(p.y for p in pts), min(p.z for p in pts)))
    hi = Vector((max(p.x for p in pts), max(p.y for p in pts), max(p.z for p in pts)))
    centre = (lo + hi) / 2.0
    radius = max((hi - lo).length / 2.0, 0.5)
    return centre, radius


def add_light(name, kind, loc, energy, colour=(1, 1, 1)):
    data = bpy.data.lights.new(name=name, type=kind)
    data.energy = energy
    data.color = colour
    obj = bpy.data.objects.new(name=name, object_data=data)
    bpy.context.collection.objects.link(obj)
    obj.location = loc
    return obj


def setup_stage(centre, radius):
    """Deterministic three-point lighting + camera (docs/43 §2)."""
    # Key / fill / rim.
    key = add_light("Key", "AREA", centre + Vector((radius * 2, -radius * 2, radius * 3)),
                    1200.0)
    fill = add_light("Fill", "AREA", centre + Vector((-radius * 2, -radius, radius)),
                     600.0, (0.85, 0.9, 1.0))
    rim = add_light("Rim", "AREA", centre + Vector((0, radius * 2, radius * 2)),
                    900.0, (1.0, 0.95, 0.85))
    for light in (key, fill, rim):
        direction = centre - light.location
        light.rotation_euler = direction.to_track_quat("-Z", "Y").to_euler()

    cam_data = bpy.data.cameras.new(name="PreviewCam")
    cam = bpy.data.objects.new(name="PreviewCam", object_data=cam_data)
    bpy.context.collection.objects.link(cam)
    cam.location = centre + Vector((radius * 2.2, -radius * 2.6, radius * 1.6))
    direction = centre - cam.location
    cam.rotation_euler = direction.to_track_quat("-Z", "Y").to_euler()
    cam_data.type = "ORTHO"
    cam_data.ortho_scale = radius * 3.2
    bpy.context.scene.camera = cam


def configure_render(asset):
    scene = bpy.context.scene
    # Blender 5.2 exposes exactly one EEVEE id: BLENDER_EEVEE. The pre-5.x
    # "_NEXT" variant was removed, so it is never probed here (docs/61 §4.2 +
    # skills/blender-core/SKILL.md migration table). Engine enums are dynamic,
    # so capability is detected by assignment instead of by inspection.
    chosen = None
    for candidate in ("BLENDER_EEVEE", "BLENDER_WORKBENCH"):
        try:
            scene.render.engine = candidate
            chosen = candidate
            break
        except (TypeError, ValueError, AttributeError):
            continue
    if chosen is None:
        warn("no render engine available; writing a flat fallback preview")
    scene.render.resolution_x = 512
    scene.render.resolution_y = 512
    scene.render.resolution_percentage = 100
    scene.render.image_settings.file_format = "PNG"
    scene.render.film_transparent = True
    scene.render.filepath = os.path.abspath(
        os.path.join(build_dir(asset), "preview.png"))
    return chosen


def main():
    a = parse_args()
    cfg = load_config(a.config)
    # --in is build/<id>/<id>.fbx (the exported deliverable).
    open_input(a.inp)

    centre, radius = frame_objects()
    setup_stage(centre, radius)
    engine = configure_render(a.asset)

    out = os.path.join(build_dir(a.asset), "preview.png")
    os.makedirs(os.path.dirname(out), exist_ok=True)
    if engine:
        try:
            bpy.ops.render.render(write_still=True)
            log("rendered preview with %s -> %s" % (engine, out))
        except Exception as exc:
            warn("render failed (%s); writing neutral placeholder" % exc)
            _flat_preview(out)
    else:
        _flat_preview(out)

    save_blend(a, "render")  # --blend-out work/<id>/06-render.blend (if .blend)
    log("done -> %s" % a.out)


def _flat_preview(out):
    """Neutral 512x512 fallback so the deliverable always has a preview.png."""
    img = bpy.data.images.new(name="preview", width=512, height=512, alpha=True)
    img.pixels.foreach_set([0.12, 0.12, 0.14, 1.0] * (512 * 512))
    img.filepath_raw = os.path.abspath(out)
    img.file_format = "PNG"
    img.save()
    bpy.data.images.remove(img)
    log("wrote fallback preview -> %s" % out)


if __name__ == "__main__":
    main()
