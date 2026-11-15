"""fake_blender.py — a Blender-argv-compatible headless runner for tests.

The Go 1.27 runner normally launches real Blender 5.2 like this::

    blender --background --factory-startup --python scripts/blender/<stage>.py -- \
        --asset <id> --config <path> --in <prev> --out <out> --blend-out <blend>

In automated tests we need the SIX stages to run with NO Blender installed.
This script accepts the EXACT same argv, installs ``mock_bpy`` into
``sys.modules``, and executes the stage script with ``runpy``. Because every
stage is launched as a separate process (just like real Blender), the mock
persists scene state through the .blend/.fbx files on disk.

Usage (test harness only)::

    python testing/fake_blender.py --background --factory-startup \
        --python scripts/blender/stage_01_modeling.py -- \
        --asset hero --config assets/hero/asset.json --in "" \
        --out work/hero/01-modeling.blend --blend-out work/hero/01-modeling.blend

Exit code = 0 on success, non-zero on failure (mirrors Blender / bpy scripts).
See SPEC.md §2 (stdlib only), docs/61 (language split), testing/mock_bpy.py.
"""

import os
import runpy
import sys
import traceback

_HERE = os.path.dirname(os.path.abspath(__file__))
_REPO = os.path.dirname(_HERE)
_BLENDER_SCRIPTS = os.path.join(_REPO, "scripts", "blender")

for p in (_HERE, _BLENDER_SCRIPTS):
    if p not in sys.path:
        sys.path.insert(0, p)

import mock_bpy  # noqa: E402


def _find(argv, flag):
    """Return the token after `flag`, or None."""
    for i, tok in enumerate(argv):
        if tok == flag and i + 1 < len(argv):
            return argv[i + 1]
    return None


def main():
    argv = sys.argv[1:]
    script = _find(argv, "--python")
    if not script:
        sys.stderr.write("fake_blender: missing --python <script>\n")
        return 2
    if not os.path.isfile(script):
        sys.stderr.write("fake_blender: stage script not found: %s\n" % script)
        return 2

    # Everything after the literal `--` is the stage's own argument list.
    if "--" in argv:
        stage_args = argv[argv.index("--") + 1:]
    else:
        stage_args = []

    mock_bpy.install()
    # parse_args() in common.py expects sys.argv to contain `--` followed by
    # the stage flags. Rebuild sys.argv accordingly for the executed script.
    sys.argv = [script, "--"] + stage_args

    try:
        runpy.run_path(script, run_name="__main__")
    except SystemExit:
        raise
    except Exception:
        traceback.print_exc()
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
