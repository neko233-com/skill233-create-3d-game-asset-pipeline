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
	BlenderVersion = "5.2"
	TripoURL       = "https://studio.tripo3d.com/"
	DefaultGo      = "1.27"
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

// KnownStrategies lists the built-in modeling strategies. "blender" is the default.
var KnownStrategies = []string{"blender", "tripo", "import"}

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

// Config is the asset.json schema (SPEC.md §7).
type Config struct {
	ID              string   `json:"id"`
	Strategy        string   `json:"strategy"`
	Kind            string   `json:"kind"`
	Humanoid        bool     `json:"humanoid"`
	ReferenceImage  string   `json:"referenceImage"`
	Source          string   `json:"source"`
	TripoURL        string   `json:"tripoUrl"`
	BlenderVersion  string   `json:"blenderVersion"`
	TargetEngine    string   `json:"targetEngine"`
	Axes            Axes     `json:"axes"`
	Budget          Budget   `json:"budget"`
	Textures        []string `json:"textures"`
	Animations      []string `json:"animations"`
	Modeling        Modeling `json:"modeling"`
	Stages          []string `json:"stages"`
}

// Default returns a config pre-filled with pipeline defaults.
func Default(id string) *Config {
	return &Config{
		ID:             id,
		Strategy:       "blender",
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
		c.Strategy = "blender"
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
}

// Validate checks the config against SPEC.md invariants.
func (c *Config) Validate() error {
	if !assetIDRe.MatchString(c.ID) {
		return fmt.Errorf("invalid asset id %q: want lowercase-kebab (e.g. hero-knight)", c.ID)
	}
	if !contains(KnownStrategies, c.Strategy) {
		return fmt.Errorf("unknown strategy %q (known: %s)", c.Strategy, strings.Join(KnownStrategies, ", "))
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
	for _, s := range c.Stages {
		if !contains(Stages, s) {
			return fmt.Errorf("unknown stage %q in config (known: %s)", s, strings.Join(Stages, ", "))
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
