// Package runner orchestrates the six-stage game-art pipeline (SPEC.md §4).
package runner

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/strategy"
)

// Manifest is written to build/<id>/manifest.json after every run.
type Manifest struct {
	AssetID        string            `json:"assetId"`
	Strategy       string            `json:"strategy"`
	BlenderVersion string            `json:"blenderVersion"`
	TripoURL       string            `json:"tripoUrl"`
	GoVersion      string            `json:"goVersion"`
	StartedAt      string            `json:"startedAt"`
	FinishedAt     string            `json:"finishedAt,omitempty"`
	Stages         map[string]string `json:"stages"` // stage -> ok | failed | skipped | pending
	Error          string            `json:"error,omitempty"`
}

type stageDef struct {
	id         string
	script     string
	blendName  string
	primaryOut func(c *Options) string
}

// Options controls one `assetctl run`.
type Options struct {
	RootDir    string
	ConfigPath string
	AssetID    string
	StrategyID string // override config.Strategy
	From       string
	Only       []string
	DryRun     bool
	BlenderExe string
	Stdout     io.Writer
	Stderr     io.Writer
}

// six stages, fixed order (SPEC.md §4). Stage 1 input comes from the strategy.
func stageDefs() []stageDef {
	return []stageDef{
		{"modeling", "stage_01_modeling.py", "01-modeling.blend",
			func(o *Options) string { return filepath.Join(o.RootDir, "work", o.AssetID, "01-modeling.blend") }},
		{"topology-uv", "stage_02_topology_uv.py", "02-topology-uv.blend",
			func(o *Options) string { return filepath.Join(o.RootDir, "work", o.AssetID, "02-topology-uv.blend") }},
		{"bake", "stage_03_bake.py", "03-bake.blend",
			func(o *Options) string { return filepath.Join(o.RootDir, "work", o.AssetID, "03-bake.blend") }},
		{"pbr", "stage_04_pbr.py", "04-pbr.blend",
			func(o *Options) string { return filepath.Join(o.RootDir, "work", o.AssetID, "04-pbr.blend") }},
		{"unity-export", "stage_05_unity_export.py", "05-unity-export.blend",
			func(o *Options) string {
				return filepath.Join(o.RootDir, "build", o.AssetID, o.AssetID+".fbx")
			}},
		{"render", "stage_06_render.py", "06-render.blend",
			func(o *Options) string { return filepath.Join(o.RootDir, "build", o.AssetID, "preview.png") }},
	}
}

// FindBlender resolves the Blender 5.2 executable. Order: explicit flag,
// ASSETCTL_BLENDER env, PATH, then platform-specific common install dirs.
func FindBlender(explicit string) (string, error) {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	}
	if env := os.Getenv("ASSETCTL_BLENDER"); env != "" {
		candidates = append(candidates, env)
	}
	if p, err := exec.LookPath("blender"); err == nil {
		candidates = append(candidates, p)
	}
	if runtime.GOOS == "windows" {
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		base := filepath.Join(programFiles, "Blender Foundation")
		for _, v := range []string{"Blender 5.2", "Blender 5.1", "Blender 4.5"} {
			candidates = append(candidates, filepath.Join(base, v, "blender.exe"))
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("Blender %s executable not found; pass --blender <path> or set ASSETCTL_BLENDER",
		config.BlenderVersion)
}

// Run executes the pipeline. Exit-code friendly error values are returned as-is.
func Run(opts Options) (*Manifest, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.RootDir == "" {
		opts.RootDir = "."
	}
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	if opts.AssetID == "" {
		opts.AssetID = cfg.ID
	}
	if opts.StrategyID != "" {
		cfg.Strategy = opts.StrategyID
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	m := &Manifest{
		AssetID:        cfg.ID,
		Strategy:       cfg.Strategy,
		BlenderVersion: config.BlenderVersion,
		TripoURL:       config.TripoURL,
		GoVersion:      config.DefaultGo,
		StartedAt:      time.Now().Format(time.RFC3339),
		Stages:         map[string]string{},
	}
	defs := stageDefs()
	for _, d := range defs {
		m.Stages[d.id] = "pending"
	}

	selected := selectStages(defs, opts.From, opts.Only)
	if len(selected) == 0 {
		return nil, fmt.Errorf("no stages selected (from=%q only=%v)", opts.From, opts.Only)
	}
	if !containsStage(selected, "modeling") {
		// resuming: stages before `from` are considered done
		for _, d := range defs {
			if indexOf(defs, d.id) < indexOf(defs, opts.From) || (len(opts.Only) > 0 && !inList(opts.Only, d.id)) {
				if m.Stages[d.id] == "pending" {
					m.Stages[d.id] = "skipped"
				}
			}
		}
	}

	s, err := strategy.Get(cfg.Strategy)
	if err != nil {
		return nil, err
	}
	ctx := &strategy.RunContext{RootDir: opts.RootDir, Config: cfg, DryRun: opts.DryRun}

	var stage1Input string
	if containsStage(selected, "modeling") {
		res, acqErr := s.Acquire(ctx)
		if acqErr != nil {
			m.Error = acqErr.Error()
			m.FinishedAt = time.Now().Format(time.RFC3339)
			_ = writeManifest(opts, m)
			return m, acqErr
		}
		stage1Input = res.InputPath
		fmt.Fprintf(opts.Stdout, "[strategy] %s (default=%s) -> %s\n",
			s.ID(), strategy.DefaultID(), res.Note)
	}

	blenderExe := opts.BlenderExe
	if !opts.DryRun {
		blenderExe, err = FindBlender(opts.BlenderExe)
		if err != nil {
			return m, err
		}
		fmt.Fprintf(opts.Stdout, "[blender] %s\n", blenderExe)
	}

	prevOut := stage1Input
	if prevOut == "" && !containsStage(selected, "modeling") {
		// Resuming with --from / --only: the first selected stage must read the
		// output of the stage immediately before it (the previously-completed
		// one), not an empty --in. e.g. --from topology-uv reads
		// work/<id>/01-modeling.blend; --only unity-export reads 04-pbr.blend.
		if idx := indexOf(defs, selected[0].id); idx > 0 {
			prevOut = defs[idx-1].primaryOut(&opts)
		}
	}
	for _, d := range selected {
		out := d.primaryOut(&opts)
		blendOut := filepath.Join(opts.RootDir, "work", opts.AssetID, d.blendName)
		if err := runStage(opts, blenderExe, d, prevOut, out, blendOut, cfg); err != nil {
			m.Stages[d.id] = "failed"
			m.Error = fmt.Sprintf("stage %s: %v", d.id, err)
			m.FinishedAt = time.Now().Format(time.RFC3339)
			_ = writeManifest(opts, m)
			return m, fmt.Errorf("stage %s: %w", d.id, err)
		}
		m.Stages[d.id] = "ok"
		fmt.Fprintf(opts.Stdout, "[stage] %-14s OK -> %s\n", d.id, out)
		prevOut = out
	}

	// Humanoid gate: after stage 5, verify the skeleton (SPEC.md §5).
	if cfg.Humanoid && (containsStage(selected, "unity-export") || stageAfter(selected, "unity-export")) {
		if err := checkHumanoidRig(opts, blenderExe, cfg); err != nil {
			m.Error = "rig_check: " + err.Error()
			m.FinishedAt = time.Now().Format(time.RFC3339)
			_ = writeManifest(opts, m)
			return m, err
		}
		fmt.Fprintln(opts.Stdout, "[rig] humanoid rig_check OK")
	}

	m.FinishedAt = time.Now().Format(time.RFC3339)
	if err := writeManifest(opts, m); err != nil {
		return m, err
	}
	return m, nil
}

func runStage(opts Options, blenderExe string, d stageDef, in, out, blendOut string, cfg *config.Config) error {
	script := filepath.Join(opts.RootDir, "scripts", "blender", d.script)
	args := []string{
		"--background", "--factory-startup", "--python", script, "--",
		"--asset", cfg.ID,
		"--config", opts.ConfigPath,
		"--in", in,
		"--out", out,
		"--blend-out", blendOut,
	}
	// Dry-run is purely informational: no script existence check, no dirs,
	// no Blender lookup. It just prints the six commands that would execute.
	if opts.DryRun {
		fmt.Fprintf(opts.Stdout, "[dry-run] blender %s\n", strings.Join(args, " "))
		return nil
	}
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("missing stage script %s", script)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(blendOut), 0o755); err != nil {
		return err
	}
	cmd := exec.Command(blenderExe, args...)
	cmd.Dir = opts.RootDir
	return stream(cmd, opts.Stdout, opts.Stderr)
}

func checkHumanoidRig(opts Options, blenderExe string, cfg *config.Config) error {
	script := filepath.Join(opts.RootDir, "scripts", "blender", "rig_check.py")
	fbx := filepath.Join(opts.RootDir, "build", cfg.ID, cfg.ID+".fbx")
	args := []string{"--background", "--factory-startup", "--python", script, "--", "--input", fbx}
	if opts.DryRun {
		fmt.Fprintf(opts.Stdout, "[dry-run] blender %s\n", strings.Join(args, " "))
		return nil
	}
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("missing rig_check.py: %s", script)
	}
	cmd := exec.Command(blenderExe, args...)
	cmd.Dir = opts.RootDir
	return stream(cmd, opts.Stdout, opts.Stderr)
}

func stream(cmd *exec.Cmd, stdout, stderr io.Writer) error {
	pOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	pErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go forward(pOut, stdout)
	go forward(pErr, stderr)
	return cmd.Wait()
}

func forward(r io.Reader, w io.Writer) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		fmt.Fprintln(w, s.Text())
	}
}

func writeManifest(opts Options, m *Manifest) error {
	if opts.DryRun {
		return nil
	}
	dir := filepath.Join(opts.RootDir, "build", m.AssetID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
}

func selectStages(defs []stageDef, from string, only []string) []stageDef {
	if len(only) > 0 {
		out := []stageDef{}
		for _, d := range defs {
			if inList(only, d.id) {
				out = append(out, d)
			}
		}
		return out
	}
	start := 0
	if from != "" {
		for i, d := range defs {
			if d.id == from {
				start = i
				break
			}
		}
	}
	return defs[start:]
}

func containsStage(defs []stageDef, id string) bool {
	for _, d := range defs {
		if d.id == id {
			return true
		}
	}
	return false
}

func stageAfter(defs []stageDef, id string) bool {
	// true if any selected stage comes after `id` in pipeline order
	all := stageDefs()
	idx := indexOf(all, id)
	for _, d := range defs {
		if indexOf(all, d.id) > idx {
			return true
		}
	}
	return false
}

func indexOf(defs []stageDef, id string) int {
	for i, d := range defs {
		if d.id == id {
			return i
		}
	}
	return -1
}

func inList(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// IsManualStep reports whether an error is the Tripo "human must download" case.
func IsManualStep(err error) bool {
	return errors.Is(err, strategy.ErrManualStepRequired)
}
