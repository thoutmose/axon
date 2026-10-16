package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpeechServer(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/audio/voices":
			_, _ = w.Write([]byte(`{"voices":["af_heart","ff_siwis","bm_george","custom","bad voice!"]}`))
		case "/v1/audio/speech":
			_ = json.NewDecoder(r.Body).Decode(&got)
			if got["input"] == "fail" {
				http.Error(w, "model not loaded", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("ID3 mp3"))
		}
	}))
	defer srv.Close()
	t.Setenv("AXON_TTS_URL", srv.URL+"/")
	t.Setenv("AXON_TTS_TOKEN", "secret")
	s := SpeechFromEnv()

	vs, err := s.Voices(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []Voice{{"af_heart", "en"}, {"ff_siwis", "fr"}, {"bm_george", "en"}, {"custom", ""}}
	if len(vs) != len(want) {
		t.Fatalf("voices = %+v", vs)
	}
	for i := range want {
		if vs[i] != want[i] {
			t.Errorf("voice %d = %+v, want %+v", i, vs[i], want[i])
		}
	}

	audio, err := s.Say(t.Context(), "Hello", "af_heart", 1.1)
	if err != nil || string(audio) != "ID3 mp3" {
		t.Fatalf("say = %q, %v", audio, err)
	}
	if got["model"] != "kokoro" || got["voice"] != "af_heart" || got["response_format"] != "mp3" || got["speed"] != 1.1 {
		t.Errorf("sent %v", got)
	}
	if _, err := s.Say(t.Context(), "fail", "af_heart", 1); err == nil {
		t.Error("a failing server is an error")
	}
	t.Setenv("AXON_TTS_URL", "")
	if SpeechFromEnv() != nil {
		t.Error("no address, no server")
	}
}
