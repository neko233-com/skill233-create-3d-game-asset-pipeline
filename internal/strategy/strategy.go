// Package strategy implements pluggable modeling strategies for stage 1.
// Per SPEC.md §3: "blender" is the DEFAULT strategy, "tripo" is one of many.
package strategy

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neko233-com/skill233-create-3d-game-asset-pipeline/internal/config"
)

// ErrManualStepRequired means a human must act (e.g. download a GLB from the
// Tripo web UI) before the run can continue. The runner prints guidance and
// exits with a dedicated code instead of a stack trace.
var ErrManualStepRequired = errors.New("manual step required")

// RunContext carries everything a strategy needs for one asset run.
type RunContext struct {
	RootDir string       // repo root
	Config  *config.Config
	DryRun  bool
}

// AcquireResult describes what stage 1 should consume.
// InputPath empty => stage_01_modeling.py generates geometry procedurally
// from config.modeling (the "blender" default strategy).
type AcquireResult struct {
	InputPath string // raw/<id>.glb / .fbx / .obj, or "" for procedural
	Note      string // human-readable line for the log
}

// Strategy is the extension point. Implement ID/Description/IsDefault/Acquire
// and call Register in init(). See SPEC.md §3.
type Strategy interface {
	ID() string
	Description() string
	IsDefault() bool
	Acquire(ctx *RunContext) (*AcquireResult, error)
}

var registry = map[string]Strategy{}

// Register adds a strategy. Panics on duplicate IDs (programmer error).
func Register(s Strategy) {
	if s == nil {
		panic("strategy: nil strategy")
	}
	if _, exists := registry[s.ID()]; exists {
		panic("strategy: duplicate id " + s.ID())
	}
	registry[s.ID()] = s
}

// Get returns a registered strategy by ID.
func Get(id string) (Strategy, error) {
	s, ok := registry[id]
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q (registered: %s)", id, strings.Join(IDs(), ", "))
	}
	return s, nil
}

// IDs returns sorted registered strategy IDs.
func IDs() []string {
	out := make([]string, 0, len(registry))
	for id := range registry {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// All returns all strategies sorted by ID.
func All() []Strategy {
	ids := IDs()
	out := make([]Strategy, 0, len(ids))
	for _, id := range ids {
		out = append(out, registry[id])
	}
	return out
}

// DefaultID returns the ID of the default strategy ("blender").
func DefaultID() string {
	for _, s := range All() {
		if s.IsDefault() {
			return s.ID()
		}
	}
	return ""
}

// RawDir / WorkDir / BuildDir are the canonical layout (SPEC.md §6).
func (c *RunContext) RawDir() string   { return filepath.Join(c.RootDir, "raw") }
func (c *RunContext) WorkDir() string  { return filepath.Join(c.RootDir, "work", c.Config.ID) }
func (c *RunContext) BuildDir() string { return filepath.Join(c.RootDir, "build", c.Config.ID) }
func (c *RunContext) StageOut(n int, stage string) string {
	return filepath.Join(c.WorkDir(), fmt.Sprintf("%02d-%s.blend", n, stage))
}
