// Package images re-encodes uploaded pictures (EP-16): decode PNG or JPEG,
// scale down to fit a maximum side, and write PNG, so only pixels survive —
// no EXIF, no location, nothing executable.
package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // decoder for uploads
	"image/png"
)

// ErrNotImage is returned for files that are not usable PNG or JPEG pictures.
var ErrNotImage = errors.New("not a PNG or JPEG image")

// Fit decodes data and returns a PNG no larger than maxSide on either side.
func Fit(data []byte, maxSide int) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotImage
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width*cfg.Height > 40_000_000 {
		return nil, fmt.Errorf("%w: use a picture up to 40 megapixels", ErrNotImage)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotImage
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	w, h := b.Dx(), b.Dy()
	if w > maxSide || h > maxSide {
		if w >= h {
			w, h = maxSide, max(1, h*maxSide/b.Dx())
		} else {
			w, h = max(1, w*maxSide/b.Dy()), maxSide
		}
		rgba = Resize(rgba, w, h)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Resize scales src to w×h by averaging the source pixels behind each target
// pixel (a box filter: simple, and good for shrinking).
func Resize(src *image.RGBA, w, h int) *image.RGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := y*sh/h, max((y+1)*sh/h, y*sh/h+1)
		for x := 0; x < w; x++ {
			x0, x1 := x*sw/w, max((x+1)*sw/w, x*sw/w+1)
			var r, g, b, a, n uint64
			for sy := y0; sy < y1 && sy < sh; sy++ {
				for sx := x0; sx < x1 && sx < sw; sx++ {
					c := src.RGBAAt(sx, sy)
					r, g, b, a = r+uint64(c.R), g+uint64(c.G), b+uint64(c.B), a+uint64(c.A)
					n++
				}
			}
			if n > 0 {
				// An average of 8-bit values is at most 255.
				dst.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), uint8(a / n)}) //nolint:gosec // see above
			}
		}
	}
	return dst
}
