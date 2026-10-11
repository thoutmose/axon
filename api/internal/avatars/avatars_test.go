package avatars

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func picture(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 200, 255})
		}
	}
	return img
}

// withText inserts a PNG tEXt chunk (where cameras and apps put metadata).
func withText(t *testing.T, img image.Image, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	b := buf.Bytes()
	data := append([]byte("Comment\x00"), text...)
	chunk := make([]byte, 0, len(data)+12)
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(len(data)))
	body := append([]byte("tEXt"), data...)
	chunk = append(chunk, body...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(body))
	return append(append(append([]byte{}, b[:33]...), chunk...), b[33:]...) // after the IHDR chunk
}

func TestNormalizeStripsMetadataCropsAndResizes(t *testing.T) {
	in := withText(t, picture(600, 400), "GPS 48.8566 2.3522")
	if !bytes.Contains(in, []byte("GPS")) {
		t.Fatal("fixture lacks metadata")
	}
	out, err := normalize(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("GPS")) {
		t.Fatal("metadata survived")
	}
	img, format, err := image.Decode(bytes.NewReader(out))
	if err != nil || format != "png" || img.Bounds().Dx() != Size || img.Bounds().Dy() != Size {
		t.Fatalf("out = %v %s %v", img.Bounds(), format, err)
	}

	var j bytes.Buffer
	_ = jpeg.Encode(&j, picture(300, 900), nil)
	if _, err := normalize(j.Bytes()); err != nil {
		t.Fatalf("jpeg: %v", err)
	}
}

func TestNormalizeRefusesNonImagesAndTinyOnes(t *testing.T) {
	if _, err := normalize([]byte("<svg onload=alert(1)>")); err == nil {
		t.Fatal("accepted a non-image")
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, picture(8, 8))
	if _, err := normalize(buf.Bytes()); err == nil {
		t.Fatal("accepted a tiny image")
	}
}

func TestDefaultIsStableAndEscaped(t *testing.T) {
	id := uuid.MustParse("01a12ad2-3edd-7d5a-a4e2-e9d1bf5655b3")
	a, b := Default(id, "Ada Lovelace"), Default(id, "Ada Lovelace")
	if !bytes.Equal(a, b) || !strings.Contains(string(a), ">AL<") {
		t.Fatalf("default = %s", a)
	}
	if s := string(Default(id, "<script> x")); strings.Contains(s, "<script") || !strings.Contains(s, "&lt;X") {
		t.Fatalf("not escaped: %s", s)
	}
}
