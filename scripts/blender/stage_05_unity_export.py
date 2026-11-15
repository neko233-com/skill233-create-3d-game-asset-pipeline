"""Stage 5 · Unity export (SPEC.md §4/§5/§9, docs/42).

Exports MESH + ARMATURE to build/<id>/<id>.fbx with the exact seven FBX
parameters from SPEC.md §9 (single source of truth: common.FBX_EXPORT).
Unity import MUST use Bake Axis Conversion; do NOT use ACT's
"Unity (Pre-Rotated)" profile (it cannot carry animations).

Also writes the empty `.humanoid` marker for `assetctl validate
--expect-humanoid` when the asset is humanoid.

Input : work/<id>/04-pbr.blend
Output: build/<id>/<id>.fbx (+ PBR PNGs already on disk)
Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, parse_args, load_config, open_input, save_blend, ensure_textures,
    export_fbx, write_humanoid_marker, build_dir,
)


def main():
    a = parse_args()
    cfg = load_config(a.config)
    open_input(a.inp)

    humanoid = bool(cfg.get("humanoid"))
    # Guarantee declared channels exist even if stage 3 was skipped via --from.
    ensure_textures(cfg, a.asset)

    fbx = os.path.join(build_dir(a.asset), a.asset + ".fbx")
    export_fbx(fbx, cfg=cfg, humanoid=humanoid)

    if humanoid:
        write_humanoid_marker(a.asset)

    save_blend(a, "unity-export")  # --blend-out work/<id>/05-unity-export.blend (if .blend)
    log("done -> %s" % a.out)


if __name__ == "__main__":
    main()
