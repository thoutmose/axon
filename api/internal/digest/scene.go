package digest

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
)

// The picture at the top of a digest: a landscape drawn for the day, with
// the sky of its theme. It needs no image service and no network; the seed
// makes it the same in the email and the chat message.

type palette struct {
	skyTop, skyBottom, sun, land color.RGBA
	night                        bool
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255} //nolint:gosec // bytes of a 24-bit colour
}

var palettes = map[string]palette{
	"start":       {rgb(0x7FB3FF), rgb(0xFFD6A5), rgb(0xFFF1C1), rgb(0x3D5A80), false},
	"focus":       {rgb(0x5B9BD5), rgb(0xCDE7F7), rgb(0xFFFBE6), rgb(0x2F5D50), false},
	"persistence": {rgb(0x6A5ACD), rgb(0xFF9E7A), rgb(0xFFE29A), rgb(0x4A2C4F), false},
	"learning":    {rgb(0x3B4A8C), rgb(0xA7B8F0), rgb(0xFFF4D6), rgb(0x2B2D5C), false},
	"habits":      {rgb(0x8EC5FC), rgb(0xE0F7D4), rgb(0xFFFDE7), rgb(0x2E6B3F), false},
	"rest":        {rgb(0x8E7DBE), rgb(0xF6C6D9), rgb(0xFFF0F5), rgb(0x4B3F72), false},
	"reflection":  {rgb(0x0F1C3F), rgb(0x41507F), rgb(0xF4F1DE), rgb(0x0B1530), true},
	"progress":    {rgb(0xF7B267), rgb(0xFCE38A), rgb(0xFFFFFF), rgb(0x7A4E2D), false},
	"planning":    {rgb(0x2EC4B6), rgb(0xCBF3F0), rgb(0xFFFFFF), rgb(0x1B4D4A), false},
	"simplicity":  {rgb(0xBFD7EA), rgb(0xF5F7FA), rgb(0xFFFFFF), rgb(0x6C7A89), false},
}

// Accent is the theme's colour for headings and borders in rich renderers.
func Accent(theme string) color.RGBA {
	p, ok := palettes[theme]
	if !ok {
		p = palettes["focus"]
	}
	return p.land
}

// SceneSize is the picture's size in pixels.
const (
	SceneWidth  = 1200
	SceneHeight = 360
)

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	return color.RGBA{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		255,
	}
}

// ridge is a smooth random skyline: a sum of sines with random phases.
type ridge struct{ amp, freq, phase [4]float64 }

func newRidge(r *rand.Rand, roughness float64) ridge {
	var g ridge
	for i := range 4 {
		g.amp[i] = roughness / math.Pow(2, float64(i))
		g.freq[i] = (1.2 + r.Float64()*1.5) * math.Pow(2.1, float64(i))
		g.phase[i] = r.Float64() * 2 * math.Pi
	}
	return g
}

func (g ridge) at(x float64) float64 {
	y := 0.0
	for i := range 4 {
		y += g.amp[i] * math.Sin(x*g.freq[i]*2*math.Pi/SceneWidth*2+g.phase[i])
	}
	return y
}

// Scene draws the picture for a digest as a PNG.
func Scene(d Digest) []byte {
	p, ok := palettes[d.Theme]
	if !ok {
		p = palettes["focus"]
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(d.Seed))
	r := rand.New(rand.NewPCG(h.Sum64(), 0x5eed)) //nolint:gosec // a picture, not a secret
	img := image.NewRGBA(image.Rect(0, 0, SceneWidth, SceneHeight))

	// Sky, top to bottom.
	for y := range SceneHeight {
		c := mix(p.skyTop, p.skyBottom, float64(y)/SceneHeight*1.15)
		for x := range SceneWidth {
			img.SetRGBA(x, y, c)
		}
	}
	// Stars at night.
	if p.night {
		for range 140 {
			x, y := r.IntN(SceneWidth), r.IntN(SceneHeight*2/3)
			img.SetRGBA(x, y, mix(p.skyTop, rgb(0xFFFFFF), 0.5+r.Float64()/2))
		}
	}
	// Sun (or moon) with a soft glow.
	sx, sy := float64(SceneWidth)*(0.2+r.Float64()*0.6), float64(SceneHeight)*(0.28+r.Float64()*0.17)
	radius := 34 + r.Float64()*16
	for y := range SceneHeight {
		for x := range SceneWidth {
			dist := math.Hypot(float64(x)-sx, float64(y)-sy)
			switch {
			case dist < radius:
				img.SetRGBA(x, y, p.sun)
			case dist < radius+1.5: // a smooth edge
				img.SetRGBA(x, y, mix(img.RGBAAt(x, y), p.sun, (radius+1.5-dist)/1.5))
			case dist < radius*4:
				glow := (1 - (dist-radius)/(radius*3)) * 0.35
				img.SetRGBA(x, y, mix(img.RGBAAt(x, y), p.sun, glow*glow))
			}
		}
	}
	// Four ranges of hills, nearer ones darker and rougher.
	for layer := range 4 {
		depth := float64(layer) / 3
		base := float64(SceneHeight) * (0.52 + 0.13*float64(layer))
		g := newRidge(r, 22+18*depth)
		col := mix(mix(p.skyBottom, p.land, 0.35), p.land, depth)
		for x := range SceneWidth {
			top := int(base + g.at(float64(x)))
			for y := max(0, top); y < SceneHeight; y++ {
				// A little haze at the foot of each range.
				haze := math.Max(0, 1-float64(y-top)/90) * 0.12 * (1 - depth)
				img.SetRGBA(x, y, mix(col, p.skyBottom, haze))
			}
		}
	}
	// A few birds on a bright day.
	if !p.night {
		for range 2 + r.IntN(3) {
			bx, by := float64(r.IntN(SceneWidth-80)+40), float64(SceneHeight)*(0.15+r.Float64()*0.2)
			span := 7 + r.Float64()*5
			for t := -span; t <= span; t += 0.5 {
				y := by - math.Abs(math.Sin(t/span*math.Pi))*span*0.45
				img.SetRGBA(int(bx+t), int(y), p.land)
				img.SetRGBA(int(bx+t), int(y)+1, p.land)
			}
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	_ = enc.Encode(&buf, img)
	return buf.Bytes()
}
