"""Compatibility shim — the real verifier is Go 1.27 (`assetctl verify`).

Per AGENTS.md the delivery pipeline contains no system-Python code outside
Blender's bundled bpy. This file exists only so the documented
`python scripts/verify.py` command keeps working; every check (build, vet,
race, coverage, stage-level Python tests, doctor, CLI smoke) is implemented in
internal/verify/verify.go and written in Go.

Usage:  python scripts/verify.py [--root <repo-root>]
"""

import os
import subprocess
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def main():
    args = ["go", "run", "./cmd/assetctl", "verify", "--root", REPO]
    if len(sys.argv) > 1 and sys.argv[1] == "--root" and len(sys.argv) > 2:
        args[-1] = os.path.abspath(sys.argv[2])
    return subprocess.call(args, cwd=REPO)


if __name__ == "__main__":
    sys.exit(main())
