package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/demo"
)

func pngOf(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 180
	}
	img.Set(0, 0, color.Black)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func (c *client) avatar() (int, string, []byte) {
	c.e.t.Helper()
	req, _ := http.NewRequest("GET", c.e.srv.URL+"/v1/me/avatar", nil)
	res, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(res.Body)
	checkContract(c.e.t, req, nil, res, buf.Bytes())
	return res.StatusCode, res.Header.Get("Content-Type"), buf.Bytes()
}

func TestProfilePictureWithAGeneratedDefault(t *testing.T) {
	e := newEnvWith(t, auth.Config{Demo: true})
	alice := e.client()
	alice.register("Ada Lovelace")

	code, ct, body := alice.avatar()
	if code != 200 || ct != "image/svg+xml" || !bytes.Contains(body, []byte(">AL<")) {
		t.Fatalf("default = %d %s %s", code, ct, body)
	}
	if code, _ := alice.do("PUT", "/v1/me/avatar", []byte("not a picture"), "Content-Type", "image/png"); code != 422 {
		t.Fatalf("garbage = %d", code)
	}
	if code, _ := alice.do("PUT", "/v1/me/avatar", pngOf(40, 40), "Content-Type", "image/gif"); code != 415 {
		t.Fatalf("gif = %d", code)
	}
	if code, out := alice.do("PUT", "/v1/me/avatar", noisyPNG(400, 300), "Content-Type", "image/png"); code != 204 {
		t.Fatalf("upload = %d %s", code, out)
	}
	code, ct, body = alice.avatar()
	img, err := png.Decode(bytes.NewReader(body))
	if code != 200 || ct != "image/png" || err != nil || img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
		t.Fatalf("uploaded = %d %s %v", code, ct, err)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me", nil), []byte(`"avatar_updated_at":"`)) {
		t.Fatal("me lacks the avatar version")
	}
	alice.mustDo(204, "DELETE", "/v1/me/avatar", nil)
	if _, ct, _ := alice.avatar(); ct != "image/svg+xml" {
		t.Fatalf("after delete = %s", ct)
	}

	// The shared demo cannot change its picture.
	if _, err := demo.Reset(context.Background(), pool, time.Now()); err != nil {
		t.Fatal(err)
	}
	visitor := e.client()
	visitor.mustDo(204, "POST", "/v1/auth/demo", nil)
	if code, _ := visitor.do("PUT", "/v1/me/avatar", pngOf(40, 40), "Content-Type", "image/png"); code != 403 {
		t.Fatalf("demo upload = %d", code)
	}
}
