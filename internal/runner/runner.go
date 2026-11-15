// Package runner orchestrates the six-stage game-art pipeline (SPEC.md §4).
package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

// Launcher abstracts how a stage process is spawned (SPEC.md §3A.2).
// Production uses CommandLauncher (real Blender 5.2 + bundled bpy); CI / tests
// use ScriptLauncher, which runs testing/fake_blender.py — a pure-stdlib bpy +
// mathutils simulator — so `go test ./...` needs no Blender installation.
type Launcher interface {
	Command(argv []string, dir string) *exec.Cmd
	Label() string
}

// CommandLauncher launches real Blender 5.2 with the blender-style argv.
type CommandLauncher struct{ Exe string }

func (c CommandLauncher) Command(argv []string, dir string) *exec.Cmd {
	cmd := exec.Command(c.Exe, argv...)
	cmd.Dir = dir
	return cmd
}
func (c CommandLauncher) Label() string { return "blender" }

// ScriptLauncher launches testing/fake_blender.py with the SAME blender-style
// argv. The Python shim parses --python and the literal -- and re-exposes them
// to the stage script, so stage code is byte-for-byte identical in both modes.
type ScriptLauncher struct {
	Python string
	Script string // absolute path to testing/fake_blender.py
}

func (s ScriptLauncher) Command(argv []string, dir string) *exec.Cmd {
	full := append([]string{s.Script}, argv...)
	cmd := exec.Command(s.Python, full...)
	cmd.Dir = dir
	return cmd
}
func (s ScriptLauncher) Label() string { return "fake_blender.py" }

// NewScriptLauncher resolves a Python interpreter and the fake_blender.py shim
// under repoRoot. ASSETCTL_PYTHON overrides interpreter detection.
func NewScriptLauncher(repoRoot string) (ScriptLauncher, error) {
	python := os.Getenv("ASSETCTL_PYTHON")
	if python == "" {
		for _, c := range []string{"python3", "python", "py"} {
			if p, err := exec.LookPath(c); err == nil {
				python = p
				break
			}
		}
	}
	if python == "" {
		return ScriptLauncher{}, fmt.Errorf("python interpreter not found; set ASSETCTL_PYTHON")
	}
	script := filepath.Join(repoRoot, "testing", "fake_blender.py")
	if _, err := os.Stat(script); err != nil {
		return ScriptLauncher{}, fmt.Errorf("missing fake_blender.py: %s", script)
	}
	return ScriptLauncher{Python: python, Script: script}, nil
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
	Launcher   Launcher // optional; when set, overrides CommandLauncher (CI/tests)
	ScriptsDir string   // optional; default <RootDir>/scripts/blender
	Stdout     io.Writer
	Stderr     io.Writer
}

// scriptsDir returns the directory holding the six stage scripts + rig_check.
// In E2E tests this points at the repo's scripts/blender (absolute), while the
// working files live under a temp RootDir.
func scriptsDir(opts Options) string {
	if opts.ScriptsDir != "" {
		return opts.ScriptsDir
	}
	return filepath.Join(opts.RootDir, "scripts", "blender")
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

// FindBlender resolves the Blender 5.2 executable on Windows / Linux / macOS.
// Order: explicit flag, ASSETCTL_BLENDER, PATH, then OS-specific install dirs.
// Every candidate is version-checked, so a 4.x found on PATH cannot silently
// drive a 5.2 pipeline. Set ASSETCTL_SKIP_BLENDER_VERSION=1 to bypass probing
// (e.g. offline CI smoke tests).
func FindBlender(explicit string) (string, error) {
	var lastErr error
	for _, c := range blenderCandidates(explicit) {
		st, err := os.Stat(c)
		if err != nil || st.IsDir() {
			continue
		}
		if os.Getenv("ASSETCTL_SKIP_BLENDER_VERSION") != "1" {
			if err := CheckBlenderVersion(c); err != nil {
				lastErr = err
				continue
			}
		}
		return c, nil
	}
	msg := fmt.Sprintf("Blender %s executable not found; pass --blender <path> or set ASSETCTL_BLENDER",
		config.BlenderVersion)
	if lastErr != nil {
		return "", fmt.Errorf("%s (last candidate rejected: %w)", msg, lastErr)
	}
	return "", errors.New(msg)
}

// blenderCandidates lists platform-specific Blender install locations.
func blenderCandidates(explicit string) []string {
	var out []string
	add := func(p string) {
		if p != "" {
			out = append(out, p)
		}
	}
	add(explicit)
	add(os.Getenv("ASSETCTL_BLENDER"))
	if p, err := exec.LookPath("blender"); err == nil {
		add(p)
	}
	if home, err := os.UserHomeDir(); err == nil {
		switch runtime.GOOS {
		case "darwin":
			for _, app := range []string{"Blender.app", "Blender 5.2.app"} {
				add(filepath.Join("/Applications", app, "Contents", "MacOS", "Blender"))
				add(filepath.Join(home, "Applications", app, "Contents", "MacOS", "Blender"))
			}
		case "linux":
			for _, p := range []string{
				"/usr/bin/blender", "/usr/local/bin/blender", "/opt/blender/blender",
				"/opt/blender-5.2/blender", "/snap/bin/blender",
			} {
				add(p)
			}
			add(filepath.Join(home, "blender", "blender"))
		}
	}
	if runtime.GOOS == "windows" {
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		// ONLY the 5.2 series is supported (SPEC.md §2); older installs are
		// deliberately not probed so a stale 4.x can never be picked up.
		add(filepath.Join(programFiles, "Blender Foundation", "Blender 5.2", "blender.exe"))
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			add(filepath.Join(local, "Programs", "Blender Foundation", "Blender 5.2", "blender.exe"))
		}
	}
	return out
}

// CheckBlenderVersion runs `blender --version` and requires the 5.2 series.
// It is exported so doctor/tests can report a precise reason.
func CheckBlenderVersion(exe string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s --version: %w", exe, err)
	}
	text := string(out)
	if !strings.Contains(text, "Blender "+config.BlenderVersion) {
		first := strings.TrimSpace(text)
		if len(first) > 120 {
			first = first[:120]
		}
		return fmt.Errorf("%s reports %q, want Blender %s", exe, first, config.BlenderVersion)
	}
	return nil
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

	// Resume semantics (SPEC.md §3A.1): --from / --only must preserve stages
	// already reported "ok" by a previous run. Without this, a resumed run
	// overwrites completed stages with "skipped" and the DoD gate in
	// validate.go can never pass. A FULL run always starts from scratch.
	resuming := opts.From != "" || len(opts.Only) > 0
	if resuming && !opts.DryRun {
		if prev, err := readManifest(filepath.Join(opts.RootDir, "build", m.AssetID, "manifest.json")); err == nil {
			for id, status := range prev.Stages {
				if status == "ok" {
					if _, known := m.Stages[id]; known {
						m.Stages[id] = "ok"
					}
				}
			}
		}
	}
	for _, d := range defs {
		if !containsStage(selected, d.id) && m.Stages[d.id] != "ok" {
			m.Stages[d.id] = "skipped"
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

	// Resolve how stages will be launched. Dry-run never needs a launcher.
	var launcher Launcher
	if !opts.DryRun {
		if opts.Launcher != nil {
			launcher = opts.Launcher
			fmt.Fprintf(opts.Stdout, "[launcher] %s\n", launcher.Label())
		} else {
			exe, ferr := FindBlender(opts.BlenderExe)
			if ferr != nil {
				return m, ferr
			}
			fmt.Fprintf(opts.Stdout, "[blender] %s\n", exe)
			launcher = CommandLauncher{Exe: exe}
		}
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
		if err := runStage(opts, launcher, d, prevOut, out, blendOut, cfg); err != nil {
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
		if err := checkHumanoidRig(opts, launcher, cfg); err != nil {
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

func runStage(opts Options, launcher Launcher, d stageDef, in, out, blendOut string, cfg *config.Config) error {
	script := filepath.Join(scriptsDir(opts), d.script)
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
	cmd := launcher.Command(args, opts.RootDir)
	if err := stream(cmd, opts.Stdout, opts.Stderr); err != nil {
		return fmt.Errorf("launch %s: %w", d.id, err)
	}
	// SPEC.md §3A.2: every real stage must actually produce its declared --out,
	// otherwise a silently-failing Blender/mock script would be reported as OK.
	if info, err := os.Stat(out); err != nil || info.IsDir() {
		return fmt.Errorf("stage %s: declared --out not produced: %s", d.id, out)
	}
	return nil
}

func checkHumanoidRig(opts Options, launcher Launcher, cfg *config.Config) error {
	script := filepath.Join(scriptsDir(opts), "rig_check.py")
	fbx := filepath.Join(opts.RootDir, "build", cfg.ID, cfg.ID+".fbx")
	args := []string{"--background", "--factory-startup", "--python", script, "--", "--input", fbx}
	if opts.DryRun {
		fmt.Fprintf(opts.Stdout, "[dry-run] blender %s\n", strings.Join(args, " "))
		return nil
	}
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("missing rig_check.py: %s", script)
	}
	cmd := launcher.Command(args, opts.RootDir)
	return stream(cmd, opts.Stdout, opts.Stderr)
}

// lockedWriter serialises concurrent stdout/stderr writes. os/exec copies the
// two streams from separate goroutines, so without the mutex a single shared
// buffer (tests, logs) is a data race under `go test -race`.
type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// stream uses Cmd.Stdout/Stderr instead of manual StdoutPipe+Wait goroutines:
// os/exec guarantees the copy goroutines finish before Run returns, so stage
// output is never truncated and no Scanner line-length limit applies.
func stream(cmd *exec.Cmd, stdout, stderr io.Writer) error {
	var mu sync.Mutex
	cmd.Stdout = lockedWriter{mu: &mu, w: stdout}
	cmd.Stderr = lockedWriter{mu: &mu, w: stderr}
	return cmd.Run()
}

// readManifest loads a previous run's manifest for resume semantics.
func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Stages == nil {
		return nil, errors.New("manifest has no stages")
	}
	return &m, nil
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
