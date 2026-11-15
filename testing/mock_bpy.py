"""mock_bpy.py — a deterministic, dependency-free stand-in for Blender 5.2's
``bpy`` + ``mathutils`` used ONLY by the automated test harness.

It is installed into ``sys.modules`` by ``testing/fake_blender.py`` before a
stage script is executed, so ``import bpy`` inside ``scripts/blender/*.py``
resolves to this mock instead of a real Blender install. The pipeline then runs
end-to-end on a stock CPython (no Blender, no GPU, no network).

Design notes
------------
* Every stage runs in a SEPARATE process (the Go runner launches Blender once
  per stage). To carry scene state across processes, the ``.blend`` / ``.fbx``
  files produced here are simply ``pickle`` snapshots of a plain-data scene
  graph; ``open_mainfile`` / the import operators restore them. This makes the
  mock faithful to the real file-based hand-off contract between stages.
* ``.png`` outputs are written as real, valid PNG files (stdlib ``zlib`` +
  ``struct`` only) so Go's ``validate`` / ``reference`` and the runner's
  output-existence checks all behave as in production.
* bpy + stdlib only in the *stage scripts*; this mock itself uses stdlib only
  and is NEVER shipped inside Blender. See SPEC.md §2 and docs/61.
"""

import os
import pickle
import struct
import sys
import types
import zlib


# ---------------------------------------------------------------------------
# Scene state (pickled into .blend / .fbx snapshots)
# ---------------------------------------------------------------------------
def _fresh_state():
    return {
        "objects": [], "meshes": [], "armatures": [], "materials": [],
        "images": [], "curves": [], "cameras": [], "lights": [], "actions": [],
    }


_state = _fresh_state()


def _load_state(new):
    # Mutate in place so the Collection objects (which look up _state[key]
    # dynamically) keep working after a load.
    _state.clear()
    _state.update(new)


# Real FBX files start with this 23+4 byte binary header. The mock writes it
# before the pickle payload so Go's integrity checks see a plausible FBX,
# while _load_snapshot can still restore the scene graph from the same file.
FBX_HEADER = b"Kaydara FBX Binary  \x00\x1a\x00" + struct.pack("<I", 7500)


def _load_snapshot(path):
    try:
        with open(path, "rb") as fh:
            prefix = fh.read(len(FBX_HEADER))
            fh.seek(len(FBX_HEADER) if prefix == FBX_HEADER else 0)
            loaded = pickle.load(fh)
        if isinstance(loaded, dict) and "objects" in loaded:
            _load_state(loaded)
            return True
    except Exception:
        pass
    return False


def _write_snapshot(path, fbx=False):
    d = os.path.dirname(os.path.abspath(path))
    if d:
        os.makedirs(d, exist_ok=True)
    with open(path, "wb") as fh:
        if fbx:
            fh.write(FBX_HEADER)
        pickle.dump(_state, fh)


# ---------------------------------------------------------------------------
# mathutils
# ---------------------------------------------------------------------------
class Vector(object):
    def __init__(self, seq):
        vals = list(seq)
        self.x = float(vals[0]) if len(vals) > 0 else 0.0
        self.y = float(vals[1]) if len(vals) > 1 else 0.0
        self.z = float(vals[2]) if len(vals) > 2 else 0.0

    def __iter__(self):
        return iter((self.x, self.y, self.z))

    def __repr__(self):
        return "Vector((%g, %g, %g))" % (self.x, self.y, self.z)

    def _bin(self, other, op):
        return Vector((op(self.x, other.x), op(self.y, other.y), op(self.z, other.z)))

    def __add__(self, other):
        return self._bin(other, lambda a, b: a + b)

    def __sub__(self, other):
        return self._bin(other, lambda a, b: a - b)

    def __truediv__(self, scalar):
        return Vector((self.x / scalar, self.y / scalar, self.z / scalar))

    @property
    def length(self):
        return (self.x ** 2 + self.y ** 2 + self.z ** 2) ** 0.5

    def to_track_quat(self, track, up):
        return _Quat()


class _Quat(object):
    def to_euler(self):
        return Vector((0, 0, 0))


class Matrix(object):
    """Identity-ish matrix: only ``matrix_world @ Vector`` is used by stage 6."""

    def __matmul__(self, other):
        return other


# ---------------------------------------------------------------------------
# PNG writer (real PNG so downstream checks see a genuine image file)
# ---------------------------------------------------------------------------
def _png_chunk(tag, data):
    return (struct.pack(">I", len(data)) + tag + data +
            struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF))


def _write_png(path, w, h, rgba):
    w = max(1, int(w))
    h = max(1, int(h))
    c = [int(max(0, min(255, round(float(v) * 255)))) for v in (list(rgba) + [1.0])[:4]]
    pixel = bytes(c)
    row = pixel * w
    raw = b"".join(b"\x00" + row for _ in range(h))
    png = (b"\x89PNG\r\n\x1a\n"
           + _png_chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
           + _png_chunk(b"IDAT", zlib.compress(raw, 6))
           + _png_chunk(b"IEND", b""))
    d = os.path.dirname(os.path.abspath(path))
    if d:
        os.makedirs(d, exist_ok=True)
    with open(path, "wb") as fh:
        fh.write(png)


# ---------------------------------------------------------------------------
# Data blocks
# ---------------------------------------------------------------------------
class _Base(object):
    def __init__(self, name=""):
        self.name = name
        self.users = 0


class Poly(object):
    def __init__(self, n=3):
        self.vertices = list(range(n))


class UVLayer(_Base):
    pass


class UVLayerCollection(object):
    def __init__(self, layers):
        self.layers = layers

    def new(self, name="UVMap", **_kw):
        layer = UVLayer(name)
        self.layers.append(layer)
        return layer

    def __iter__(self):
        return iter(self.layers)

    def __len__(self):
        return len(self.layers)

    def __bool__(self):
        return len(self.layers) > 0


class Mesh(_Base):
    def __init__(self, name="Mesh"):
        _Base.__init__(self, name)
        self._kind = "MESH"
        self.polygons = [Poly(3) for _ in range(12)]
        self.uv_layers = UVLayerCollection([UVLayer("UVMap")])
        self.materials = []          # plain list: append / index / len / truthiness


class Bone(_Base):
    def __init__(self, name):
        _Base.__init__(self, name)
        self.head = (0.0, 0.0, 0.0)
        self.head_local = (0.0, 0.0, 0.0)
        self.tail = (0.0, 0.0, 0.0)
        self.parent = None
        self.use_connect = False


class BoneCollection(object):
    """edit_bones: supports .new(name) and [name] lookup."""

    def __init__(self, bones):
        self.bones = bones

    def new(self, name):
        b = Bone(name)
        self.bones.append(b)
        return b

    def __getitem__(self, key):
        if isinstance(key, int):
            return self.bones[key]
        for b in self.bones:
            if b.name == key:
                return b
        raise KeyError(key)

    def __iter__(self):
        return iter(self.bones)

    def __len__(self):
        return len(self.bones)


class Armature(_Base):
    def __init__(self, name="Armature"):
        _Base.__init__(self, name)
        self._kind = "ARMATURE"
        self.bones = []
        self.edit_bones = BoneCollection(self.bones)


class Light(_Base):
    def __init__(self, name="Light", kind="AREA"):
        _Base.__init__(self, name)
        self.type = kind
        self.energy = 0.0
        self.color = (1.0, 1.0, 1.0)
        self._kind = "LIGHT"


class Camera(_Base):
    def __init__(self, name="Camera"):
        _Base.__init__(self, name)
        self.type = "PERSP"
        self.ortho_scale = 1.0
        self._kind = "CAMERA"


class Socket(object):
    def __init__(self):
        self.default_value = 0.0


class SocketCollection(object):
    def __getitem__(self, _key):
        return Socket()


class Node(object):
    def __init__(self, ntype=""):
        self.type = ntype
        self.location = (0, 0)
        self.label = ""
        self.image = None
        self.blend_type = ""
        self.outputs = SocketCollection()
        self.inputs = SocketCollection()


class NodeCollection(object):
    def __init__(self):
        self.nodes = []

    def clear(self):
        self.nodes = []

    def new(self, ntype):
        n = Node(ntype)
        self.nodes.append(n)
        return n


class Links(object):
    def new(self, _a, _b):
        return None


class NodeTree(object):
    def __init__(self):
        self.nodes = NodeCollection()
        self.links = Links()


class Material(_Base):
    def __init__(self, name="Material"):
        _Base.__init__(self, name)
        self.use_nodes = False
        self.node_tree = NodeTree()


class PixelBuffer(object):
    def __init__(self):
        self._color = (0.5, 0.5, 0.5, 1.0)

    def foreach_set(self, seq):
        vals = list(seq)
        if len(vals) >= 4:
            self._color = tuple(vals[:4])


class ColorSpace(object):
    def __init__(self):
        self.name = "sRGB"


class Image(_Base):
    def __init__(self, name="Image", width=64, height=64):
        _Base.__init__(self, name)
        self.width = int(width)
        self.height = int(height)
        self.pixels = PixelBuffer()
        self.filepath_raw = ""
        self.file_format = "PNG"
        self.colorspace_settings = ColorSpace()

    def save(self):
        if self.filepath_raw:
            _write_png(self.filepath_raw, self.width, self.height, self.pixels._color)


class Action(_Base):
    pass


class Modifier(object):
    def __init__(self, name="", mtype=""):
        self.name = name
        self.type = mtype
        self.levels = 0
        self.render_levels = 0
        self.ratio = 1.0
        self.object = None


class ModifierCollection(object):
    def __init__(self):
        self.mods = []

    def new(self, name="", type=""):
        m = Modifier(name, type)
        self.mods.append(m)
        return m

    def __iter__(self):
        return iter(self.mods)

    def __len__(self):
        return len(self.mods)


class VertexGroup(object):
    def __init__(self, name=""):
        self.name = name


class VertexGroupCollection(object):
    def __init__(self):
        self.groups = []

    def new(self, name=""):
        g = VertexGroup(name)
        self.groups.append(g)
        return g

    def __iter__(self):
        return iter(self.groups)

    def __len__(self):
        return len(self.groups)

    def __bool__(self):
        return len(self.groups) > 0


class Strip(object):
    def __init__(self, name="", start=0, action=None):
        self.name = name
        self.start = start
        self.action = action


class StripCollection(object):
    def __init__(self):
        self.strips = []

    def new(self, name="", start=0, action=None):
        s = Strip(name, start, action)
        self.strips.append(s)
        return s


class Track(object):
    def __init__(self):
        self.name = ""
        self.strips = StripCollection()


class TrackCollection(object):
    def __init__(self):
        self.tracks = []

    def new(self):
        t = Track()
        self.tracks.append(t)
        return t


class AnimationData(object):
    def __init__(self):
        self.action = None
        self.nla_tracks = TrackCollection()


class PoseBone(object):
    def __init__(self, name):
        self.name = name

    def keyframe_insert(self, data_path="", frame=0, group=""):
        return True


class Pose(object):
    def __init__(self, armature_data):
        self._data = armature_data

    @property
    def bones(self):
        return [PoseBone(b.name) for b in getattr(self._data, "bones", [])]


class Object(_Base):
    _BOUND_BOX = [(-1, -1, -1), (-1, -1, 1), (-1, 1, -1), (-1, 1, 1),
                  (1, -1, -1), (1, -1, 1), (1, 1, -1), (1, 1, 1)]

    def __init__(self, name="Object", data=None):
        _Base.__init__(self, name)
        self.data = data
        self.type = getattr(data, "_kind", "MESH")
        if data is not None and not hasattr(data, "_kind"):
            data._kind = self.type
        self.matrix_world = Matrix()
        self.bound_box = list(self._BOUND_BOX)
        self.location = Vector((0, 0, 0))
        self.rotation_euler = Vector((0, 0, 0))
        self.modifiers = ModifierCollection()
        self.vertex_groups = VertexGroupCollection()
        self.parent = None
        self.show_in_front = False
        self.animation_data = None

    def select_set(self, _val):
        return None

    def animation_data_create(self):
        if self.animation_data is None:
            self.animation_data = AnimationData()
        return self.animation_data

    @property
    def pose(self):
        return Pose(self.data)

    def evaluated_get(self, _dg):
        return self

    def to_mesh(self):
        return self.data

    def to_mesh_clear(self):
        return None


# ---------------------------------------------------------------------------
# bpy.data collections
# ---------------------------------------------------------------------------
class Collection(object):
    def __init__(self, key):
        self.key = key

    def _list(self):
        return _state[self.key]

    def __iter__(self):
        return iter(self._list())

    def __len__(self):
        return len(self._list())

    def __bool__(self):
        return len(self._list()) > 0

    def __getitem__(self, idx):
        return self._list()[idx]

    def get(self, name, default=None):
        for item in self._list():
            if getattr(item, "name", None) == name:
                return item
        return default

    def remove(self, item, do_unlink=True, **_kw):
        try:
            self._list().remove(item)
        except ValueError:
            pass

    def load(self, path, check_existing=True):
        name = os.path.splitext(os.path.basename(path))[0]
        if check_existing:
            for img in self._list():
                if getattr(img, "name", None) == name:
                    return img
        img = Image(name=name, width=64, height=64)
        img.filepath_raw = os.path.abspath(path)
        self._list().append(img)
        return img

    def new(self, *args, **kwargs):
        name = args[0] if args else kwargs.get("name", "")
        key = self.key
        if key == "objects":
            data = args[1] if len(args) > 1 else kwargs.get("object_data")
            obj = Object(name, data)
            self._list().append(obj)
            return obj
        if key == "meshes":
            item = Mesh(name)
        elif key == "armatures":
            item = Armature(name)
        elif key == "lights":
            item = Light(name, kwargs.get("type", "AREA"))
        elif key == "cameras":
            item = Camera(name)
        elif key == "materials":
            item = Material(name)
        elif key == "images":
            item = Image(name, kwargs.get("width", 64), kwargs.get("height", 64))
        elif key == "actions":
            item = Action(name)
        else:
            item = _Base(name)
        self._list().append(item)
        return item


class Data(object):
    def __init__(self):
        self.objects = Collection("objects")
        self.meshes = Collection("meshes")
        self.armatures = Collection("armatures")
        self.materials = Collection("materials")
        self.images = Collection("images")
        self.curves = Collection("curves")
        self.cameras = Collection("cameras")
        self.lights = Collection("lights")
        self.actions = Collection("actions")


# ---------------------------------------------------------------------------
# bpy.context / bpy.ops
# ---------------------------------------------------------------------------
class RenderSettings(object):
    """Blender 5.x removed BLENDER_EEVEE_NEXT. Dynamic enums raise TypeError
    on an invalid assignment, so the mock enforces the same rule; a stage
    script that regresses to the 4.x identifier fails its tests."""

    ENGINES = ("BLENDER_EEVEE", "BLENDER_WORKBENCH", "CYCLES")

    def __init__(self):
        self._engine = "BLENDER_EEVEE"
        self.resolution_x = 512
        self.resolution_y = 512
        self.resolution_percentage = 100
        self.image_settings = types.SimpleNamespace(file_format="PNG")
        self.film_transparent = True
        self.filepath = ""
        self.bake = types.SimpleNamespace(margin=4)

    @property
    def engine(self):
        return self._engine

    @engine.setter
    def engine(self, value):
        if value not in self.ENGINES:
            raise TypeError(
                "enum %r not found in %r (Blender 5.x removed "
                "BLENDER_EEVEE_NEXT)" % (value, self.ENGINES))
        self._engine = value


class Scene(object):
    def __init__(self):
        self.camera = None
        self.render = RenderSettings()
        self.cycles = types.SimpleNamespace(bake_type="AO")


class _LinkObjects(object):
    def __init__(self):
        self.active = None

    def link(self, obj):
        if obj not in _state["objects"]:
            _state["objects"].append(obj)


class ViewLayer(object):
    def __init__(self):
        self.objects = _LinkObjects()


class Context(object):
    def __init__(self, scene):
        self.scene = scene
        self.view_layer = ViewLayer()
        self.collection = types.SimpleNamespace(objects=_LinkObjects())
        self.active_object = None

    def evaluated_depsgraph_get(self):
        return object()


def _add_mesh_object(name="ImportedMesh"):
    mesh = Mesh(name + "_mesh")
    _state["meshes"].append(mesh)
    obj = Object(name, mesh)
    _state["objects"].append(obj)
    return obj


class _NoOpOps(object):
    def _new_primitive(self, name):
        # Real Blender creates a mesh object AND makes it the active object;
        # stage_01_modeling.py reads bpy.context.active_object right after.
        obj = _add_mesh_object(name)
        _context.active_object = obj
        _context.view_layer.objects.active = obj
        return {"FINISHED"}

    def primitive_cube_add(self, size=2.0, **_kw):
        return self._new_primitive("Cube")

    def primitive_uv_sphere_add(self, radius=1.0, segments=32, ring_count=16, **_kw):
        return self._new_primitive("Sphere")

    def primitive_cylinder_add(self, radius=1.0, depth=2.0, vertices=32, **_kw):
        return self._new_primitive("Cylinder")

    def primitive_plane_add(self, size=2.0, **_kw):
        return self._new_primitive("Plane")

    def select_all(self, action="SELECT", **_kw):
        return {"FINISHED"}

    def delete(self, use_global=False, **_kw):
        _state["objects"].clear()

    def mode_set(self, mode="OBJECT", **_kw):
        return {"FINISHED"}

    def modifier_apply(self, modifier="", **_kw):
        return {"FINISHED"}

    def origin_set(self, type="ORIGIN_GEOMETRY", center="BOUNDS", **_kw):
        return {"FINISHED"}

    def transform_apply(self, location=False, rotation=False, scale=True, **_kw):
        return {"FINISHED"}

    def bake(self, type="AO", **_kw):
        return {"FINISHED"}

    def smart_project(self, island_margin=0.02, **_kw):
        return {"FINISHED"}

    def seams_from_islands(self, **_kw):
        return {"FINISHED"}


class _WMOps(object):
    def open_mainfile(self, filepath="", **_kw):
        if not _load_snapshot(filepath):
            _add_mesh_object()
        return {"FINISHED"}

    def save_as_mainfile(self, filepath="", **_kw):
        _write_snapshot(filepath)
        return {"FINISHED"}

    def obj_import(self, filepath="", **_kw):
        _do_import(filepath)
        return {"FINISHED"}


class _ImportOps(object):
    def gltf(self, filepath="", **_kw):
        _do_import(filepath)
        return {"FINISHED"}

    def fbx(self, filepath="", **_kw):
        _do_import(filepath)
        return {"FINISHED"}


class _ExportOps(object):
    def fbx(self, **kwargs):
        path = kwargs.get("filepath", "")
        if path:
            _write_snapshot(path, fbx=True)
        # Record the export parameters so stage-level tests can assert the
        # SPEC.md §9 seven-parameter Unity contract.
        _state.setdefault("exportLog", []).append(
            {k: repr(v) for k, v in sorted(kwargs.items())})
        return {"FINISHED"}


class _RenderOps(object):
    def render(self, write_still=True, **_kw):
        path = _scene.render.filepath
        if path:
            _write_png(path, _scene.render.resolution_x,
                       _scene.render.resolution_y, (0.12, 0.12, 0.14, 1.0))
        return {"FINISHED"}


def _do_import(path):
    if not _load_snapshot(path):
        _add_mesh_object()


class Ops(object):
    def __init__(self):
        noop = _NoOpOps()
        self.object = noop
        self.mesh = noop
        self.uv = noop
        self.wm = _WMOps()
        self.import_scene = _ImportOps()
        self.export_scene = _ExportOps()
        self.render = _RenderOps()


_scene = Scene()
_context = Context(_scene)
_data = Data()


class Bpy(object):
    """The object placed in sys.modules['bpy']."""

    def __init__(self):
        self.data = _data
        self.context = _context
        self.ops = Ops()
        # env_report-style metadata (see skills/blender-core/SKILL.md).
        self.app = types.SimpleNamespace(
            version=(5, 2, 0), version_string="5.2.0",
            background=True, binary_path="blender")


def install():
    """Install the mock modules. Call BEFORE any stage script is imported."""
    sys.modules["bpy"] = Bpy()
    mu = types.ModuleType("mathutils")
    mu.Vector = Vector
    mu.Matrix = Matrix
    sys.modules["mathutils"] = mu
    return sys.modules["bpy"]
