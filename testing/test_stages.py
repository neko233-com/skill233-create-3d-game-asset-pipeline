"""Stage-level tests: all 10 scripts/blender modules, their contracts and the
Blender 5.x API rules. Runs on stock CPython with mock_bpy / fake_blender.
"""

import hashlib
import json
import os
import unittest

from harness import ASSET, MAIN_SCRIPTS, SCRIPTS, STAGES, PipelineContext

FBX_KEYS = ("axis_forward", "axis_up", "global_scale", "apply_unit_scale",
            "add_leaf_bones", "path_mode")


class TestStagePipeline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.ctx = PipelineContext().run_pipeline().run_auxiliary_scripts()

    @classmethod
    def tearDownClass(cls):
        cls.ctx.cleanup()

    def test_all_six_stages_succeeded(self):
        for _stage, script, _blend in STAGES:
            procs = self.ctx.results.get(script, [])
            self.assertEqual(len(procs), 1, "%s not executed exactly once" % script)
            self.assertEqual(procs[0].returncode, 0,
                             "%s failed: %s" % (script, procs[0].stderr))

    def test_all_ten_scripts_executed(self):
        # Every module in scripts/blender is executable under mock_bpy, not
        # just the six pipeline stages (SPEC.md §3A.2 / §5A).
        self.assertEqual(len(MAIN_SCRIPTS), 10)
        for script in MAIN_SCRIPTS:
            self.assertIn(script, self.ctx.results, "%s never ran" % script)
            p = self.ctx.results[script][0]
            self.assertEqual(p.returncode, 0,
                             "%s failed: %s\n%s" % (script, p.stdout, p.stderr))

    def test_ten_contract_files(self):
        files = self.ctx.contract_files()
        self.assertEqual(len(files), 10, "want 10 stage contracts, got %s" % files)
        for p in files:
            with open(p, "r", encoding="utf-8") as fh:
                doc = json.load(fh)
            self.assertEqual(doc["schema"], 1)
            self.assertEqual(doc["asset"], ASSET)
            self.assertEqual(doc["stage"], os.path.splitext(os.path.basename(p))[0])

    def test_stage_contracts_are_complete(self):
        for stage, _script, _blend in STAGES:
            with self.subTest(stage=stage):
                doc = self.ctx.contract(stage)
                self.assertEqual(doc["schema"], 1)
                self.assertEqual(doc["stage"], stage)
                self.assertEqual(doc["asset"], ASSET)
                self.assertEqual(doc["tripoUrl"], "https://studio.tripo3d.com/")
                self.assertEqual(doc["blenderVersion"], "5.2")
                self.assertTrue(doc["outputs"], "%s produced no files" % stage)
                for key in FBX_KEYS:
                    self.assertIn(key, doc["fbxExport"])

    def test_contract_digests_match_real_files(self):
        checked = 0
        for p in self.ctx.contract_files():
            with open(p, "r", encoding="utf-8") as fh:
                doc = json.load(fh)
            for entry in doc["outputs"]:
                full = os.path.join(self.ctx.root, entry["path"])
                self.assertTrue(os.path.isfile(full), "missing %s" % entry["path"])
                with open(full, "rb") as fh:
                    raw = fh.read()
                self.assertEqual(len(raw), entry["bytes"], entry["path"])
                self.assertEqual(hashlib.sha256(raw).hexdigest(), entry["sha256"],
                                 entry["path"])
                checked += 1
        self.assertGreater(checked, 20, "expected many contracted artifacts")

    def test_unity_fbx_export_parameters(self):
        # SPEC.md §9: the seven-parameter contract is identical in every stage
        # contract, so no script can silently diverge from the Unity axes.
        for stage, _script, _blend in STAGES:
            with self.subTest(stage=stage):
                e = self.ctx.contract(stage)["fbxExport"]
                self.assertEqual(e["axis_forward"], "-Z")
                self.assertEqual(e["axis_up"], "Y")
                self.assertEqual(e["global_scale"], 1.0)
                self.assertTrue(e["apply_unit_scale"])
                self.assertFalse(e["add_leaf_bones"])
                self.assertEqual(e["path_mode"], "COPY")

    def test_deliverable_magic_bytes(self):
        fbx = os.path.join(self.ctx.build, ASSET + ".fbx")
        with open(fbx, "rb") as fh:
            self.assertTrue(fh.read(20).startswith(b"Kaydara FBX Binary"))
        for name in ("preview.png", ASSET + "_basecolor.png",
                     ASSET + "_normal.png", ASSET + "_roughness.png",
                     ASSET + "_metallic.png", ASSET + "_ao.png"):
            with open(os.path.join(self.ctx.build, name), "rb") as fh:
                self.assertEqual(fh.read(8), b"\x89PNG\r\n\x1a\n", name)

    def test_structured_json_output(self):
        for _stage, script, _blend in STAGES:
            p = self.ctx.results[script][0]
            self.assertIn("@@JSON@@", p.stdout,
                          "%s must emit a machine-readable contract line" % script)

    def test_humanoid_gate_contract(self):
        doc = self.ctx.contract("rig-check")
        self.assertEqual(doc["result"], "ok")
        self.assertEqual(doc["gate"], "humanoid")
        self.assertEqual(doc["requiredBones"], 21)

    def test_no_deprecated_5x_api(self):
        # Rules from skills/blender-core/SKILL.md (5.x API changes that break
        # old tutorials). Regression guard for real Blender 5.2 compatibility.
        forbidden = ("BLENDER_EEVEE_NEXT", "scene.node_tree", "action.fcurves",
                     "bgl", "read_factory_settings", "import PIL", "pip install")
        for name in sorted(os.listdir(SCRIPTS)):
            if not name.endswith(".py"):
                continue
            with open(os.path.join(SCRIPTS, name), "r", encoding="utf-8") as fh:
                src = fh.read()
            for token in forbidden:
                self.assertNotIn(token, src, "%s uses deprecated %s" % (name, token))

    def test_all_scripts_compile_with_stdlib_only(self):
        import py_compile
        for name in MAIN_SCRIPTS:
            py_compile.compile(os.path.join(SCRIPTS, name), doraise=True)


if __name__ == "__main__":
    unittest.main()
