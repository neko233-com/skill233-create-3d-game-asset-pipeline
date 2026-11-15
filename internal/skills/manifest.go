package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

// Manifest mirrors skills/manifest.json (documentation index, MIT source).
type Manifest struct {
	Source        string          `json:"source"`
	License       string          `json:"license"`
	BlenderTarget string          `json:"blenderTarget"`
	Skills        []ManifestSkill `json:"skills"`
}

type ManifestSkill struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	ReferenceScripts []string `json:"referenceScripts"`
	GoCommand        string   `json:"goCommand"`
	BpyScript        string   `json:"bpyScript"`
}

// LoadManifest reads the documentation index from disk.
func LoadManifest(root string) (*Manifest, error) {
	p := filepath.Join(root, "skills", "manifest.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read skills/manifest.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse skills/manifest.json: %w", err)
	}
	return &m, nil
}

// ValidateCatalog proves that the Go registry, the copied SKILL.md corpus,
// the manifest.json index and the single bpy dispatcher all agree. Called by
// `assetctl skills doctor`, `assetctl verify` and Go unit tests.
func ValidateCatalog(root string) error {
	m, err := LoadManifest(root)
	if err != nil {
		return err
	}
	if m.License != "MIT" {
		return fmt.Errorf("skills license = %q, want MIT", m.License)
	}
	if m.BlenderTarget != config.BlenderVersion {
		return fmt.Errorf("skills blenderTarget = %q, want %s (AGENTS.md)", m.BlenderTarget, config.BlenderVersion)
	}
	if !strings.Contains(m.Source, "EL4CTEO/blender-skills") {
		return fmt.Errorf("skills source must credit EL4CTEO/blender-skills, got %q", m.Source)
	}
	if len(m.Skills) != Count() {
		return fmt.Errorf("manifest has %d skills, Go catalog has %d", len(m.Skills), Count())
	}

	manifestIDs := make([]string, 0, len(m.Skills))
	for _, ms := range m.Skills {
		manifestIDs = append(manifestIDs, ms.ID)
		s, ok := ByID(ms.ID)
		if !ok {
			return fmt.Errorf("manifest skill %q is not registered in Go catalog", ms.ID)
		}
		wantGo := "assetctl skills run " + s.ID
		if ms.GoCommand != wantGo {
			return fmt.Errorf("manifest %s goCommand = %q, want %q", s.ID, ms.GoCommand, wantGo)
		}
		wantBpy := "scripts/blender/skill_run.py --skill " + s.ID
		if ms.BpyScript != wantBpy {
			return fmt.Errorf("manifest %s bpyScript = %q, want %q", s.ID, ms.BpyScript, wantBpy)
		}
		for _, rel := range []string{"SKILL.md", "ADAPTATION.md"} {
			p := filepath.Join(root, "skills", s.ID, rel)
			st, err := os.Stat(p)
			if err != nil || st.IsDir() || st.Size() == 0 {
				return fmt.Errorf("missing/empty skills/%s/%s", s.ID, rel)
			}
		}
	}
	sort.Strings(manifestIDs)
	goIDs := IDs()
	for i := range goIDs {
		if goIDs[i] != manifestIDs[i] {
			return fmt.Errorf("catalog/manifest id drift at %d: %q vs %q", i, goIDs[i], manifestIDs[i])
		}
	}

	dispatcher := filepath.Join(root, "scripts", "blender", "skill_run.py")
	src, err := os.ReadFile(dispatcher)
	if err != nil {
		return fmt.Errorf("missing %s: %w", dispatcher, err)
	}
	for _, s := range Catalog {
		needle := "\"" + s.Operation + "\""
		if !strings.Contains(string(src), needle) {
			return fmt.Errorf("skill_run.py has no operation %s for %s", needle, s.ID)
		}
	}
	return nil
}
