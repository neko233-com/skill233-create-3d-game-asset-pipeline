// assetctl — Go 1.27 CLI orchestrator for the Unity 3D game-asset pipeline.
// Zero third-party dependencies. See SPEC.md §8 for the command contract.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/reference"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/runner"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/strategy"
	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/validate"
)

const version = "assetctl 2.0.0 (go1.27, blender5.2, tripo=https://studio.tripo3d.com/, default-supplier=tripo)"

// reorderFlags moves leading flag tokens (and their values) ahead of the
// positional arguments so Go's flag.FlagSet can parse them regardless of
// where the user placed them. It keeps the first token's position as the
// command/subcommand.
func reorderFlags(args []string) []string {
	if len(args) < 2 {
		return args
	}
	// `assetctl blender <script> -- <args...>` uses a literal `--` passthrough;
	// reordering would corrupt it, so leave that command untouched.
	if args[0] == "blender" {
		return args
	}
	// Protect the command and (for `reference prepare`) the subcommand.
	head := []string{args[0]}
	rest := args[1:]
	if args[0] == "reference" && len(rest) > 0 && rest[0] == "prepare" {
		head = append(head, "prepare")
		rest = rest[1:]
	}
	flags := []string{}
	pos := []string{}
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		// Normalize --flag to -flag so flagTakesValue and flag.FlagSet agree.
		if strings.HasPrefix(a, "--") && len(a) > 2 {
			a = "-" + a[2:]
		}
		if strings.HasPrefix(a, "-") && a != "-" && a != "--" {
			flags = append(flags, a)
			// Value flags consume the next token, even if it starts with a
			// drive letter or contains path separators.
			if !strings.Contains(a, "=") && flagTakesValue(a) && i+1 < len(rest) {
				i++
				flags = append(flags, rest[i])
			}
		} else {
			pos = append(pos, a)
		}
	}
	out := append([]string{}, head...)
	out = append(out, flags...)
	out = append(out, pos...)
	return out
}

func flagTakesValue(name string) bool {
	switch name {
	case "-strategy", "-supplier", "-kind", "-root", "-from", "-only",
		"-blender", "-size", "-bg", "-handoff", "-source":
		return true
	}
	return false
}

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		if runner.IsManualStep(err) {
			// Tripo half-automatic: guidance already printed by the strategy.
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func dispatch(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	// Go's flag package stops at the first positional argument, which makes
	// natural CLI usage (`assetctl init hero-knight --humanoid`) fail. Reorder
	// so flags can appear before or after positional args.
	args = reorderFlags(args)
	switch args[0] {
	case "version", "-v", "--version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	case "strategies":
		return cmdStrategies(args[1:])
	case "stages":
		return cmdStages()
	case "init":
		return cmdInit(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "reference":
		return cmdReference(args[1:])
	case "validate":
		return cmdValidate(args[1:])
	case "doctor":
		return cmdDoctor(args[1:])
	case "blender":
		return cmdBlender(args[1:])
	default:
		return fmt.Errorf("unknown command %q (try `assetctl help`)", args[0])
	}
}

func usage() {
	fmt.Printf(`%s

游戏美术管线向：建模 → 拓扑/UV → 烘焙 → PBR 材质 → Unity 导出/同步 → 渲染合成
默认生产供应商：tripo (DEFAULT) / blender / import | 语言：Go 1.27 + Blender 5.2 原生 bpy
Tripo handoff：manual | file | fixture（默认供应商也可脚本化 / CI 无人值守）

Commands:
  version                       打印版本与工具链
  strategies                    列出建模策略（标注 DEFAULT）
  stages                        列出六阶段
  init <asset-id>               创建 assets/<id>/asset.json
  run <asset-id>                执行流水线
  reference prepare <in> <out>  归一化参考图（纯 Go，无第三方依赖）
  validate <build-dir>          交付前校验 (DoD，含 manifest 六阶段全 ok)
  doctor                        工具链 / 供应商 / 脚本齐备性报告
  blender <script> [-- args...] 透传 Blender 5.2

Run: assetctl <command> -h  for command flags.
`, version)
}

func cmdStrategies(args []string) error {
	fs := flag.NewFlagSet("strategies", flag.ContinueOnError)
	_ = fs.Parse(args)
	fmt.Printf("%-10s %-9s %s\n", "ID", "DEFAULT", "DESCRIPTION")
	for _, s := range strategy.All() {
		def := ""
		if s.IsDefault() {
			def = "YES"
		}
		fmt.Printf("%-10s %-9s %s\n", s.ID(), def, s.Description())
	}
	fmt.Printf("\n切换供应商：assetctl init <id> --strategy blender | --supplier tripo\n")
	fmt.Printf("Tripo 入口：%s\n", config.TripoURL)
	return nil
}

func cmdStages() error {
	for i, s := range config.Stages {
		fmt.Printf("%d. %s\n", i+1, s)
	}
	return nil
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	st := fs.String("strategy", config.DefaultSupplier, "modeling supplier: tripo|blender|import")
	supplier := fs.String("supplier", "", "alias for --strategy (overrides it)")
	kind := fs.String("kind", "prop", "asset kind: prop|character")
	humanoid := fs.Bool("humanoid", false, "humanoid rig + animation")
	handoff := fs.String("handoff", "manual", "tripo handoff mode: manual|file|fixture")
	source := fs.String("source", "", "handoff file source (also used by import strategy)")
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: assetctl init <asset-id> [--strategy tripo] [--supplier <id>] [--handoff manual|file|fixture] [--source <path>] [--kind character] [--humanoid]")
	}
	id := fs.Arg(0)
	c := config.Default(id)
	c.Strategy = *st
	if *supplier != "" {
		c.Strategy = *supplier
	}
	c.Kind = *kind
	c.Humanoid = *humanoid
	c.Handoff = config.Handoff{Mode: *handoff, Source: *source}
	if *source != "" {
		c.Source = *source
	}
	if *humanoid {
		c.Animations = []string{"idle", "walk", "attack"}
	}
	if err := c.Validate(); err != nil {
		return err
	}
	path := filepath.Join(*root, "assets", id, "asset.json")
	if err := c.Save(path); err != nil {
		return err
	}
	fmt.Printf("[init] created %s (strategy=%s, default=%s, handoff=%s)\n",
		path, c.Strategy, strategy.DefaultID(), c.Handoff.Mode)
	if c.Strategy == "tripo" {
		switch c.Handoff.Mode {
		case "manual":
			fmt.Printf("[init] Tripo 人工步骤见 %s ，下载 GLB 到 raw/%s.glb\n", config.TripoURL, id)
		case "file":
			fmt.Printf("[init] Tripo file handoff：把 %s 放到 raw/%s.glb（或运行前设 ASSETCTL_TRIPO_SOURCE）\n", c.Handoff.Source, id)
		case "fixture":
			fmt.Printf("[init] Tripo fixture handoff：CI 无人值守，自动生成确定性最小 GLB\n")
		}
	}
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	st := fs.String("strategy", "", "override config.strategy")
	from := fs.String("from", "", "resume from stage (e.g. topology-uv)")
	only := fs.String("only", "", "comma-separated stages to run")
	dry := fs.Bool("dry-run", false, "print commands, do not execute")
	blenderExe := fs.String("blender", "", "path to Blender 5.2 executable")
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: assetctl run <asset-id> [--strategy <id>] [--from <stage>] [--only <stage,...>] [--dry-run]")
	}
	id := fs.Arg(0)
	var onlyList []string
	if *only != "" {
		onlyList = strings.Split(*only, ",")
		for i := range onlyList {
			onlyList[i] = strings.TrimSpace(onlyList[i])
		}
		for _, s := range onlyList {
			found := false
			for _, known := range config.Stages {
				if s == known {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("unknown stage in --only: %q", s)
			}
		}
	}
	if *from != "" {
		ok := false
		for _, s := range config.Stages {
			if s == *from {
				ok = true
			}
		}
		if !ok {
			return fmt.Errorf("unknown stage in --from: %q", *from)
		}
	}
	cfgPath := filepath.Join(*root, "assets", id, "asset.json")
	m, err := runner.Run(runner.Options{
		RootDir:    *root,
		ConfigPath: cfgPath,
		AssetID:    id,
		StrategyID: *st,
		From:       *from,
		Only:       onlyList,
		DryRun:     *dry,
		BlenderExe: *blenderExe,
	})
	if err != nil {
		return err
	}
	fmt.Printf("[run] manifest -> %s\n", filepath.Join(*root, "build", id, "manifest.json"))
	_ = m
	return nil
}

func cmdReference(args []string) error {
	if len(args) == 0 || args[0] != "prepare" {
		return errors.New("usage: assetctl reference prepare <in.png> <outdir> [--size 1024] [--bg 808080]")
	}
	fs := flag.NewFlagSet("reference prepare", flag.ContinueOnError)
	size := fs.Int("size", 1024, "output square edge")
	bg := fs.String("bg", "808080", "background hex")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: assetctl reference prepare <in.png> <outdir> [--size 1024] [--bg 808080]")
	}
	out, err := reference.Prepare(fs.Arg(0), fs.Arg(1), *size, *bg)
	if err != nil {
		return err
	}
	fmt.Printf("[reference] %s -> %s (%dx%d)\n", fs.Arg(0), out, *size, *size)
	return nil
}

func cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	humanoid := fs.Bool("expect-humanoid", false, "require humanoid rig marker")
	textures := fs.Bool("expect-textures", false, "require PBR textures")
	manifest := fs.Bool("expect-manifest", true, "require manifest.json with six ok stages (DoD)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: assetctl validate <build-dir> [--expect-humanoid] [--expect-textures] [--expect-manifest=true|false]")
	}
	r, err := validate.Validate(fs.Arg(0), validate.Options{
		ExpectHumanoid: *humanoid,
		ExpectTextures: *textures,
		ExpectManifest: *manifest,
	})
	if err != nil {
		return err
	}
	if !r.OK() {
		fmt.Fprintf(os.Stderr, "[validate] FAIL for %s:\n", r.Dir)
		for _, i := range r.Issues {
			fmt.Fprintln(os.Stderr, "  -", i)
		}
		return fmt.Errorf("%d issue(s)", len(r.Issues))
	}
	fmt.Printf("[validate] OK: %s (%d model(s), %d texture(s))\n",
		filepath.Base(filepath.Clean(r.Dir)), len(r.Models), len(r.Textures))
	return nil
}

// cmdDoctor reports the toolchain / supplier / repo-layout state (SPEC.md §3A.3).
// A missing Blender 5.2 is a WARN (not fatal): CI runs with ScriptLauncher.
// Only genuine repo defects (missing stage scripts) make doctor exit non-zero.
func cmdDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Println("[doctor] assetctl toolchain report")
	fmt.Printf("  Go version        : %s (target %s)\n", runtime.Version(), config.DefaultGo)
	fmt.Printf("  Blender version   : %s (required)\n", config.BlenderVersion)
	exe, err := runner.FindBlender("")
	if err != nil {
		fmt.Printf("  Blender executable: WARN not found (%v)\n", err)
		fmt.Println("                       production `assetctl run` needs Blender 5.2;")
		fmt.Println("                       `go test ./...` / CI use ScriptLauncher (fake_blender.py).")
	} else {
		fmt.Printf("  Blender executable: OK %s\n", exe)
	}
	if env := os.Getenv("ASSETCTL_BLENDER"); env != "" {
		fmt.Printf("  ASSETCTL_BLENDER  : %s\n", env)
	}
	fmt.Printf("  Suppliers         : %s (DEFAULT=%s)\n", strings.Join(strategy.IDs(), ", "), strategy.DefaultID())
	fmt.Printf("  Tripo URL         : %s\n", config.TripoURL)
	fmt.Printf("  Unity axes        : forward=-Z up=Y scale=1.0 (Bake Axis Conversion)\n")
	if h := os.Getenv("ASSETCTL_TRIPO_HANDOFF"); h != "" {
		fmt.Printf("  ASSETCTL_TRIPO_HANDOFF: %s\n", h)
	}
	if s := os.Getenv("ASSETCTL_TRIPO_SOURCE"); s != "" {
		fmt.Printf("  ASSETCTL_TRIPO_SOURCE : %s\n", s)
	}

	scripts := filepath.Join(*root, "scripts", "blender")
	want := []string{
		"common.py", "stage_01_modeling.py", "stage_02_topology_uv.py",
		"stage_03_bake.py", "stage_04_pbr.py", "stage_05_unity_export.py",
		"stage_06_render.py", "rig_check.py", "humanoid_rig.py", "tripo_to_unity.py",
	}
	missing := []string{}
	for _, name := range want {
		if _, err := os.Stat(filepath.Join(scripts, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		fmt.Printf("  Stage scripts     : FAIL missing %s\n", strings.Join(missing, ", "))
		return fmt.Errorf("doctor: missing %d scripts/blender file(s)", len(missing))
	}
	fmt.Printf("  Stage scripts     : OK (%d files in %s)\n", len(want), scripts)
	fmt.Println("[doctor] OK (no fatal repo defects)")
	return nil
}

func cmdBlender(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: assetctl blender <script.py> [-- --args...]")
	}
	exe, err := runner.FindBlender("")
	if err != nil {
		return err
	}
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	var script string
	var extra []string
	if sep >= 0 {
		script = args[0]
		extra = args[sep+1:]
	} else {
		script = args[0]
		extra = args[1:]
	}
	blenderArgs := []string{"--background", "--factory-startup", "--python", script}
	if len(extra) > 0 {
		blenderArgs = append(blenderArgs, "--")
		blenderArgs = append(blenderArgs, extra...)
	}
	cmd := exec.Command(exe, blenderArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
