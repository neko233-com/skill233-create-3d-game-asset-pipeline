// Package config loads and validates asset.json pipeline configs.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// BlenderVersion is the ONLY supported Blender target (AGENTS.md / SPEC.md
	// §2). Do not widen this to a range: every bpy module in scripts/blender
	// is written against 5.2's API surface, and asset.json validation rejects
	// anything else.
	BlenderVersion = "5.2"
	TripoURL       = "https://studio.tripo3d.com/"
	DefaultGo      = "1.27"

	// DefaultSupplier is the production modeling supplier. Tripo is a browser
	// based, human-in-the-loop supplier; see the Handoff mechanism (SPEC.md §3).
	DefaultSupplier = "tripo"
)

var assetIDRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Stages is the fixed game-art pipeline order (modeling → topology/UV → bake →
// PBR → Unity export/sync → render composite). See SPEC.md §4.
var Stages = []string{
	"modeling",
	"topology-uv",
	"bake",
	"pbr",
	"unity-export",
	"render",
}

// KnownStrategies lists the built-in modeling suppliers/strategies. The
// Supplier == Strategy equivalence is documented in SPEC.md §3; "tripo" is the
// default production supplier, "blender" (procedural) and "import" are
// alternatives, and the registry is the extension point for future suppliers.
var KnownStrategies = []string{"tripo", "blender", "import"}

// HandoffModes enumerates how the (default, semi-automatic) Tripo supplier is
// fed its export. "manual" is production default (human downloads a GLB),
// "file" copies an upstream drop, "fixture" generates a deterministic GLB for
// unattended CI. See SPEC.md §3 and docs/10-strategies.md.
var HandoffModes = []string{"manual", "file", "fixture"}

type Axes struct {
	Forward string  `json:"forward"`
	Up      string  `json:"up"`
	Scale   float64 `json:"scale"`
}

type Budget struct {
	MaxTriangles   int `json:"maxTriangles"`
	MaxTextureSize int `json:"maxTextureSize"`
}

type Modeling struct {
	Primitive    string  `json:"primitive"`
	Size         float64 `json:"size"`
	Subdivisions int     `json:"subdivisions"`
}

// Handoff describes how the default supplier's model reaches raw/<id>.<ext>.
// Env overrides: ASSETCTL_TRIPO_HANDOFF (Mode), ASSETCTL_TRIPO_SOURCE (Source).
type Handoff struct {
	Mode   string `json:"mode"`
	Source string `json:"source"`
}

// Config is the asset.json schema (SPEC.md §7).
type Config struct {
	ID             string   `json:"id"`
	Strategy       string   `json:"strategy"`
	Kind           string   `json:"kind"`
	Humanoid       bool     `json:"humanoid"`
	ReferenceImage string   `json:"referenceImage"`
	Source         string   `json:"source"`
	TripoURL       string   `json:"tripoUrl"`
	BlenderVersion string   `json:"blenderVersion"`
	TargetEngine   string   `json:"targetEngine"`
	Axes           Axes     `json:"axes"`
	Budget         Budget   `json:"budget"`
	Textures       []string `json:"textures"`
	Animations     []string `json:"animations"`
	Modeling       Modeling `json:"modeling"`
	Stages         []string `json:"stages"`
	Handoff        Handoff  `json:"handoff"`
}

// Default returns a config pre-filled with production defaults: Tripo is the
// default supplier and the handoff is manual (human download step).
func Default(id string) *Config {
	return &Config{
		ID:             id,
		Strategy:       DefaultSupplier,
		Kind:           "prop",
		Humanoid:       false,
		ReferenceImage: "raw/" + id + ".png",
		TripoURL:       TripoURL,
		BlenderVersion: BlenderVersion,
		TargetEngine:   "unity-urp",
		Axes:           Axes{Forward: "-Z", Up: "Y", Scale: 1.0},
		Budget:         Budget{MaxTriangles: 30000, MaxTextureSize: 2048},
		Textures:       []string{"basecolor", "normal", "roughness", "metallic"},
		Animations:     []string{},
		Modeling:       Modeling{Primitive: "cube", Size: 1.0, Subdivisions: 2},
		Stages:         append([]string(nil), Stages...),
		Handoff:        Handoff{Mode: "manual", Source: ""},
	}
}

// Load reads a JSON config from disk and fills zero values with defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) applyDefaults() {
	if c.Strategy == "" {
		c.Strategy = DefaultSupplier
	}
	if c.Kind == "" {
		c.Kind = "prop"
	}
	if c.TripoURL == "" {
		c.TripoURL = TripoURL
	}
	if c.BlenderVersion == "" {
		c.BlenderVersion = BlenderVersion
	}
	if c.TargetEngine == "" {
		c.TargetEngine = "unity-urp"
	}
	if c.Axes.Forward == "" {
		c.Axes.Forward = "-Z"
	}
	if c.Axes.Up == "" {
		c.Axes.Up = "Y"
	}
	if c.Axes.Scale == 0 {
		c.Axes.Scale = 1.0
	}
	if c.Budget.MaxTriangles == 0 {
		c.Budget.MaxTriangles = 30000
	}
	if c.Budget.MaxTextureSize == 0 {
		c.Budget.MaxTextureSize = 2048
	}
	if len(c.Textures) == 0 {
		c.Textures = []string{"basecolor", "normal", "roughness", "metallic"}
	}
	if c.Modeling.Primitive == "" {
		c.Modeling.Primitive = "cube"
	}
	if c.Modeling.Size == 0 {
		c.Modeling.Size = 1.0
	}
	if len(c.Stages) == 0 {
		c.Stages = append([]string(nil), Stages...)
	}
	if c.Handoff.Mode == "" {
		c.Handoff.Mode = "manual"
	}
}

// EffectiveHandoff resolves the handoff mode/source, applying env overrides.
// ASSETCTL_TRIPO_HANDOFF overrides Mode, ASSETCTL_TRIPO_SOURCE overrides Source.
// An unknown env mode is returned as-is so Validate can reject it.
func (c *Config) EffectiveHandoff() Handoff {
	h := c.Handoff
	if v := os.Getenv("ASSETCTL_TRIPO_HANDOFF"); v != "" {
		h.Mode = v
	}
	if v := os.Getenv("ASSETCTL_TRIPO_SOURCE"); v != "" {
		h.Source = v
	}
	return h
}

// ValidTextures is the exact PBR channel suffix set (SPEC.md §11), shared
// with scripts/blender/common.py PBR_SUFFIXES.
var ValidTextures = []string{"basecolor", "normal", "roughness", "metallic", "ao"}

// Validate checks the config against SPEC.md invariants. It validates the
// EFFECTIVE handoff (env overrides included), because that is what the Tripo
// supplier actually consumes; validating only the persisted value lets an
// invalid ASSETCTL_TRIPO_HANDOFF through into production.
func (c *Config) Validate() error {
	if !assetIDRe.MatchString(c.ID) {
		return fmt.Errorf("invalid asset id %q: want lowercase-kebab (e.g. hero-knight)", c.ID)
	}
	if !contains(KnownStrategies, c.Strategy) {
		return fmt.Errorf("unknown strategy %q (known: %s)", c.Strategy, strings.Join(KnownStrategies, ", "))
	}
	if c.Kind != "prop" && c.Kind != "character" {
		return fmt.Errorf("kind must be prop|character, got %q", c.Kind)
	}
	if c.TargetEngine != "unity-urp" {
		return fmt.Errorf("targetEngine must be unity-urp, got %q", c.TargetEngine)
	}
	if c.BlenderVersion != BlenderVersion {
		return fmt.Errorf("blenderVersion must be %s, got %q", BlenderVersion, c.BlenderVersion)
	}
	if c.TripoURL != TripoURL {
		return fmt.Errorf("tripoUrl must be %s, got %q", TripoURL, c.TripoURL)
	}
	if c.Axes.Forward != "-Z" || c.Axes.Up != "Y" || c.Axes.Scale != 1.0 {
		return fmt.Errorf("axes must be -Z forward / Y up / scale 1.0 for Unity (SPEC.md §9)")
	}
	if c.Budget.MaxTriangles <= 0 || c.Budget.MaxTextureSize <= 0 {
		return fmt.Errorf("budget must be positive, got %+v", c.Budget)
	}
	for _, tex := range c.Textures {
		if !contains(ValidTextures, tex) {
			return fmt.Errorf("unknown texture channel %q (known: %s)", tex, strings.Join(ValidTextures, ", "))
		}
	}
	h := c.EffectiveHandoff()
	if !contains(HandoffModes, h.Mode) {
		return fmt.Errorf("unknown handoff mode %q (config=%q; env ASSETCTL_TRIPO_HANDOFF may override; known: %s)",
			h.Mode, c.Handoff.Mode, strings.Join(HandoffModes, ", "))
	}
	// A "file" handoff needs a source: handoff.source (or the env override).
	if h.Mode == "file" && strings.TrimSpace(h.Source) == "" {
		return fmt.Errorf("handoff mode %q requires handoff.source (or ASSETCTL_TRIPO_SOURCE)", h.Mode)
	}
	if c.Strategy == "import" && strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("strategy %q requires config.source", c.Strategy)
	}
	// The six stages are a FIXED, ordered contract (SPEC.md §4). A partial or
	// reordered list silently breaks --from / --only resume and the DoD gate.
	if len(c.Stages) != len(Stages) {
		return fmt.Errorf("config.stages must contain exactly %d stages, got %d", len(Stages), len(c.Stages))
	}
	for i, want := range Stages {
		if c.Stages[i] != want {
			return fmt.Errorf("config.stages[%d] = %q, want %q (fixed order)", i, c.Stages[i], want)
		}
	}
	return nil
}

// Save writes the config as indented JSON.
func (c *Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ErrNotFound is returned for missing asset directories.
var ErrNotFound = errors.New("asset config not found")
