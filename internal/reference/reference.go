// Package reference normalizes concept art into a Tripo-friendly square image.
// Pure Go 1.27 standard library — NO third-party dependencies (SPEC.md §2).
package reference

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Prepare reads src, fits it into a size×size square with a neutral grey
// background (match the Tripo reference-image guidance), and writes PNG.
func Prepare(src, outDir string, size int, bgHex string) (string, error) {
	if size <= 0 {
		return "", errors.New("--size must be > 0")
	}
	bg, err := parseHex(bgHex)
	if err != nil {
		return "", err
	}
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()
	srcImg, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode %s (supported: PNG/JPEG): %w", src, err)
	}

	// 6% margin, like a human cropping for a front-view turnaround.
	inner := int(float64(size) * 0.88)
	sw, sh := fit(srcImg.Bounds().Dx(), srcImg.Bounds().Dy(), inner, inner)

	canvas := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	scaled := image.NewRGBA(image.Rect(0, 0, sw, sh))
	// Nearest-neighbour-free resample via Re-Render (stdlib has no bilinear;
	// use a simple box filter for predictable quality).
	boxScale(scaled, srcImg)

	x := (size - sw) / 2
	y := (size - sh) / 2
	draw.Draw(canvas, image.Rect(x, y, x+sw, y+sh), scaled, image.Point{}, draw.Over)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	out := filepath.Join(outDir, base+".png")
	of, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer of.Close()
	if err := png.Encode(of, canvas); err != nil {
		return "", err
	}
	return out, nil
}

// boxScale is a small area-average downscaler (stdlib-only).
func boxScale(dst *image.RGBA, src image.Image) {
	sb := src.Bounds()
	db := dst.Bounds()
	for dy := 0; dy < db.Dy(); dy++ {
		y0 := sb.Min.Y + dy*sb.Dy()/db.Dy()
		y1 := sb.Min.Y + (dy+1)*sb.Dy()/db.Dy()
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for dx := 0; dx < db.Dx(); dx++ {
			x0 := sb.Min.X + dx*sb.Dx()/db.Dx()
			x1 := sb.Min.X + (dx+1)*sb.Dx()/db.Dx()
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a, n uint64
			for yy := y0; yy < y1 && yy < sb.Max.Y; yy++ {
				for xx := x0; xx < x1 && xx < sb.Max.X; xx++ {
					cr, cg, cb, ca := src.At(xx, yy).RGBA()
					r += uint64(cr >> 8)
					g += uint64(cg >> 8)
					b += uint64(cb >> 8)
					a += uint64(ca >> 8)
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			dst.Set(db.Min.X+dx, db.Min.Y+dy, color.RGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n),
			})
		}
	}
}

func fit(w, h, maxW, maxH int) (int, int) {
	if w <= maxW && h <= maxH {
		return w, h
	}
	ratio := float64(maxW) / float64(w)
	if rh := float64(maxH) / float64(h); rh < ratio {
		ratio = rh
	}
	nw := int(float64(w) * ratio)
	nh := int(float64(h) * ratio)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return nw, nh
}

func parseHex(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid --bg %q (want 6 hex chars, e.g. 808080)", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid --bg %q: %w", s, err)
	}
	return color.RGBA{
		R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255,
	}, nil
}

// RegisterDecodeers ensures JPEG decoder is linked (PNG is registered by image/png).
func init() { _ = jpeg.DefaultQuality }

// FlagSet exposes flags for the `assetctl reference prepare` subcommand.
func FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("reference prepare", flag.ContinueOnError)
	fs.Int("size", 1024, "output square edge in pixels")
	fs.String("bg", "808080", "background hex color")
	return fs
}
