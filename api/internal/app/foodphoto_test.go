package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	jpg "image/jpeg"
	"regexp"
	"strings"
	"testing"
)

// photoWithLocation is a small JPEG carrying an EXIF block with a place in it.
func photoWithLocation(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := range 64 {
		img.Set(x, 10, color.RGBA{200, 80, 40, 255})
	}
	var b bytes.Buffer
	if err := jpg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	payload := append([]byte("Exif\x00\x00"), []byte("GPS 48.8566N 2.3522E")...)
	if len(payload) != 26 {
		t.Fatal("the segment length below assumes a 26-byte payload")
	}
	app1 := append([]byte{0xFF, 0xE1, 0x00, 28}, payload...) // length includes its own two bytes
	data := b.Bytes()
	return append(append(append([]byte{}, data[:2]...), app1...), data[2:]...)
}

// A photo of a meal (EP-89): only with the consents, sent without its
// metadata, answered with checked lines, counted against the budget.
func TestMealPhotoEstimate(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	photo := photoWithLocation(t)
	post := func() (int, []byte) {
		return alice.do("POST", "/v1/health/food/estimate", photo, "Content-Type", "image/jpeg")
	}
	if code, out := post(); code != 403 {
		t.Fatalf("without consents = %d %s", code, out)
	}
	alice.enableAgents()
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	if code, out := post(); code != 403 || !strings.Contains(string(out), "food_photos") {
		t.Fatalf("without the photo consent = %d %s", code, out)
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/food_photos", []byte(`{"granted":true}`))
	fake.load(finalText(`{"items":[{"name":"Lentil curry","grams":300,"kcal":390,"protein_g":21,"carbs_g":52,"fat_g":9},{"name":"Rice","grams":-1,"kcal":200}],"note":"The portion of rice is hard to judge."}`))
	code, out := post()
	if code != 200 {
		t.Fatalf("estimate = %d %s", code, out)
	}
	var est struct {
		Items []struct {
			Name string  `json:"name"`
			Kcal float64 `json:"kcal"`
		} `json:"items"`
		Note string `json:"note"`
	}
	_ = json.Unmarshal(out, &est)
	if len(est.Items) != 1 || est.Items[0].Name != "Lentil curry" || est.Items[0].Kcal != 390 || est.Note == "" {
		t.Fatalf("estimate = %s", out)
	}
	sent, _ := json.Marshal(fake.lastRequests()[0].Messages)
	m := regexp.MustCompile(`"data":"([^"]+)"`).FindSubmatch(sent)
	if m == nil {
		t.Fatalf("no image sent: %s", sent)
	}
	sentJPEG, _ := base64.StdEncoding.DecodeString(string(m[1]))
	if !bytes.HasPrefix(sentJPEG, []byte{0xFF, 0xD8}) || bytes.Contains(sentJPEG, []byte("Exif")) || bytes.Contains(sentJPEG, []byte("GPS")) {
		t.Fatal("the photo reached the provider with its metadata")
	}
	var runs int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM agent_runs WHERE agent = 'food-photo' AND cost_micros > 0`).Scan(&runs); err != nil || runs != 1 {
		t.Fatalf("recorded runs = %d, %v", runs, err)
	}
	if code, _ := alice.do("POST", "/v1/health/food/estimate", []byte("not a picture"), "Content-Type", "image/jpeg"); code != 422 {
		t.Fatalf("not a picture = %d", code)
	}
}
