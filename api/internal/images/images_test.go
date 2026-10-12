package images

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestFitScalesDownKeepingProportions(t *testing.T) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3000, 1500)))
	out, err := Fit(buf.Bytes(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	img, _, _ := image.Decode(bytes.NewReader(out))
	if img.Bounds().Dx() != 1024 || img.Bounds().Dy() != 512 {
		t.Fatalf("size = %v", img.Bounds())
	}
	if _, err := Fit([]byte("GIF89a"), 1024); err == nil {
		t.Fatal("accepted a non-image")
	}
}
