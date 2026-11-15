package reference

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, p string, w, h int, c color.RGBA) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func readPNG(t *testing.T, p string) image.Image {
	t.Helper()
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestPrepareSquareAndBackground(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "concept.png")
	writePNG(t, in, 200, 100, color.RGBA{R: 200, G: 10, B: 10, A: 255})
	outDir := filepath.Join(dir, "out")
	out, err := Prepare(in, outDir, 1024, "808080")
	if err != nil {
		t.Fatal(err)
	}
	img := readPNG(t, out)
	if img.Bounds().Dx() != 1024 || img.Bounds().Dy() != 1024 {
		t.Fatalf("not square: %v", img.Bounds())
	}
	// Margin pixel should be the neutral background.
	bg := img.At(2, 2)
	r, g, b, _ := bg.RGBA()
	if r>>8 != 0x80 || g>>8 != 0x80 || b>>8 != 0x80 {
		t.Fatalf("background not 808080: %v", bg)
	}
}

func TestPrepareRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	if _, err := Prepare(filepath.Join(dir, "nope.png"), dir, 512, "808080"); err == nil {
		t.Fatal("expected error for missing file")
	}
	in := filepath.Join(dir, "concept.png")
	writePNG(t, in, 10, 10, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	if _, err := Prepare(in, dir, 0, "808080"); err == nil {
		t.Fatal("expected error for size<=0")
	}
	if _, err := Prepare(in, dir, 512, "zzz"); err == nil {
		t.Fatal("expected error for bad bg hex")
	}
}

func TestParseHex(t *testing.T) {
	c, err := parseHex("#1a2b3c")
	if err != nil {
		t.Fatal(err)
	}
	if c.R != 0x1a || c.G != 0x2b || c.B != 0x3c || c.A != 255 {
		t.Fatalf("parseHex mismatch: %+v", c)
	}
}
