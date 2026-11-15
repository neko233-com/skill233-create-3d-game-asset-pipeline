"""rig_check.py — Unity-Humanoid rig validator (SPEC.md §5 gate, docs/50).

Called by `assetctl run` after stage 5 for `humanoid: true` assets:

    blender --background --factory-startup --python scripts/blender/rig_check.py \
        -- --input build/<id>/<id>.fbx

Also accepts the stage-style form (docs/50 example):

    ... -- --asset <id> --config <asset.json> --in build/<id>/<id>.fbx

So `--input` and `--in` are aliases; `--asset`/`--config` are optional.
Exits 0 on pass, non-zero on any failure (Go runner treats non-zero as gate fail).

Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, parse_args, open_input, first_armature, write_contract, derive_asset,
)

REQUIRED = [
    "Hips", "Spine", "Chest", "Neck", "Head",
    "LeftShoulder", "LeftUpperArm", "LeftLowerArm", "LeftHand",
    "RightShoulder", "RightUpperArm", "RightLowerArm", "RightHand",
    "LeftUpperLeg", "LeftLowerLeg", "LeftFoot", "LeftToes",
    "RightUpperLeg", "RightLowerLeg", "RightFoot", "RightToes",
]


def norm(name):
    return name.strip().lower().replace(" ", "").replace("_", "")


def fail(msg):
    print("[rig_check] FAIL: %s" % msg)
    sys.stdout.flush()
    sys.exit(1)


def main():
    a = parse_args()
    inp = a.inp
    if not inp:
        # Last-resort: derive from --asset if only stage-style args were given.
        if a.asset:
            inp = os.path.join("build", a.asset, a.asset + ".fbx")
    if not inp:
        fail("no input FBX (pass --input / --in, or --asset)")

    path = os.path.abspath(inp)
    if not os.path.isfile(path):
        fail("input not found: %s" % path)
    ext = os.path.splitext(path)[1].lower()
    if ext not in (".fbx", ".glb", ".gltf", ".blend"):
        fail("unsupported format %s" % ext)

    open_input(path)  # clear + import (FBX/GLB) or open .blend

    arm = first_armature()
    if arm is None:
        fail("no armature found in %s" % inp)

    bones = {norm(b.name): b for b in arm.data.bones}
    log("armature=%s bones=%d" % (arm.name, len(bones)))

    missing = [b for b in REQUIRED if norm(b) not in bones]
    if missing:
        print("[rig_check] FAIL: missing required bones:")
        for m in missing:
            print("  - %s" % m)
        sys.stdout.flush()
        sys.exit(1)

    names = [b.name for b in arm.data.bones]
    if len(names) != len(set(names)):
        fail("duplicate bone names")

    print("[rig_check] OK: all required Humanoid bones present")
    for key in ("hips", "spine", "head", "leftupperarm", "rightupperarm",
                "leftupperleg", "rightupperleg"):
        b = bones[key]
        print("  %s: head=%s" % (b.name, tuple(round(v, 3) for v in b.head_local)))
    sys.stdout.flush()
    # Gate result is a first-class, machine-checkable artefact.
    write_contract(a.asset or derive_asset(path), "rig-check", [path],
                   inp=path, out=path,
                   extra={"gate": "humanoid", "result": "ok",
                          "requiredBones": len(REQUIRED), "bones": len(bones)})


if __name__ == "__main__":
    main()
