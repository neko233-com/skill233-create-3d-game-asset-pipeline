"""common.py — shared helpers for every scripts/blender/*.py stage.

Runs inside Blender 5.2's bundled Python: bpy + stdlib ONLY (no Pillow, no pip).
Every stage is launched by the Go 1.27 runner as:

    blender --background --factory-startup --python <script> -- \
        --asset <id> --config <asset.json> --in <prev> --out <out> \
        --blend-out <work/NN-*.blend>

So cwd is the repo root and all paths are root-relative. See SPEC.md §5.
Tripo source: https://studio.tripo3d.com/
"""

import argparse
import json
import os
import sys

import bpy

TRIPO_URL = "https://studio.tripo3d.com/"
BLENDER_VERSION = "5.2"
UNITY_ENGINE = "unity-urp"

# FBX export contract (SPEC.md §9) — single source of truth for stage 5.
# Do not change these seven values without editing SPEC.md first.
FBX_EXPORT = dict(
    axis_forward="-Z",
    axis_up="Y",
    global_scale=1.0,
    apply_unit_scale=True,
    add_leaf_bones=False,
    path_mode="COPY",
)

PBR_SUFFIXES = ("basecolor", "normal", "roughness", "metallic", "ao")


def log(msg):
    print("[pipeline] %s" % msg)
    sys.stdout.flush()


def warn(msg):
    print("[pipeline][WARN] %s" % msg)
    sys.stdout.flush()


# ---------------------------------------------------------------------------
# Argument handling
# ---------------------------------------------------------------------------
class Args(object):
    def __init__(self, asset, config, inp, out, blend_out):
        self.asset = asset
        self.config = config
        self.inp = inp
        self.out = out
        self.blend_out = blend_out


def parse_args():
    """Parse the tokens after the literal `--`.

    `--in` and `--input` are aliases so rig_check.py can be called both by the
    Go runner (`-- --input <fbx>`) and in stage-style form (`--in <fbx>`).
    Everything except `--asset` is optional so the humanoid gate call, which
    passes only `--input`, still works. parse_known_args keeps us tolerant of
    extra flags a human may add.
    """
    argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
    p = argparse.ArgumentParser(add_help=False)
    p.add_argument("--asset", default="")
    p.add_argument("--config", default="")
    p.add_argument("--in", "--input", dest="inp", default="")
    p.add_argument("--out", default="")
    p.add_argument("--blend-out", dest="blend_out", default="")
    ns, _unknown = p.parse_known_args(argv)
    return Args(ns.asset, ns.config, ns.inp, ns.out, ns.blend_out)


def load_config(path):
    """Read asset.json. Missing path -> empty dict (gate calls pass none)."""
    if not path or not os.path.isfile(path):
        return {}
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)


# ---------------------------------------------------------------------------
# Path helpers
# ---------------------------------------------------------------------------
def ensure_dir(path):
    os.makedirs(os.path.dirname(os.path.abspath(path)) or ".", exist_ok=True)


def build_dir(asset):
    return os.path.join(os.getcwd(), "build", asset)


def work_dir(asset):
    return os.path.join(os.getcwd(), "work", asset)


def abs_path(p):
    return os.path.abspath(p)


def texture_size(cfg):
    """Placeholder bake size. Real Cycles bakes use budget.maxTextureSize;
    the deterministic stdlib/bpy placeholder is clamped to keep CI cheap."""
    budget = (cfg.get("budget") or {}) if isinstance(cfg, dict) else {}
    n = int(budget.get("maxTextureSize", 2048) or 2048)
    return max(64, min(n, 512))


def write_humanoid_marker(asset):
    """Create the empty .humanoid marker so `assetctl validate
    --expect-humanoid` passes for any asset id (not just hero*/character*)."""
    marker = os.path.join(build_dir(asset), ".humanoid")
    ensure_dir(marker)
    with open(marker, "w", encoding="utf-8") as fh:
        fh.write("")
    log("wrote humanoid marker -> %s" % marker)


# ---------------------------------------------------------------------------
# Scene lifecycle
# ---------------------------------------------------------------------------
def clear_scene():
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.object.delete(use_global=False)
    for block in (bpy.data.meshes, bpy.data.armatures, bpy.data.materials,
                  bpy.data.images, bpy.data.curves, bpy.data.cameras,
                  bpy.data.lights):
        for item in list(block):
            try:
                if item.users == 0:
                    block.remove(item)
            except Exception:
                pass


def open_input(inp):
    """Prepare the stage's scene.

    - empty        -> clear (procedural modeling by the default blender strategy)
    - .blend       -> open the previous stage's work file
    - glb/gltf/fbx/obj -> clear + import the strategy's raw output
    """
    if not inp:
        clear_scene()
        return
    ext = os.path.splitext(inp)[1].lower()
    path = abs_path(inp)
    clear_scene()
    if ext == ".blend":
        bpy.ops.wm.open_mainfile(filepath=path)
    elif ext in (".glb", ".gltf"):
        bpy.ops.import_scene.gltf(filepath=path)
    elif ext == ".fbx":
        bpy.ops.import_scene.fbx(filepath=path)
    elif ext == ".obj":
        bpy.ops.wm.obj_import(filepath=path)
    else:
        warn("unknown input extension %r; starting from empty scene" % ext)
    log("opened input %s" % inp)


def save_blend(a):
    """Save the .blend work file(s). Stage 5/6 have a non-.blend --out (FBX /
    PNG), so only paths ending in .blend are written; duplicates are removed."""
    seen = set()
    for p in (a.out, a.blend_out):
        if p and p.lower().endswith(".blend") and abs_path(p) not in seen:
            seen.add(abs_path(p))
            ensure_dir(p)
            bpy.ops.wm.save_as_mainfile(filepath=abs_path(p))
            log("saved blend -> %s" % p)


def set_active(obj):
    bpy.ops.object.select_all(action="DESELECT")
    obj.select_set(True)
    bpy.context.view_layer.objects.active = obj


def all_meshes():
    return [o for o in bpy.data.objects if o.type == "MESH"]


def first_armature():
    for o in bpy.data.objects:
        if o.type == "ARMATURE":
            return o
    return None


def cleanup_scene(cfg):
    """ACT Cleanup Empties / Negative Scale alignment (SPEC.md §10)."""
    for obj in list(bpy.data.objects):
        if obj.type == "EMPTY":
            bpy.data.objects.remove(obj, do_unlink=True)
    for obj in list(bpy.data.objects):
        if obj.type in ("MESH", "ARMATURE"):
            try:
                set_active(obj)
                bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
            except Exception as exc:
                warn("transform_apply on %s skipped: %s" % (obj.name, exc))
    log("cleanup done (%d mesh(es))" % len(all_meshes()))


# ---------------------------------------------------------------------------
# Texture helpers (bpy-native, no Pillow)
# ---------------------------------------------------------------------------
def texture_path(asset, suffix):
    return os.path.join(build_dir(asset), "%s_%s.png" % (asset, suffix))


# Flat placeholder colours for the procedural `blender` strategy. Real bakes
# (tripo/import high-poly -> stage 3 Cycles bake) replace these with detail.
FLAT = {
    "basecolor": (0.55, 0.42, 0.30, 1.0),
    "normal": (0.50, 0.50, 1.00, 1.0),
    "roughness": (0.80, 0.80, 0.80, 1.0),
    "metallic": (0.00, 0.00, 0.00, 1.0),
    "ao": (1.00, 1.00, 1.00, 1.0),
}
DATA_MAPS = ("normal", "roughness", "metallic", "ao")


def make_solid_png(path, size, rgba):
    """Write a size×size solid-colour PNG using only bpy + stdlib."""
    ensure_dir(path)
    name = os.path.splitext(os.path.basename(path))[0]
    img = bpy.data.images.new(name=name, width=size, height=size, alpha=True)
    px = list(rgba) * (size * size)
    img.pixels.foreach_set(px)
    img.filepath_raw = abs_path(path)
    img.file_format = "PNG"
    try:
        img.save()
    finally:
        bpy.data.images.remove(img)
    log("wrote texture -> %s (%dx%d)" % (path, size, size))


def ensure_textures(cfg, asset):
    """Generate every declared PBR channel (+ao) as a deterministic placeholder.

    idempotent: an existing real bake is never overwritten.
    """
    tex_list = cfg.get("textures") or list(PBR_SUFFIXES[:4])
    suffixes = [s for s in tex_list if s in PBR_SUFFIXES]
    size = texture_size(cfg)
    for s in suffixes:
        p = texture_path(asset, s)
        if not os.path.isfile(p):
            make_solid_png(p, size, FLAT.get(s, (0.5, 0.5, 0.5, 1.0)))
    # AO is optional but harmless; validate.go accepts `_ao`.
    ao = texture_path(asset, "ao")
    if "ao" not in suffixes and not os.path.isfile(ao):
        make_solid_png(ao, size, FLAT["ao"])
    return suffixes


def set_colorspace(image, suffix):
    """Basecolor = sRGB; data maps = Non-Color (docs/41 §4)."""
    try:
        image.colorspace_settings.name = "sRGB" if suffix == "basecolor" else "Non-Color"
    except Exception as exc:
        warn("colorspace for %s skipped: %s" % (image.name, exc))


# ---------------------------------------------------------------------------
# FBX export (stage 5 and the tripo_to_unity shim)
# ---------------------------------------------------------------------------
def export_fbx(path, cfg=None, humanoid=None):
    """Export MESH + ARMATURE with the SPEC.md §9 contract."""
    cfg = cfg or {}
    if humanoid is None:
        humanoid = bool(cfg.get("humanoid"))
    ensure_dir(path)
    bpy.ops.object.select_all(action="DESELECT")
    for obj in bpy.data.objects:
        if obj.type in ("MESH", "ARMATURE"):
            obj.select_set(True)
    kwargs = dict(FBX_EXPORT)
    kwargs.update(
        filepath=abs_path(path),
        use_selection=True,
        bake_anim=bool(humanoid),
        bake_anim_use_all_actions=bool(humanoid),
        embed_textures=False,
    )
    bpy.ops.export_scene.fbx(**kwargs)
    log("exported FBX -> %s (humanoid=%s)" % (path, humanoid))
