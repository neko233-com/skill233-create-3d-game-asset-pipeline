"""skill_run.py — unified Blender 5.2 bpy dispatcher for ALL registered skills.

The 28 SKILL.md documents under skills/ are copied (MIT) knowledge from
EL4CTEO/blender-skills. We do NOT vendor their Python source. Instead this ONE
bpy module exposes one deterministic operation per skill id; Go 1.27
(`assetctl skills run <id>`) validates arguments, launches it with the same
blender-style argv as the six-stage pipeline and verifies the JSON contract it
publishes.

    blender --background --factory-startup --python scripts/blender/skill_run.py -- \
        --skill blender-asset-qa --asset hero-knight --config assets/.../asset.json \
        --out build/hero-knight/contracts/skill-blender-asset-qa.json \
        --blend-out work/hero-knight/skills/blender-asset-qa.blend \
        [--param key=value ...]

Runs with Blender 5.2's bundled Python (bpy + stdlib only), --factory-startup
safe, and also under testing/fake_blender.py (mock_bpy) for automatic tests.
See SPEC.md §3B, skills/README.md, testing/test_skills.py.
"""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, clear_scene, ensure_dir,
    abs_path, build_dir, emit_json, write_contract, FBX_EXPORT, TRIPO_URL,
    BLENDER_VERSION,
)


# ---------------------------------------------------------------------------
# Shared, mock-friendly primitives
# ---------------------------------------------------------------------------
def _stats():
    """Numbers, not adjectives (blender-core Verify rule)."""
    return {
        "objects": len(bpy.data.objects),
        "meshes": len([o for o in bpy.data.objects if o.type == "MESH"]),
        "materials": len(bpy.data.materials),
        "images": len(bpy.data.images),
        "actions": len(bpy.data.actions),
        "lights": len(bpy.data.lights),
        "cameras": len(bpy.data.cameras),
        "engine": bpy.context.scene.render.engine,
        "blender": getattr(getattr(bpy, "app", None), "version_string", BLENDER_VERSION),
    }


def _ensure_mesh(asset):
    objs = [o for o in bpy.data.objects if o.type == "MESH"]
    if objs:
        return objs[0]
    bpy.ops.mesh.primitive_cube_add(size=2.0)
    objs = [o for o in bpy.data.objects if o.type == "MESH"]
    if not objs:
        mesh = bpy.data.meshes.new(name=asset + "_mesh")
        obj = bpy.data.objects.new(name=asset, object_data=mesh)
        try:
            bpy.context.collection.objects.link(obj)
        except Exception as exc:  # context-dependent in background mode
            warn("link skipped: %s" % exc)
    else:
        obj = objs[0]
    obj.name = asset
    return obj


def _safe(label, fn, obj, asset, params):
    try:
        value = fn(obj, asset, params)
        return {"op": label, "ok": True,
                "value": str(value) if value is not None else ""}
    except Exception as exc:
        # Real Blender 5.2 may lack an optional add-on; never abort a skill.
        warn("skill step %s skipped: %s" % (label, exc))
        return {"op": label, "ok": False, "error": str(exc)}


def _mod(name, mtype):
    return lambda obj, asset, params: obj.modifiers.new(name=name, type=mtype)


def _mat(name):
    return lambda obj, asset, params: bpy.data.materials.new(name=name)


def _node(ntype):
    def _fn(obj, asset, params):
        mat = bpy.data.materials.new(name=asset + "_" + ntype)
        node = mat.node_tree.nodes.new(ntype)
        return "%s/%s" % (mat.name, node.type)
    return _fn


def _image(name, size=64):
    return lambda obj, asset, params: bpy.data.images.new(
        name=name, width=size, height=size, alpha=True)


def _light(name, kind):
    def _fn(obj, asset, params):
        data = bpy.data.lights.new(name=name, type=kind)
        data.energy = 800.0
        return data.name
    return _fn


def _action(name):
    return lambda obj, asset, params: bpy.data.actions.new(name=name)


def _vertex_group(name):
    return lambda obj, asset, params: obj.vertex_groups.new(name=name)


def _write_file(rel, content):
    def _fn(obj, asset, params):
        path = os.path.join("work", asset, "skills", rel)
        ensure_dir(path)
        with open(path, "w", encoding="utf-8") as fh:
            fh.write(content)
        return path
    return _fn


def _set_engine(engine):
    return lambda obj, asset, params: setattr(bpy.context.scene.render, "engine", engine)


def _fbx_params(obj, asset, params):
    return dict(FBX_EXPORT, asset=asset, supplier=params.get("supplier", "tripo"),
                tripoUrl=TRIPO_URL)


def _qa_counts(obj, asset, params):
    names = [o.name for o in bpy.data.objects]
    return {
        "objects": len(names),
        "duplicateNames": len(names) - len(set(names)),
        "negativeScale": len([o for o in bpy.data.objects
                              if any(float(v) < 0 for v in tuple(o.location))]),
        "triangleBudget": int(params.get("maxTriangles", 30000)),
    }


def _contract_count(obj, asset, params):
    d = os.path.join(build_dir(asset), "contracts")
    files = sorted(f for f in os.listdir(d) if f.endswith(".json")) if os.path.isdir(d) else []
    return {"existingContracts": len(files), "stages": 6}


def _humanoid_bones(obj, asset, params):
    from humanoid_rig import create_humanoid_rig
    arm = create_humanoid_rig(asset, ["idle", "walk", "attack"])
    return {"bones": len(arm.data.bones), "rig": arm.name}


def _clips(obj, asset, params):
    arm = next((o for o in bpy.data.objects if o.type == "ARMATURE"), None)
    if arm is None:
        from humanoid_rig import create_humanoid_rig
        arm = create_humanoid_rig(asset, ["idle"])
    arm.animation_data_create()
    act = bpy.data.actions.new(name=asset + "-idle")
    arm.animation_data.action = act
    track = arm.animation_data.nla_tracks.new()
    track.strips.new(name=asset + "-idle", start=1, action=act)
    return {"clips": 1, "action": act.name}


def _export_fbx(obj, asset, params):
    path = os.path.join("work", asset, "skills", "export-pipelines.fbx")
    ensure_dir(path)
    bpy.ops.export_scene.fbx(filepath=abs_path(path), **dict(FBX_EXPORT))
    return path


def _render_still(obj, asset, params):
    scene = bpy.context.scene
    scene.render.engine = "BLENDER_EEVEE"
    scene.render.resolution_x = 128
    scene.render.resolution_y = 128
    scene.render.image_settings.file_format = "PNG"
    path = os.path.join("work", asset, "skills", "rendering.png")
    ensure_dir(path)
    scene.render.filepath = abs_path(path)
    bpy.ops.render.render(write_still=True)
    return path


# ---------------------------------------------------------------------------
# 28 operations — one per registered skill (internal/skills/catalog.go)
# Every step is isolated with _safe so an optional Blender add-on can never
# break the skill contract. All primitives work under mock_bpy too.
# ---------------------------------------------------------------------------
OPS = {
    "core": [("env_report", lambda o, a, p: _stats()),
             ("app_version", lambda o, a, p: getattr(bpy.app, "version_string", BLENDER_VERSION))],
    "scene_setup": [("rename_root", lambda o, a, p: setattr(o, "name", a + "_root")),
                    ("render_resolution", lambda o, a, p: setattr(bpy.context.scene.render, "resolution_x", 1024))],
    "hardsurface_modeling": [("bevel", _mod("Bevel", "BEVEL")),
                             ("weighted_normal", _mod("WeightedNormal", "WEIGHTED_NORMAL"))],
    "sculpting": [("multires", _mod("Multires", "MULTIRES")),
                  ("uv_sphere", lambda o, a, p: bpy.ops.mesh.primitive_uv_sphere_add(radius=1.0))],
    "retopology_uv": [("uv_layer", lambda o, a, p: o.data.uv_layers.new(name="UVMap")),
                      ("smart_project", lambda o, a, p: bpy.ops.uv.smart_project(island_margin=0.02)),
                      ("decimate", _mod("Decimate", "DECIMATE"))],
    "geometry_nodes": [("nodes_modifier", _mod("GeometryNodes", "NODES")),
                       ("mesh_datablock", lambda o, a, p: bpy.data.meshes.new(name=a + "_gn"))],
    "pbr_materials": [("material", _mat("M_PBR")),
                      ("principled", _node("BSDF_PRINCIPLED")),
                      ("basecolor_image", _image("skill_basecolor"))],
    "texturing": [("decal_image", _image("skill_decal")),
                  ("image_node", _node("ShaderNodeTexImage")),
                  ("assign_material", lambda o, a, p: o.data.materials.append(bpy.data.materials.new(name=a + "_Mat")))],
    "texture_baking": [("bake_target", _image("skill_bake", 128)),
                       ("bake_ao", lambda o, a, p: bpy.ops.object.bake(type="AO"))],
    "character_rigging": [("humanoid_bones", _humanoid_bones),
                          ("skin_weights", _vertex_group("Hips"))],
    "facial_shapekeys": [("arkit_action", _action("ARKit52")),
                         ("viseme_action", _action("Visemes")),
                         ("face_group", _vertex_group("Head"))],
    "animation": [("clips", _clips), ("loop_action", _action("loop-check"))],
    "clothing": [("cloth_modifier", _mod("Cloth", "CLOTH")),
                 ("body_group", _vertex_group("Body"))],
    "hair_grooming": [("hair_particles", _mod("Hair", "PARTICLE_SYSTEM")),
                      ("curves", lambda o, a, p: bpy.data.curves.new(name=a + "_curves", type="CURVE") if hasattr(bpy.data, "curves") else "n/a")],
    "game_optimization": [("lod_decimate", _mod("LOD0", "DECIMATE")),
                          ("stats", lambda o, a, p: _qa_counts(o, a, p))],
    "export_pipelines": [("fbx_params", _fbx_params), ("export_fbx", _export_fbx)],
    "pipeline_automation": [("batch_object", lambda o, a, p: bpy.data.objects.new(name=a + "_batch", object_data=o.data)),
                            ("manifest", _write_file("batch.json", json.dumps({"batch": True})))],
    "asset_qa": [("qa_counts", _qa_counts), ("stats", lambda o, a, p: _stats())],
    "workflow_recipes": [("contract_scan", _contract_count),
                         ("pipeline_summary", lambda o, a, p: {"stages": 6, "supplier": p.get("supplier", "tripo")})],
    "rendering": [("eevee", _set_engine("BLENDER_EEVEE")), ("render_still", _render_still)],
    "lighting_lookdev": [("key", _light("Key", "AREA")),
                         ("fill", _light("Fill", "AREA")),
                         ("camera", lambda o, a, p: bpy.data.cameras.new(name="PreviewCam"))],
    "stylized_npr": [("toon_material", _mat("M_Toon")),
                     ("toon_node", _node("ShaderNodeBsdfToon")),
                     ("outline", _mod("Solidify", "SOLIDIFY"))],
    "environment_kits": [("terrain_mesh", lambda o, a, p: bpy.data.meshes.new(name=a + "_terrain")),
                         ("scatter", _mod("Scatter", "NODES"))],
    "time_of_day": [("sun", _light("Sun", "SUN")),
                    ("mood", _write_file("mood.json", json.dumps({"time": "golden-hour"})))],
    "weather_fx": [("rain_particles", _mod("Rain", "PARTICLE_SYSTEM")),
                   ("wet_surface", _mat("M_Wet")),
                   ("puddle_mask", _image("puddle_mask", 64))],
    "simulation": [("rigid_body", _mod("RigidBody", "RIGID_BODY")),
                   ("cloth", _mod("ClothSim", "CLOTH"))],
    "video_editing": [("shot_action", _action("Shot_01")),
                      ("fps", lambda o, a, p: setattr(bpy.context.scene.render, "resolution_percentage", 100))],
    "addon_development": [("manifest_toml", _write_file("blender_manifest.toml",
                                                        'schema_version = "1.0.0"\nid = "assetctl_skill"\nversion = "1.0.0"\n')),
                          ("ops_module", _write_file("operators.py", "# Blender 5.2 extension stub\n"))],
}


# ---------------------------------------------------------------------------
# Args + main
# ---------------------------------------------------------------------------
class SkillArgs(object):
    def __init__(self, base, skill, params):
        self.__dict__.update(vars(base))
        self.skill = skill
        self.params = params


def parse_skill_args():
    base = parse_args()  # --asset/--config/--in/--out/--blend-out
    argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
    skill, params = "", {}
    for i, tok in enumerate(argv):
        if tok == "--skill" and i + 1 < len(argv):
            skill = argv[i + 1]
        if tok == "--param" and i + 1 < len(argv):
            k, _, v = argv[i + 1].partition("=")
            if k:
                params[k] = v
    return SkillArgs(base, skill, params)


def apply_operation(operation, asset, params):
    if operation not in OPS:
        raise ValueError("unknown bpy operation %r" % operation)
    obj = _ensure_mesh(asset)
    steps = []
    for entry in OPS[operation]:
        # Both ("label", fn) and a bare function are accepted; a malformed
        # entry must not crash the whole dispatcher.
        if isinstance(entry, tuple):
            label, fn = entry
        else:
            label, fn = getattr(entry, "__name__", "step"), entry
        steps.append(_safe(label, fn, obj, asset, params))
    files = []
    for step in steps:
        value = step.get("value", "")
        if step["ok"] and isinstance(value, str) and os.path.isfile(value):
            files.append(value)
    return {
        "operation": operation,
        "skill": None,  # replaced by main() with the real skill id
        "steps": steps,
        "ok": all(s["ok"] for s in steps),
        "applied": [s["op"] for s in steps if s["ok"]],
        "stats": _stats(),
        "extraOutputs": files,
    }


def main():
    a = parse_skill_args()
    cfg = load_config(a.config)
    if not a.skill:
        warn("missing --skill; run `assetctl skills list`")
        sys.exit(2)
    # Skill ids are "blender-<operation>" except the catalog id itself is also
    # accepted. Go always passes the full id; derive the dispatch key from it.
    operation = a.skill[len("blender-"):].replace("-", "_") if a.skill.startswith("blender-") else a.skill.replace("-", "_")
    if operation not in OPS:
        warn("unregistered skill %s (operation %s)" % (a.skill, operation))
        sys.exit(2)

    if a.inp:
        open_input(a.inp)
    else:
        clear_scene()

    result = apply_operation(operation, a.asset, a.params)
    result["skill"] = a.skill
    result["config"] = {"id": cfg.get("id", a.asset),
                        "strategy": cfg.get("strategy", "tripo"),
                        "blenderVersion": cfg.get("blenderVersion", BLENDER_VERSION)}
    if not result["ok"]:
        warn("skill %s completed with skipped optional steps" % a.skill)

    blend = a.blend_out or os.path.join("work", a.asset, "skills", a.skill + ".blend")
    ensure_dir(blend)
    bpy.ops.wm.save_as_mainfile(filepath=abs_path(blend))

    outputs = [blend] + result.pop("extraOutputs", [])
    write_contract(a.asset, "skill-" + a.skill, outputs,
                   inp=a.inp, out=a.out, blend_out=blend,
                   extra={"skill": a.skill, "operation": operation,
                          "result": result, "params": a.params,
                          "source": "skills/manifest.json"})
    emit_json({"skill": a.skill, "operation": operation,
               "ok": result["ok"], "applied": result["applied"]})
    log("skill %s done -> %s" % (a.skill, a.out))


if __name__ == "__main__":
    main()
