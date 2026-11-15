"""Automatic tests for the 28 registered Blender skills.

Every skill is executed through testing/fake_blender.py (mock_bpy) with the
SAME blender-style argv that `assetctl skills run` emits. No Blender install,
no network, no Python from the reference repository.
"""

import glob
import json
import os
import subprocess
import sys
import tempfile
import unittest

from harness import FAKE_BLENDER, REPO, SCRIPTS

DISPATCHER = os.path.join(SCRIPTS, "skill_run.py")
MANIFEST = os.path.join(REPO, "skills", "manifest.json")
CATALOG_GO = os.path.join(REPO, "internal", "skills", "catalog.go")
ASSET = "skill-probe"


class TestSkills(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        with open(MANIFEST, "r", encoding="utf-8") as fh:
            cls.manifest = json.load(fh)
        cls.ids = [s["id"] for s in cls.manifest["skills"]]
        cls.root = tempfile.mkdtemp(prefix="assetctl-skills-unittest-")
        cls.procs = {}
        for sid in cls.ids:
            out = os.path.join(cls.root, "build", ASSET, "contracts",
                               "skill-%s.json" % sid)
            blend = os.path.join(cls.root, "work", ASSET, "skills", sid + ".blend")
            argv = [sys.executable, FAKE_BLENDER, "--background",
                    "--factory-startup", "--python", DISPATCHER, "--",
                    "--skill", sid, "--asset", ASSET, "--out", out,
                    "--blend-out", blend, "--param", "supplier=tripo"]
            cls.procs[sid] = subprocess.run(argv, cwd=cls.root,
                                            capture_output=True, text=True)

    @classmethod
    def tearDownClass(cls):
        import shutil
        shutil.rmtree(cls.root, ignore_errors=True)

    def test_manifest_metadata(self):
        self.assertEqual(self.manifest["license"], "MIT")
        self.assertEqual(self.manifest["blenderTarget"], "5.2")
        self.assertIn("EL4CTEO/blender-skills", self.manifest["source"])
        self.assertEqual(len(self.ids), 28)

    def test_all_28_skills_execute(self):
        for sid in self.ids:
            with self.subTest(skill=sid):
                p = self.procs[sid]
                self.assertEqual(p.returncode, 0,
                                 "%s failed:\n%s\n%s" % (sid, p.stdout, p.stderr))
                self.assertIn("@@JSON@@", p.stdout)

    def test_all_28_contracts(self):
        files = sorted(glob.glob(os.path.join(self.root, "build", ASSET,
                                              "contracts", "*.json")))
        self.assertEqual(len(files), 28)
        for p in files:
            with open(p, "r", encoding="utf-8") as fh:
                doc = json.load(fh)
            self.assertEqual(doc["schema"], 1)
            self.assertEqual(doc["asset"], ASSET)
            self.assertEqual(doc["stage"], "skill-" + doc["skill"])
            self.assertEqual(doc["source"], "skills/manifest.json")
            self.assertTrue(doc["outputs"], doc["skill"])
            self.assertTrue(doc["result"]["ok"], doc["skill"])
            self.assertTrue(doc["result"]["applied"], doc["skill"])
            for entry in doc["outputs"]:
                full = os.path.join(self.root, entry["path"])
                self.assertTrue(os.path.isfile(full), entry["path"])
                import hashlib
                with open(full, "rb") as fh:
                    raw = fh.read()
                self.assertEqual(len(raw), entry["bytes"])
                self.assertEqual(hashlib.sha256(raw).hexdigest(), entry["sha256"])

    def test_go_catalog_and_dispatcher_cover_manifest(self):
        with open(CATALOG_GO, "r", encoding="utf-8") as fh:
            go = fh.read()
        with open(DISPATCHER, "r", encoding="utf-8") as fh:
            py = fh.read()
        for entry in self.manifest["skills"]:
            sid = entry["id"]
            op = sid[len("blender-"):].replace("-", "_")
            self.assertIn('"%s"' % sid, go, "Go catalog missing " + sid)
            self.assertIn('"%s":' % op, py, "bpy dispatcher missing " + op)
            self.assertEqual(entry["goCommand"], "assetctl skills run " + sid)
            self.assertEqual(entry["bpyScript"],
                             "scripts/blender/skill_run.py --skill " + sid)

    def test_reference_repository_python_is_not_vendored(self):
        # MIT compliance: only SKILL.md knowledge is copied; no .py source from
        # EL4CTEO/blender-skills lives under skills/.
        py_files = glob.glob(os.path.join(REPO, "skills", "**", "*.py"),
                             recursive=True)
        self.assertEqual(py_files, [], "vendored reference Python found: %s" % py_files)
        self.assertTrue(os.path.isfile(os.path.join(
            REPO, "skills", "LICENSE-blender-skills.txt")))

    def test_documentation_files_present(self):
        for sid in self.ids:
            for name in ("SKILL.md", "ADAPTATION.md"):
                p = os.path.join(REPO, "skills", sid, name)
                self.assertTrue(os.path.isfile(p), p)
                self.assertGreater(os.path.getsize(p), 0, p)


if __name__ == "__main__":
    unittest.main()
