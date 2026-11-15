"""Normalize a concept image into a Tripo-friendly square reference.

Usage:
    python scripts/python/prepare_reference.py concept.png raw/ --size 1024

Requires: pillow  (python -m pip install pillow)
Tripo: https://studio.tripo3d.com/
"""

import argparse
import os
from pathlib import Path

from PIL import Image, ImageOps


def normalize(src, out_path, size=1024, bg=(128, 128, 128), pad=0.06):
    img = Image.open(src)
    img = ImageOps.exif_transpose(img).convert("RGBA")

    # Fit inside a square with margin.
    inner = int(size * (1 - 2 * pad))
    img.thumbnail((inner, inner), Image.LANCZOS)

    canvas = Image.new("RGBA", (size, size), bg + (255,))
    x = (size - img.width) // 2
    y = (size - img.height) // 2
    canvas.alpha_composite(img, (x, y))

    out_path.parent.mkdir(parents=True, exist_ok=True)
    canvas.convert("RGB").save(out_path, "PNG")
    print(f"[prepare_reference] {src} -> {out_path} ({size}x{size})")


def main():
    p = argparse.ArgumentParser(description="Normalize Tripo reference image")
    p.add_argument("input", help="input image")
    p.add_argument("output_dir", help="output directory")
    p.add_argument("--size", type=int, default=1024)
    p.add_argument("--bg", default="808080", help="background hex, e.g. 808080")
    args = p.parse_args()

    bg = tuple(int(args.bg[i:i + 2], 16) for i in (0, 2, 4))
    src = Path(args.input)
    out_dir = Path(args.output_dir)
    out = out_dir / f"{src.stem}.png"
    normalize(src, out, args.size, bg)


if __name__ == "__main__":
    main()
