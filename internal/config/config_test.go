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
	if c.Strategy != "blender" {
		t.Fatalf("default strategy = %q, want blender", c.Strategy)
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
	c.Strategy = "tripo"
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
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
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
