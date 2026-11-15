"""humanoid_rig.py — Unity-Humanoid skeleton + placeholder clips.

Used by stage_01_modeling.py when asset.json has `humanoid: true`.
Bone names / hierarchy follow docs/50-humanoid-animation.md §2 and are checked
by rig_check.py (SPEC.md §5 gate). Runs inside Blender 5.2 native Python.

Route B in docs/50: script-authored rig (vs Tripo auto-rig at
https://studio.tripo3d.com/ ). The procedural armature is a deterministic
stand-in; replace bone positions / actions with a real bind + mocap for
production, keeping the SAME names so Avatar mapping stays stable.
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402
from mathutils import Vector  # noqa: E402  (bundled with Blender's Python)

from common import log, set_active  # noqa: E402

# (name, parent, head_xyz, tail_xyz) — Z-up rest pose, ~1.7 units tall.
BONES = [
    ("Hips", None, (0, 0, 0.95), (0, 0, 1.05)),
    ("Spine", "Hips", (0, 0, 1.05), (0, 0, 1.28)),
    ("Chest", "Spine", (0, 0, 1.28), (0, 0, 1.50)),
    ("Neck", "Chest", (0, 0, 1.50), (0, 0, 1.60)),
    ("Head", "Neck", (0, 0, 1.60), (0, 0, 1.80)),
    ("LeftShoulder", "Chest", (0, 0, 1.48), (0.18, 0, 1.48)),
    ("LeftUpperArm", "LeftShoulder", (0.18, 0, 1.48), (0.45, 0, 1.48)),
    ("LeftLowerArm", "LeftUpperArm", (0.45, 0, 1.48), (0.70, 0, 1.48)),
    ("LeftHand", "LeftLowerArm", (0.70, 0, 1.48), (0.85, 0, 1.48)),
    ("RightShoulder", "Chest", (0, 0, 1.48), (-0.18, 0, 1.48)),
    ("RightUpperArm", "RightShoulder", (-0.18, 0, 1.48), (-0.45, 0, 1.48)),
    ("RightLowerArm", "RightUpperArm", (-0.45, 0, 1.48), (-0.70, 0, 1.48)),
    ("RightHand", "RightLowerArm", (-0.70, 0, 1.48), (-0.85, 0, 1.48)),
    ("LeftUpperLeg", "Hips", (0.12, 0, 0.92), (0.12, 0, 0.52)),
    ("LeftLowerLeg", "LeftUpperLeg", (0.12, 0, 0.52), (0.12, 0, 0.12)),
    ("LeftFoot", "LeftLowerLeg", (0.12, 0, 0.12), (0.12, 0.18, 0.05)),
    ("LeftToes", "LeftFoot", (0.12, 0.18, 0.05), (0.12, 0.30, 0.04)),
    ("RightUpperLeg", "Hips", (-0.12, 0, 0.92), (-0.12, 0, 0.52)),
    ("RightLowerLeg", "RightUpperLeg", (-0.12, 0, 0.52), (-0.12, 0, 0.12)),
    ("RightFoot", "RightLowerLeg", (-0.12, 0, 0.12), (-0.12, 0.18, 0.05)),
    ("RightToes", "RightFoot", (-0.12, 0.18, 0.05), (-0.12, 0.30, 0.04)),
]

REQUIRED = [b[0] for b in BONES]


def create_humanoid_rig(asset, animations):
    """Build the armature object and its edit bones. Returns the Object."""
    arm_data = bpy.data.armatures.new(name=asset + "_Armature")
    arm = bpy.data.objects.new(name=asset + "_Rig", object_data=arm_data)
    bpy.context.collection.objects.link(arm)
    set_active(arm)

    bpy.ops.object.mode_set(mode="EDIT")
    for name, parent, head, tail in BONES:
        eb = arm_data.edit_bones.new(name)
        eb.head = Vector(head)
        eb.tail = Vector(tail)
        if parent:
            eb.parent = arm_data.edit_bones[parent]
            eb.use_connect = False
    bpy.ops.object.mode_set(mode="OBJECT")
    arm.show_in_front = True
    log("created humanoid rig %s (%d bones)" % (arm.name, len(BONES)))
    return arm


def attach_mesh(obj, arm):
    """Parent a mesh to the armature with an ARMATURE modifier (no weights yet)."""
    try:
        obj.parent = arm
        mod = obj.modifiers.new(name="Armature", type="ARMATURE")
        mod.object = arm
        if not obj.vertex_groups:
            obj.vertex_groups.new(name="Hips")
        log("attached mesh %s -> %s" % (obj.name, arm.name))
    except Exception as exc:
        log("WARN attach_mesh(%s): %s" % (obj.name, exc))


def create_clips(arm, asset, animations):
    """One NLA action per clip, named `<asset>-<action>` (SPEC.md §11).

    A tiny 2-key rest-pose loop is enough for Unity to import each clip and to
    pass rig_check.py / `bake_anim=True`; replace with real animation curves.
    """
    actions = animations or ["idle"]
    arm.animation_data_create()
    for i, action_name in enumerate(actions):
        clip = "%s-%s" % (asset, action_name)
        act = bpy.data.actions.new(name=clip)
        try:
            # Assign first: keyframe_insert writes into the active action.
            arm.animation_data.action = act
            # Two identical keys -> a valid, loopable rest-pose placeholder.
            for pb in arm.pose.bones:
                for frame in (1, 24):
                    try:
                        pb.keyframe_insert(data_path="location", frame=frame,
                                           group=pb.name)
                    except Exception:
                        pass
            track = arm.animation_data.nla_tracks.new()
            track.name = clip
            track.strips.new(name=clip, start=1 + i * 30, action=act)
        except Exception as exc:
            log("WARN clip %s: %s" % (clip, exc))
        log("created clip %s" % clip)


if __name__ == "__main__":
    # Standalone execution under testing/fake_blender.py: build a complete
    # Unity-Humanoid rig, persist it and publish a contract like every stage.
    from common import (  # noqa: E402
        parse_args, derive_asset, write_contract, ensure_dir, abs_path,
    )

    a = parse_args()
    asset = a.asset or derive_asset(a.inp or a.out) or "rig-probe"
    clips = ["idle", "walk", "attack"]
    arm = create_humanoid_rig(asset, clips)
    create_clips(arm, asset, clips)
    out = a.blend_out or ("work/%s/humanoid-rig.blend" % asset)
    ensure_dir(out)
    bpy.ops.wm.save_as_mainfile(filepath=abs_path(out))
    write_contract(asset, "humanoid-rig", [out], inp=a.inp, out=out,
                   blend_out=out,
                   extra={"bones": len(arm.data.bones), "clips": clips})
