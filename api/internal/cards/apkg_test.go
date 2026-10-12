package cards

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestAPKGForAnki writes a sample package when AXON_APKG_OUT is set, so the
// file can be imported with the official Anki library (see doc/operations.md).
func TestAPKGForAnki(t *testing.T) {
	var img bytes.Buffer
	_ = png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 40, 30)))
	list := []Card{
		{ID: uuid.Must(uuid.NewV7()), Deck: "Learning science", Front: "Why space practice?", Back: "Forgetting a little\nmakes retrieval stronger.", HasImage: true},
		{ID: uuid.Must(uuid.NewV7()), Deck: "Rust", Front: "What does `Option<T>` mean?", Back: "A value that may be absent & <safe>."},
	}
	data, err := APKG(context.Background(), list, func(uuid.UUID) ([]byte, error) { return img.Bytes(), nil }, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 1000 {
		t.Fatalf("package too small: %d", len(data))
	}
	if out := os.Getenv("AXON_APKG_OUT"); out != "" {
		if err := os.WriteFile(out, data, 0o600); err != nil { //nolint:gosec // a path the developer chose for the check
			t.Fatal(err)
		}
	}
}
