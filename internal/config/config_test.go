package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAndValidate(t *testing.T) {
	c := Default("hero-knight")
	if err := c.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	if c.Strategy != "tripo" {
		t.Fatalf("default strategy = %q, want tripo (default production supplier)", c.Strategy)
	}
	if c.Handoff.Mode != "manual" {
		t.Fatalf("default handoff = %q, want manual", c.Handoff.Mode)
	}
	if c.BlenderVersion != "5.2" || c.TripoURL != "https://studio.tripo3d.com/" {
		t.Fatalf("version/url drift: %+v", c)
	}
}

func TestValidateRejectsBad(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"bad id", func(c *Config) { c.ID = "Hero_Knight" }},
		{"bad strategy", func(c *Config) { c.Strategy = "photogrammetry" }},
		{"wrong blender", func(c *Config) { c.BlenderVersion = "4.2" }},
		{"wrong tripo url", func(c *Config) { c.TripoURL = "https://tripo3d.com/" }},
		{"wrong axes", func(c *Config) { c.Axes.Up = "Z" }},
		{"unknown stage", func(c *Config) { c.Stages = []string{"modeling", "magick"} }},
		{"bad handoff mode", func(c *Config) { c.Handoff.Mode = "teleport" }},
		{"file handoff no source", func(c *Config) { c.Handoff.Mode = "file"; c.Handoff.Source = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Default("hero-knight")
			tc.mut(c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "assets", "hero-knight", "asset.json")
	c := Default("hero-knight")
	c.Humanoid = true
	c.Handoff = Handoff{Mode: "file", Source: "drops/hero.glb"}
	if err := c.Save(p); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "hero-knight" || !loaded.Humanoid || loaded.Strategy != "tripo" {
		t.Fatalf("round trip mismatch: %+v", loaded)
	}
	if loaded.Handoff.Mode != "file" || loaded.Handoff.Source != "drops/hero.glb" {
		t.Fatalf("handoff round trip mismatch: %+v", loaded.Handoff)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
}

func TestEffectiveHandoffEnvOverride(t *testing.T) {
	c := Default("hero-knight")
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "fixture")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "upstream/model.glb")
	h := c.EffectiveHandoff()
	if h.Mode != "fixture" || h.Source != "upstream/model.glb" {
		t.Fatalf("env override not applied: %+v", h)
	}
	// config value unchanged (env is resolved at call time, not stored).
	if c.Handoff.Mode != "manual" {
		t.Fatalf("EffectiveHandoff mutated config: %+v", c.Handoff)
	}
}

func TestStagesOrder(t *testing.T) {
	want := []string{"modeling", "topology-uv", "bake", "pbr", "unity-export", "render"}
	if len(Stages) != len(want) {
		t.Fatalf("stage count = %d", len(Stages))
	}
	for i := range want {
		if Stages[i] != want[i] {
			t.Fatalf("stage %d = %q, want %q", i, Stages[i], want[i])
		}
	}
}

// TestValidateRejectsEffectiveHandoffDrift: an env override must be validated
// with the same rules as the persisted value (previous bug: only c.Handoff.Mode
// was checked, so an invalid ASSETCTL_TRIPO_HANDOFF slipped into production).
func TestValidateRejectsEffectiveHandoffDrift(t *testing.T) {
	c := Default("hero-knight")
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "teleport")
	if err := c.Validate(); err == nil {
		t.Fatal("invalid ASSETCTL_TRIPO_HANDOFF must be rejected")
	}
	t.Setenv("ASSETCTL_TRIPO_HANDOFF", "file")
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "")
	if err := c.Validate(); err == nil {
		t.Fatal("env-forced file handoff without source must be rejected")
	}
	t.Setenv("ASSETCTL_TRIPO_SOURCE", "drops/model.glb")
	if err := c.Validate(); err != nil {
		t.Fatalf("valid env-forced file handoff rejected: %v", err)
	}
}

func TestValidateStrictSchema(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"partial stages", func(c *Config) { c.Stages = []string{"modeling", "render"} }},
		{"reordered stages", func(c *Config) { c.Stages[0], c.Stages[1] = c.Stages[1], c.Stages[0] }},
		{"duplicate stages", func(c *Config) { c.Stages[5] = "modeling" }},
		{"bad kind", func(c *Config) { c.Kind = "vehicle" }},
		{"bad engine", func(c *Config) { c.TargetEngine = "unreal" }},
		{"bad texture channel", func(c *Config) { c.Textures = []string{"basecolor", "emissive"} }},
		{"zero budget", func(c *Config) { c.Budget.MaxTriangles = 0 }},
		{"import without source", func(c *Config) { c.Strategy = "import" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Default("hero-knight")
			c.Handoff.Source = "" // keep env-independent
			tc.mut(c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestKnownStrategiesMatchesRegistryContract(t *testing.T) {
	// config cannot import strategy (cycle), so the invariant is asserted in
	// both packages: this list must equal the registered supplier IDs.
	if len(KnownStrategies) != 3 || KnownStrategies[0] != "tripo" {
		t.Fatalf("KnownStrategies drift: %v", KnownStrategies)
	}
	if DefaultSupplier != "tripo" {
		t.Fatalf("default supplier drift: %q", DefaultSupplier)
	}
}
