"""Unit tests for the mock_bpy Blender 5.2 stand-in itself."""

import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
if HERE not in sys.path:
    sys.path.insert(0, HERE)

import mock_bpy  # noqa: E402


class TestMockBpy(unittest.TestCase):
    def setUp(self):
        mock_bpy._load_state(mock_bpy._fresh_state())
        self.bpy = mock_bpy.install()

    def test_installs_bpy_and_mathutils(self):
        self.assertIn("bpy", sys.modules)
        self.assertIn("mathutils", sys.modules)
        self.assertEqual(self.bpy.app.version, (5, 2, 0))
        self.assertEqual(self.bpy.app.version_string, "5.2.0")
        self.assertTrue(self.bpy.app.background)

    def test_render_engine_enforces_5x_enum(self):
        render = self.bpy.context.scene.render
        self.assertEqual(render.engine, "BLENDER_EEVEE")
        render.engine = "BLENDER_WORKBENCH"
        self.assertEqual(render.engine, "BLENDER_WORKBENCH")
        with self.assertRaises(TypeError):
            render.engine = "BLENDER_EEVEE_NEXT"  # removed in Blender 5.x

    def test_vector_math(self):
        Vector = mock_bpy.Vector
        v = Vector((1, 2, 3)) + Vector((4, 5, 6))
        self.assertEqual(tuple(v), (5, 7, 9))
        self.assertAlmostEqual(Vector((3, 4, 0)).length, 5.0)  # property, as in mathutils
        self.assertEqual(tuple(Vector((6, 8, 10)) / 2.0), (3, 4, 5))

    def test_png_writer_creates_real_png(self):
        path = os.path.join(tempfile.mkdtemp(), "solid.png")
        mock_bpy._write_png(path, 4, 4, (1, 0, 0, 1))
        with open(path, "rb") as fh:
            data = fh.read()
        self.assertEqual(data[:8], b"\x89PNG\r\n\x1a\n")
        self.assertGreater(len(data), 32)

    def test_fbx_snapshot_has_binary_header_and_roundtrips(self):
        self.bpy.ops.mesh.primitive_cube_add(size=1)
        self.assertEqual(len(self.bpy.data.objects), 1)
        path = os.path.join(tempfile.mkdtemp(), "asset.fbx")
        mock_bpy._write_snapshot(path, fbx=True)
        with open(path, "rb") as fh:
            self.assertTrue(fh.read(20).startswith(b"Kaydara FBX Binary"))
        mock_bpy._load_state(mock_bpy._fresh_state())
        self.assertEqual(len(self.bpy.data.objects), 0)
        self.assertTrue(mock_bpy._load_snapshot(path))
        self.assertEqual(len(self.bpy.data.objects), 1)

    def test_export_fbx_records_seven_parameter_contract(self):
        path = os.path.join(tempfile.mkdtemp(), "out.fbx")
        self.bpy.ops.export_scene.fbx(
            filepath=path, axis_forward="-Z", axis_up="Y",
            global_scale=1.0, apply_unit_scale=True,
            add_leaf_bones=False, path_mode="COPY", bake_anim=True)
        log = mock_bpy._state.get("exportLog")
        self.assertEqual(len(log), 1)
        for key in ("axis_forward", "axis_up", "global_scale",
                    "apply_unit_scale", "add_leaf_bones", "path_mode",
                    "bake_anim"):
            self.assertIn(key, log[0])

    def test_scene_state_is_pickled_between_stage_processes(self):
        self.bpy.ops.mesh.primitive_cube_add(size=2)
        self.bpy.data.materials.new(name="M_Test")
        path = os.path.join(tempfile.mkdtemp(), "01-modeling.blend")
        mock_bpy._write_snapshot(path)
        mock_bpy._load_state(mock_bpy._fresh_state())
        self.assertTrue(mock_bpy._load_snapshot(path))
        self.assertEqual(len(self.bpy.data.materials), 1)
        self.assertEqual(self.bpy.data.materials[0].name, "M_Test")


if __name__ == "__main__":
    unittest.main()
