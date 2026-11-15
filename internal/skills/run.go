// Execution of one registered skill. Go owns validation + process launch;
// Blender 5.2's bundled bpy performs the actual scene work.
package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/runner"
)

var assetIDRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Options controls one `assetctl skills run <id>`.
type Options struct {
	RootDir    string
	SkillID    string
	AssetID    string
	ConfigPath string
	Params     map[string]string
	DryRun     bool
	BlenderExe string
	Launcher   runner.Launcher // nil -> real Blender 5.2 (CommandLauncher)
	ScriptsDir string
	Stdout     io.Writer
	Stderr     io.Writer
}

// Result is the parsed skill contract.
type Result struct {
	SkillID      string
	Operation    string
	ContractPath string
	Fields       map[string]interface{}
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// Run validates, launches skill_run.py and verifies the produced contract.
func Run(opts Options) (*Result, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.RootDir == "" {
		opts.RootDir = "."
	}
	if opts.AssetID == "" {
		opts.AssetID = "skill-probe"
	}
	if !assetIDRe.MatchString(opts.AssetID) {
		return nil, fmt.Errorf("invalid asset id %q (want lowercase-kebab)", opts.AssetID)
	}
	s, ok := ByID(opts.SkillID)
	if !ok {
		return nil, fmt.Errorf("unknown skill %q (%d registered; try `assetctl skills list`)", opts.SkillID, Count())
	}
	if opts.ScriptsDir == "" {
		opts.ScriptsDir = filepath.Join(opts.RootDir, "scripts", "blender")
	}
	if opts.ConfigPath == "" {
		opts.ConfigPath = filepath.Join(opts.RootDir, "assets", opts.AssetID, "asset.json")
	}

	contract := filepath.Join(opts.RootDir, "build", opts.AssetID,
		"contracts", "skill-"+s.ID+".json")
	blendOut := filepath.Join(opts.RootDir, "work", opts.AssetID, "skills", s.ID+".blend")
	script := filepath.Join(opts.ScriptsDir, "skill_run.py")

	args := []string{
		"--background", "--factory-startup", "--python", script, "--",
		"--skill", s.ID,
		"--asset", opts.AssetID,
		"--config", opts.ConfigPath,
		"--out", contract,
		"--blend-out", blendOut,
	}
	keys := make([]string, 0, len(opts.Params))
	for k := range opts.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic argv for tests / snapshots
	for _, k := range keys {
		args = append(args, "--param", k+"="+opts.Params[k])
	}

	if opts.DryRun {
		fmt.Fprintf(opts.Stdout, "[dry-run] blender %s\n", strings.Join(args, " "))
		return &Result{SkillID: s.ID, Operation: s.Operation, ContractPath: contract}, nil
	}
	if _, err := os.Stat(script); err != nil {
		return nil, fmt.Errorf("missing bpy dispatcher %s", script)
	}
	if err := os.MkdirAll(filepath.Dir(contract), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(blendOut), 0o755); err != nil {
		return nil, err
	}

	launcher := opts.Launcher
	if launcher == nil {
		exe, err := runner.FindBlender(opts.BlenderExe)
		if err != nil {
			return nil, err
		}
		launcher = runner.CommandLauncher{Exe: exe}
	}
	cmd := launcher.Command(args, opts.RootDir)
	var mu sync.Mutex
	cmd.Stdout = lockedWriter{mu: &mu, w: opts.Stdout}
	cmd.Stderr = lockedWriter{mu: &mu, w: opts.Stderr}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("skill %s: %w", s.ID, err)
	}

	data, err := os.ReadFile(contract)
	if err != nil {
		return nil, fmt.Errorf("skill %s published no contract at %s: %w", s.ID, contract, err)
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("skill %s contract is invalid JSON: %w", s.ID, err)
	}
	if fmt.Sprint(fields["schema"]) != "1" {
		return nil, fmt.Errorf("skill %s contract schema = %v, want 1", s.ID, fields["schema"])
	}
	if fields["stage"] != "skill-"+s.ID || fields["skill"] != s.ID {
		return nil, fmt.Errorf("skill %s contract identity mismatch: %v", s.ID, fields)
	}
	if fields["operation"] != s.Operation {
		return nil, fmt.Errorf("skill %s contract operation = %v, want %s", s.ID,
			fields["operation"], s.Operation)
	}
	return &Result{
		SkillID: s.ID, Operation: s.Operation,
		ContractPath: contract, Fields: fields,
	}, nil
}

// ErrNoResult is returned when a caller asks for a result from a dry run.
var ErrNoResult = errors.New("dry run produced no contract")
