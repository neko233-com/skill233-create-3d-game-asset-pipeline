// Package validate checks a built asset directory before Unity import.
package validate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var assetIDRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var pbrHints = []string{"basecolor", "albedo", "normal", "roughness", "metallic", "ao"}

// Options for Validate.
type Options struct {
	ExpectHumanoid bool
	ExpectTextures bool
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
	return r, nil
}

func anyHint(s string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(s, h) {
			return true
		}
	}
	return false
}
