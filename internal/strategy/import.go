package strategy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ImportStrategy copies an existing model from config.source into raw/.
type ImportStrategy struct{}

func init() { Register(ImportStrategy{}) }

func (ImportStrategy) ID() string          { return "import" }
func (ImportStrategy) Description() string { return "导入已有模型 · 全自动" }
func (ImportStrategy) IsDefault() bool     { return false }

func (ImportStrategy) Acquire(ctx *RunContext) (*AcquireResult, error) {
	c := ctx.Config
	src := c.Source
	if src == "" {
		return nil, fmt.Errorf("import strategy requires config.source")
	}
	if !filepath.IsAbs(src) {
		src = filepath.Join(ctx.RootDir, src)
	}
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("import source not found: %s", src)
	}
	ext := strings.ToLower(filepath.Ext(src))
	if ext != ".glb" && ext != ".gltf" && ext != ".fbx" && ext != ".obj" {
		return nil, fmt.Errorf("unsupported source extension %q (want .glb/.gltf/.fbx/.obj)", ext)
	}
	dst := filepath.Join(ctx.RawDir(), c.ID+ext)
	if ctx.DryRun {
		return &AcquireResult{InputPath: dst, Note: "dry-run: would copy " + src + " -> " + dst}, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return nil, err
	}
	if err := copyFile(src, dst); err != nil {
		return nil, err
	}
	return &AcquireResult{InputPath: dst, Note: "copied " + src + " -> " + dst}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
