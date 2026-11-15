"""tripo_to_unity.py — thin compatibility wrapper (SPEC.md §3/§5/§9).

In the new multi-strategy architecture the FULL pipeline is driven by the
Go 1.27 runner (`assetctl run`), which executes stage_01..stage_06. Tripo is
just ONE modeling strategy; `raw/<id>.glb` is downloaded manually from
https://studio.tripo3d.com/ and flows through the shared stages.

This file remains as a light, single-step helper for quick one-off conversion
of a raw Tripo GLB straight to a Unity FBX. It deliberately reuses
common.export_fbx (the SPEC.md §9 seven-parameter contract) instead of
duplicating export settings. For the complete pipeline use:

    go run ./cmd/assetctl init hero --strategy tripo --humanoid
    # ... download GLB from https://studio.tripo3d.com/ to raw/hero.glb
    go run ./cmd/assetctl run hero

Direct usage:
    blender --background --factory-startup --python scripts/blender/tripo_to_unity.py -- \
        --input raw/hero.glb --output build/hero/hero.fbx [--humanoid]

Runs inside Blender 5.2 native Python (bpy + stdlib only).
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import bpy  # noqa: E402

from common import (  # noqa: E402
    log, warn, parse_args, load_config, open_input, cleanup_scene, export_fbx,
    ensure_textures, write_humanoid_marker, TRIPO_URL,
    write_contract, derive_asset, collect_outputs,
)


def parse_compat_args():
    """Accept both the new stage flags and the legacy --input/--output pair."""
    a = parse_args()
    # Legacy CLI: --output <fbx> (stage-style --out is the same value).
    argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
    for i, tok in enumerate(argv):
        if tok == "--output" and i + 1 < len(argv):
            a.out = argv[i + 1]
        if tok == "--humanoid":
            a.humanoid = True
    if not getattr(a, "humanoid", False):
        a.humanoid = False
    return a


def main():
    a = parse_compat_args()
    cfg = load_config(a.config)
    if not a.inp:
        warn("no --input; expected a Tripo export from %s" % TRIPO_URL)
    open_input(a.inp)
    cleanup_scene(cfg)

    fbx = a.out or os.path.join("build", a.asset or "asset",
                                (a.asset or "asset") + ".fbx")
    export_fbx(fbx, cfg=cfg, humanoid=a.humanoid or bool(cfg.get("humanoid")))
    if a.asset:
        ensure_textures(cfg, a.asset)
        if a.humanoid or cfg.get("humanoid"):
            write_humanoid_marker(a.asset)
    log("done -> %s (full pipeline: assetctl run <asset-id>)" % fbx)
    asset = a.asset or derive_asset(fbx)
    a.asset = asset
    write_contract(asset, "tripo-to-unity", [fbx] + collect_outputs(a),
                   inp=a.inp, out=fbx, blend_out=a.blend_out,
                   extra={"supplier": "tripo", "role": "standalone-shim"})


if __name__ == "__main__":
    main()
