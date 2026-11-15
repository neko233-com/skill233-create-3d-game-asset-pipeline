"""Shared harness for stage-level Python tests.

Runs the REAL scripts/blender/*.py modules through testing/fake_blender.py with
mock_bpy installed — no Blender, no GPU, no network, stdlib only. Each stage is
a separate process, exactly like `assetctl run` with real Blender 5.2.
"""

import glob
import json
import os
import struct
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
FAKE_BLENDER = os.path.join(HERE, "fake_blender.py")
SCRIPTS = os.path.join(REPO, "scripts", "blender")
ASSET = "hero-knight"

STAGES = [
    ("modeling", "stage_01_modeling.py", "01-modeling.blend"),
    ("topology-uv", "stage_02_topology_uv.py", "02-topology-uv.blend"),
    ("bake", "stage_03_bake.py", "03-bake.blend"),
    ("pbr", "stage_04_pbr.py", "04-pbr.blend"),
    ("unity-export", "stage_05_unity_export.py", "05-unity-export.blend"),
    ("render", "stage_06_render.py", "06-render.blend"),
]

MAIN_SCRIPTS = ["common.py", "stage_01_modeling.py", "stage_02_topology_uv.py",
                "stage_03_bake.py", "stage_04_pbr.py", "stage_05_unity_export.py",
                "stage_06_render.py", "rig_check.py", "humanoid_rig.py",
                "tripo_to_unity.py"]


def config_bytes(asset=ASSET):
    return {
        "id": asset, "strategy": "tripo", "kind": "character", "humanoid": True,
        "referenceImage": "raw/%s.png" % asset, "source": "",
        "tripoUrl": "https://studio.tripo3d.com/", "blenderVersion": "5.2",
        "targetEngine": "unity-urp",
        "axes": {"forward": "-Z", "up": "Y", "scale": 1.0},
        "budget": {"maxTriangles": 30000, "maxTextureSize": 256},
        "textures": ["basecolor", "normal", "roughness", "metallic"],
        "animations": ["idle", "walk", "attack"],
        "modeling": {"primitive": "cube", "size": 1.0, "subdivisions": 1},
        "stages": [s[0] for s in STAGES],
        "handoff": {"mode": "fixture", "source": ""},
    }


def minimal_glb(asset=ASSET):
    """A valid, single-triangle glTF 2.0 binary — mirrors assetctl fixture."""
    import struct as _struct
    positions = [-0.5, 0, 0, 0.5, 0, 0, 0, 1, 0]
    bin_part = _struct.pack("<9f", *positions) + _struct.pack("<3H", 0, 1, 2) + b"\x00\x00"
    doc = {
        "asset": {"version": "2.0"}, "scene": 0,
        "scenes": [{"nodes": [0]}], "nodes": [{"name": asset, "mesh": 0}],
        "meshes": [{"primitives": [{"attributes": {"POSITION": 0}, "indices": 1}]}],
        "buffers": [{"byteLength": len(bin_part)}],
        "bufferViews": [
            {"buffer": 0, "byteOffset": 0, "byteLength": 36},
            {"buffer": 0, "byteOffset": 36, "byteLength": 6},
        ],
        "accessors": [
            {"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"},
            {"bufferView": 1, "componentType": 5123, "count": 3, "type": "SCALAR"},
        ],
    }
    j = json.dumps(doc, sort_keys=True).encode()
    while len(j) % 4:
        j += b" "
    total = 12 + 8 + len(j) + 8 + len(bin_part)
    return (struct.pack("<III", 0x46546C67, 2, total) +
            struct.pack("<II", len(j), 0x4E4F534A) + j +
            struct.pack("<II", len(bin_part), 0x004E4942) + bin_part)


class PipelineContext(object):
    def __init__(self):
        self.root = tempfile.mkdtemp(prefix="assetctl-unittest-")
        self.asset = ASSET
        self.cfg = os.path.join(self.root, "assets", ASSET, "asset.json")
        os.makedirs(os.path.dirname(self.cfg), exist_ok=True)
        with open(self.cfg, "w", encoding="utf-8") as fh:
            json.dump(config_bytes(), fh)
        os.makedirs(os.path.join(self.root, "raw"), exist_ok=True)
        self.glb = os.path.join(self.root, "raw", ASSET + ".glb")
        with open(self.glb, "wb") as fh:
            fh.write(minimal_glb())
        self.work = os.path.join(self.root, "work", ASSET)
        self.build = os.path.join(self.root, "build", ASSET)
        os.makedirs(self.work)
        os.makedirs(self.build)
        self.results = {}

    def run_script(self, script, extra):
        argv = [sys.executable, FAKE_BLENDER, "--background", "--factory-startup",
                "--python", os.path.join(SCRIPTS, script), "--"] + extra
        proc = subprocess.run(argv, cwd=self.root, capture_output=True, text=True)
        self.results.setdefault(script, []).append(proc)
        return proc

    def stage_args(self, stage, inp, out, blend_name):
        return ["--asset", self.asset, "--config", self.cfg,
                "--in", inp, "--out", out,
                "--blend-out", os.path.join(self.work, blend_name)]

    def run_pipeline(self):
        prev = self.glb
        for stage, script, blend_name in STAGES:
            if stage == "unity-export":
                out = os.path.join(self.build, self.asset + ".fbx")
            elif stage == "render":
                out = os.path.join(self.build, "preview.png")
            else:
                out = os.path.join(self.work, blend_name)
            proc = self.run_script(script, self.stage_args(stage, prev, out, blend_name))
            if proc.returncode != 0:
                raise AssertionError("%s failed (%d):\n%s\n%s" %
                                     (script, proc.returncode, proc.stdout, proc.stderr))
            prev = out
        return self

    def run_auxiliary_scripts(self):
        """The other four scripts/blender modules also execute under mock_bpy."""
        fbx = os.path.join(self.build, self.asset + ".fbx")
        # rig_check reads the delivered FBX (humanoid gate).
        p = self.run_script("rig_check.py", ["--input", fbx])
        assert p.returncode == 0, "rig_check failed: %s %s" % (p.stdout, p.stderr)
        # common.py library module: publishes its constants as a contract.
        p = self.run_script("common.py", ["--asset", self.asset, "--out",
                                          os.path.join(self.build, "common.out")])
        assert p.returncode == 0, "common.py failed: %s %s" % (p.stdout, p.stderr)
        # humanoid_rig.py standalone: builds a rig + its own .blend snapshot.
        p = self.run_script("humanoid_rig.py", ["--asset", self.asset])
        assert p.returncode == 0, "humanoid_rig.py failed: %s %s" % (p.stdout, p.stderr)
        # tripo_to_unity.py shim: use a SEPARATE fbx name so it cannot overwrite
        # (and invalidate the SHA-256 of) the six-stage deliverable.
        shim = os.path.join(self.build, self.asset + "-shim.fbx")
        p = self.run_script("tripo_to_unity.py", [
            "--asset", self.asset, "--config", self.cfg, "--in", self.glb,
            "--out", shim])
        assert p.returncode == 0, "tripo_to_unity.py failed: %s %s" % (p.stdout, p.stderr)
        return self

    def contract(self, stage):
        p = os.path.join(self.build, "contracts", stage + ".json")
        with open(p, "r", encoding="utf-8") as fh:
            return json.load(fh)

    def contract_files(self):
        return sorted(glob.glob(os.path.join(self.build, "contracts", "*.json")))

    def cleanup(self):
        import shutil
        shutil.rmtree(self.root, ignore_errors=True)
