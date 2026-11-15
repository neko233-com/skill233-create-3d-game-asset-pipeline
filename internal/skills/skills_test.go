package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/runner"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCatalogIntegrity(t *testing.T) {
	if Count() != 28 {
		t.Fatalf("catalog has %d skills, want 28", Count())
	}
	seenID := map[string]bool{}
	seenOp := map[string]bool{}
	for _, s := range Catalog {
		if seenID[s.ID] {
			t.Fatalf("duplicate skill id %q", s.ID)
		}
		if seenOp[s.Operation] {
			t.Fatalf("duplicate operation %q", s.Operation)
		}
		seenID[s.ID], seenOp[s.Operation] = true, true
		if s.Category == "" || s.Summary == "" || s.Operation == "" {
			t.Fatalf("incomplete skill %+v", s)
		}
	}
	if got, ok := ByID("blender-asset-qa"); !ok || got.Operation != "asset_qa" {
		t.Fatalf("ByID(blender-asset-qa) = %+v, %v", got, ok)
	}
	if _, ok := ByID("nope"); ok {
		t.Fatal("unknown id must not resolve")
	}
	ids := IDs()
	for i := 1; i < len(ids); i++ {
		if ids[i-1] > ids[i] {
			t.Fatalf("IDs() not sorted: %v", ids)
		}
	}
	if len(Operations()) != Count() {
		t.Fatal("Operations() length mismatch")
	}
}

func TestValidateCatalogAgainstRepo(t *testing.T) {
	root := repoRoot(t)
	if err := ValidateCatalog(root); err != nil {
		t.Fatalf("catalog / manifest / docs drift: %v", err)
	}
	m, err := LoadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.License != "MIT" || m.BlenderTarget != "5.2" || len(m.Skills) != Count() {
		t.Fatalf("manifest drift: %+v", m)
	}
	if err := ValidateCatalog(t.TempDir()); err == nil {
		t.Fatal("ValidateCatalog must fail on a directory without skills/")
	}
}

func TestRunDryRunAndValidation(t *testing.T) {
	var out bytes.Buffer
	res, err := Run(Options{
		RootDir: t.TempDir(), SkillID: "blender-core", AssetID: "hero-knight",
		DryRun: true, Stdout: &out, Stderr: &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.SkillID != "blender-core" || res.Operation != "core" {
		t.Fatalf("dry-run result drift: %+v", res)
	}
	if !bytes.Contains(out.Bytes(), []byte("--skill blender-core")) {
		t.Fatalf("dry-run did not print the blender argv:\n%s", out.String())
	}
	if _, err := os.Stat(res.ContractPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write a contract (err=%v)", err)
	}
	for _, tc := range []struct {
		name string
		opt  Options
	}{
		{"unknown skill", Options{SkillID: "blender-nope", DryRun: true}},
		{"bad asset id", Options{SkillID: "blender-core", AssetID: "BAD_ID", DryRun: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.opt.DryRun = true
			tc.opt.Stdout = &bytes.Buffer{}
			tc.opt.Stderr = &bytes.Buffer{}
			if _, err := Run(tc.opt); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

// TestRunAll28SkillsWithFakeBlender executes EVERY registered skill through
// the real skill_run.py under mock_bpy — no Blender 5.2 installation needed.
func TestRunAll28SkillsWithFakeBlender(t *testing.T) {
	root := repoRoot(t)
	launcher, err := runner.NewScriptLauncher(root)
	if err != nil {
		t.Fatalf("fake_blender harness unavailable: %v", err)
	}
	work := t.TempDir()
	for _, s := range Catalog {
		t.Run(s.ID, func(t *testing.T) {
			res, err := Run(Options{
				RootDir: work, SkillID: s.ID, AssetID: "skill-probe",
				Params:     map[string]string{"supplier": "tripo", "maxTriangles": "30000"},
				Launcher:   launcher,
				ScriptsDir: filepath.Join(root, "scripts", "blender"),
				Stdout:     &bytes.Buffer{}, Stderr: &bytes.Buffer{},
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.Operation != s.Operation {
				t.Fatalf("operation = %q, want %q", res.Operation, s.Operation)
			}
			if res.Fields["skill"] != s.ID || res.Fields["stage"] != "skill-"+s.ID {
				t.Fatalf("contract identity drift: %v", res.Fields)
			}
			if res.Fields["source"] != "skills/manifest.json" {
				t.Fatalf("contract missing source: %v", res.Fields)
			}
			outputs, ok := res.Fields["outputs"].([]interface{})
			if !ok || len(outputs) == 0 {
				t.Fatalf("skill produced no contracted outputs: %v", res.Fields["outputs"])
			}
			if _, err := os.Stat(res.ContractPath); err != nil {
				t.Fatal(err)
			}
		})
	}
	contracts := filepath.Join(work, "build", "skill-probe", "contracts")
	entries, err := os.ReadDir(contracts)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != Count() {
		t.Fatalf("expected %d skill contracts, got %d", Count(), len(entries))
	}
}
