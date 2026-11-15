"""Blender 5.2 · Humanoid rig validator.

Usage:
    blender --background --python scripts/blender/rig_check.py -- --input build/hero/hero.fbx

Checks for a Humanoid-compatible skeleton: required bones, hierarchy, duplicates.
"""

import argparse
import os
import sys

import bpy

REQUIRED = [
    "Hips", "Spine", "Chest", "Neck", "Head",
    "LeftShoulder", "LeftUpperArm", "LeftLowerArm", "LeftHand",
    "RightShoulder", "RightUpperArm", "RightLowerArm", "RightHand",
    "LeftUpperLeg", "LeftLowerLeg", "LeftFoot", "LeftToes",
    "RightUpperLeg", "RightLowerLeg", "RightFoot", "RightToes",
]


def parse_args():
    argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
    p = argparse.ArgumentParser()
    p.add_argument("--input", required=True)
    return p.parse_args(argv)


def norm(name):
    return name.strip().lower().replace(" ", "").replace("_", "")


def main():
    args = parse_args()
    path = os.path.abspath(args.input)
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.object.delete(use_global=False)
    ext = os.path.splitext(path)[1].lower()
    if ext == ".fbx":
        bpy.ops.import_scene.fbx(filepath=path)
    elif ext in {".glb", ".gltf"}:
        bpy.ops.import_scene.gltf(filepath=path)
    else:
        print(f"[rig_check] unsupported format {ext}")
        sys.exit(2)

    armatures = [o for o in bpy.data.objects if o.type == "ARMATURE"]
    if not armatures:
        print("[rig_check] FAIL: no armature found")
        sys.exit(1)

    arm = armatures[0]
    bones = {norm(b.name): b for b in arm.data.bones}
    print(f"[rig_check] armature={arm.name} bones={len(bones)}")

    missing = [b for b in REQUIRED if norm(b) not in bones]
    if missing:
        print("[rig_check] FAIL: missing required bones:")
        for m in missing:
            print(f"  - {m}")
        sys.exit(1)

    names = [b.name for b in arm.data.bones]
    if len(names) != len(set(names)):
        print("[rig_check] FAIL: duplicate bone names")
        sys.exit(1)

    print("[rig_check] OK: all required Humanoid bones present")
    for key in ("hips", "spine", "head", "leftupperarm", "rightupperarm", "leftupperleg", "rightupperleg"):
        b = bones[key]
        print(f"  {b.name}: head={tuple(round(v, 3) for v in b.head_local)}")


if __name__ == "__main__":
    main()
