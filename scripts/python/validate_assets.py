"""Validate a built asset directory before Unity import.

Usage:
    python scripts/python/validate_assets.py build/hero --expect-humanoid
    python scripts/python/validate_assets.py build/hero --expect-textures

Exit code 0 = all checks pass; non-zero = problems listed.
"""

import argparse
import re
import sys
from pathlib import Path

ASSET_ID_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
MODEL_EXTS = {".fbx", ".glb", ".gltf", ".obj"}
PBR_HINTS = ("basecolor", "albedo", "normal", "roughness", "metallic", "ao")


def main():
    p = argparse.ArgumentParser(description="Validate game asset directory")
    p.add_argument("directory")
    p.add_argument("--expect-humanoid", action="store_true")
    p.add_argument("--expect-textures", action="store_true")
    args = p.parse_args()

    d = Path(args.directory)
    issues = []
    if not d.is_dir():
        print(f"[validate] FAIL: not a directory: {d}")
        sys.exit(1)

    asset_id = d.name
    if not ASSET_ID_RE.match(asset_id):
        issues.append(f"asset dir name not lowercase-kebab: {asset_id!r}")

    files = [f for f in d.rglob("*") if f.is_file()]
    models = [f for f in files if f.suffix.lower() in MODEL_EXTS]
    if not models:
        issues.append("no model file (.fbx/.glb/.gltf/.obj)")

    textures = [f for f in files if f.suffix.lower() in {".png", ".jpg", ".jpeg", ".tga"}]
    if args.expect_textures and not textures:
        issues.append("--expect-textures set but no texture files")

    for t in textures:
        if not any(h in t.stem.lower() for h in PBR_HINTS):
            issues.append(f"texture lacks PBR suffix: {t.name}")

    if args.expect_humanoid:
        marker = d / ".humanoid"
        named = any("humanoid" in f.name.lower() or "rig" in f.name.lower() for f in files)
        if not (marker.exists() or named or asset_id.startswith("hero") or "character" in asset_id):
            issues.append("--expect-humanoid set but no rig marker found (add empty .humanoid file)")

    if issues:
        print(f"[validate] FAIL for {d}:")
        for i in issues:
            print(f"  - {i}")
        sys.exit(1)

    print(f"[validate] OK: {asset_id} ({len(models)} model(s), {len(textures)} texture(s))")


if __name__ == "__main__":
    main()
