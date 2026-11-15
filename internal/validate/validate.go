// Package validate checks a built asset directory before Unity import (SPEC.md §11).
package validate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

var assetIDRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var pbrHints = []string{"basecolor", "albedo", "normal", "roughness", "metallic", "ao"}

// Options for Validate.
type Options struct {
	ExpectHumanoid  bool
	ExpectTextures  bool
	ExpectManifest  bool // require build/<id>/manifest.json with six "ok" stages (DoD, SPEC.md §3A.3)
	ExpectContracts bool // require six stage contracts with matching SHA-256 digests (SPEC.md §5A)
}

// Result is the validation report.
type Result struct {
	Dir      string
	Models   []string
	Textures []string
	Issues   []string
}

func (r *Result) OK() bool { return len(r.Issues) == 0 }

// Validate inspects dir against SPEC.md §6/§11 conventions.
func Validate(dir string, opt Options) (*Result, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}
	r := &Result{Dir: dir}
	id := filepath.Base(filepath.Clean(dir))
	if !assetIDRe.MatchString(id) {
		r.Issues = append(r.Issues, fmt.Sprintf("asset dir name %q is not lowercase-kebab", id))
	}

	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		switch ext {
		case ".fbx", ".glb", ".gltf", ".obj":
			r.Models = append(r.Models, p)
		case ".png", ".jpg", ".jpeg", ".tga":
			r.Textures = append(r.Textures, p)
			stem := strings.ToLower(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
			// preview.png is the stage-6 QA render, NOT a PBR texture
			// (docs/43). Exempt its stem so a fully-built deliverable passes.
			if stem != "preview" && !anyHint(stem, pbrHints) {
				r.Issues = append(r.Issues, fmt.Sprintf("texture %s lacks PBR suffix (_basecolor/_normal/_roughness/_metallic/_ao)", filepath.Base(p)))
			}
		}
		// A silently-truncated deliverable must never pass DoD. Blender / Unity
		// both fail in confusing ways on zero-byte or corrupt files.
		if issue := checkArtifactIntegrity(p, ext); issue != "" {
			r.Issues = append(r.Issues, issue)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(r.Models) == 0 {
		r.Issues = append(r.Issues, "no model file (.fbx/.glb/.gltf/.obj)")
	}
	if opt.ExpectTextures && len(r.Textures) == 0 {
		r.Issues = append(r.Issues, "--expect-textures set but no texture files found")
	}
	if opt.ExpectHumanoid {
		marker := filepath.Join(dir, ".humanoid")
		if _, err := os.Stat(marker); err != nil {
			named := false
			for _, m := range r.Models {
				base := strings.ToLower(filepath.Base(m))
				if strings.Contains(base, "humanoid") || strings.Contains(base, "rig") {
					named = true
				}
			}
			if !named && !strings.HasPrefix(id, "hero") && !strings.Contains(id, "character") {
				r.Issues = append(r.Issues, "--expect-humanoid set but no rig marker (add empty .humanoid file)")
			}
		}
	}
	// DoD gate: when a manifest exists, all six stages MUST be "ok" regardless
	// of ExpectManifest; a missing manifest only fails when explicitly required.
	r.Issues = append(r.Issues, checkManifest(dir, opt.ExpectManifest)...)
	r.Issues = append(r.Issues, checkContracts(dir, opt.ExpectContracts)...)
	return r, nil
}

// checkManifest verifies build/<id>/manifest.json reports every one of the six
// pipeline stages (config.Stages) as "ok". Two encodings are accepted: a plain
// string ("ok") or an object with a status field ({"status":"ok"}).
func checkManifest(dir string, require bool) []string {
	p := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(p)
	if err != nil {
		if require {
			return []string{"--expect-manifest set but build/<id>/manifest.json missing"}
		}
		return nil
	}
	var doc struct {
		Stages map[string]json.RawMessage `json:"stages"`
		Error  string                     `json:"error"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return []string{fmt.Sprintf("manifest.json is not valid JSON: %v", err)}
	}
	statusOf := func(raw json.RawMessage) string {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
		var obj struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(raw, &obj); err == nil {
			return obj.Status
		}
		return ""
	}
	var issues []string
	for _, id := range config.Stages {
		raw, ok := doc.Stages[id]
		if !ok {
			issues = append(issues, fmt.Sprintf("manifest.json missing stage %q", id))
			continue
		}
		if st := statusOf(raw); st != "ok" {
			issues = append(issues, fmt.Sprintf("manifest.json stage %q status = %q (want \"ok\")", id, st))
		}
	}
	if doc.Error != "" {
		issues = append(issues, "manifest.json reports error: "+doc.Error)
	}
	return issues
}

// checkContracts enforces the v2 stage file contract (SPEC.md §5A). Every
// stage writes build/<id>/contracts/<stage>.json with the digest of each file
// it produced; the Go side re-reads those files so a stage that "succeeded"
// without producing real bytes cannot pass DoD.
func checkContracts(dir string, require bool) []string {
	contractsDir := filepath.Join(dir, "contracts")
	if st, err := os.Stat(contractsDir); err != nil || !st.IsDir() {
		if require {
			return []string{"--expect-contracts set but build/<id>/contracts/ missing"}
		}
		return nil
	}
	id := filepath.Base(filepath.Clean(dir))
	root := filepath.Dir(filepath.Dir(dir)) // build/<id> -> repo root
	var issues []string
	for _, stage := range config.Stages {
		p := filepath.Join(contractsDir, stage+".json")
		data, err := os.ReadFile(p)
		if err != nil {
			issues = append(issues, fmt.Sprintf("missing stage contract %s.json", stage))
			continue
		}
		var c struct {
			Schema    int                    `json:"schema"`
			Asset     string                 `json:"asset"`
			Stage     string                 `json:"stage"`
			FBXExport map[string]interface{} `json:"fbxExport"`
			Outputs   []struct {
				Path   string `json:"path"`
				Bytes  int64  `json:"bytes"`
				SHA256 string `json:"sha256"`
			} `json:"outputs"`
		}
		if err := json.Unmarshal(data, &c); err != nil {
			issues = append(issues, fmt.Sprintf("stage contract %s.json is invalid JSON: %v", stage, err))
			continue
		}
		if c.Schema != 1 {
			issues = append(issues, fmt.Sprintf("stage contract %s.json schema = %d, want 1", stage, c.Schema))
		}
		if c.Stage != stage {
			issues = append(issues, fmt.Sprintf("stage contract %s.json declares stage %q", stage, c.Stage))
		}
		if c.Asset != "" && c.Asset != id {
			issues = append(issues, fmt.Sprintf("stage contract %s.json asset = %q, want %q", stage, c.Asset, id))
		}
		if len(c.Outputs) == 0 {
			issues = append(issues, fmt.Sprintf("stage contract %s.json lists no produced files", stage))
		}
		for _, key := range []string{"axis_forward", "axis_up", "global_scale",
			"apply_unit_scale", "add_leaf_bones", "path_mode"} {
			if _, ok := c.FBXExport[key]; !ok {
				issues = append(issues, fmt.Sprintf("stage contract %s.json missing FBX export key %q", stage, key))
			}
		}
		for _, o := range c.Outputs {
			full := filepath.Join(root, filepath.FromSlash(o.Path))
			raw, err := os.ReadFile(full)
			if err != nil {
				issues = append(issues, fmt.Sprintf("contract %s.json references missing file %s", stage, o.Path))
				continue
			}
			if int64(len(raw)) != o.Bytes {
				issues = append(issues, fmt.Sprintf("contract %s.json: %s size %d != %d", stage, o.Path, len(raw), o.Bytes))
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != strings.ToLower(o.SHA256) {
				issues = append(issues, fmt.Sprintf("contract %s.json: %s SHA-256 mismatch", stage, o.Path))
			}
		}
	}
	return issues
}

// checkArtifactIntegrity verifies that a deliverable is non-empty and that
// binary formats carry their canonical magic bytes. ASCII FBX / OBJ / glTF
// are text formats and only get the non-empty check.
func checkArtifactIntegrity(path, ext string) string {
	// Only delivery artifacts are checked. Markers (.humanoid, deliberately
	// empty) and contracts (.json) are metadata, not model/texture payloads.
	switch ext {
	case ".fbx", ".glb", ".gltf", ".obj", ".png", ".jpg", ".jpeg", ".tga":
	default:
		return ""
	}
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("cannot stat artifact %s: %v", filepath.Base(path), err)
	}
	if st.Size() == 0 {
		return fmt.Sprintf("artifact %s is empty (0 bytes)", filepath.Base(path))
	}
	magic := map[string][]byte{
		".png": {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
		".glb": []byte("glTF"),
	}
	want, ok := magic[ext]
	if !ok {
		return ""
	}
	fh, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("cannot read artifact %s: %v", filepath.Base(path), err)
	}
	defer fh.Close()
	got := make([]byte, len(want))
	if _, err := io.ReadFull(fh, got); err != nil {
		return fmt.Sprintf("artifact %s is truncated (want %d-byte %s header)", filepath.Base(path), len(want), ext)
	}
	if !bytes.Equal(got, want) {
		return fmt.Sprintf("artifact %s has invalid %s magic (%x, want %x)", filepath.Base(path), ext, got, want)
	}
	return ""
}

func anyHint(s string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(s, h) {
			return true
		}
	}
	return false
}
