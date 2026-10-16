package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Speech is the operator's text-to-speech server, one that speaks the OpenAI
// speech API (Kokoro-FastAPI, LocalAI, openedai-speech…), for a podcast in
// natural voices (EP-132). Like Whisper, it is the operator's own, so it may
// be on a private network. Voices of real people are not offered.
type Speech struct {
	URL, Model, Token string
}

// SpeechFromEnv is the server the operator set (AXON_TTS_URL, and
// AXON_TTS_MODEL, AXON_TTS_TOKEN), or nil.
func SpeechFromEnv() *Speech {
	u := strings.TrimRight(os.Getenv("AXON_TTS_URL"), "/")
	if u == "" {
		return nil
	}
	model := os.Getenv("AXON_TTS_MODEL")
	if model == "" {
		model = "kokoro"
	}
	return &Speech{URL: u, Model: model, Token: os.Getenv("AXON_TTS_TOKEN")}
}

// A part can take a minute on a CPU; the answer is at most a few megabytes.
var speechClient = &http.Client{Timeout: 3 * time.Minute}

const maxSpeech = 20 << 20

// Voice is one of the server's voices and the language its name says.
type Voice struct {
	ID   string `json:"id"`
	Lang string `json:"lang"`
}

// kokoroLangs: Kokoro names its voices by language, then sex: af_heart is an
// American English woman's voice, ff_siwis a French one.
var kokoroLangs = map[byte]string{'a': "en", 'b': "en", 'f': "fr", 'e': "es", 'i': "it", 'p': "pt", 'j': "ja", 'z': "zh", 'h': "hi"}

var voiceRe = regexp.MustCompile(`^[A-Za-z0-9_.+-]{1,64}$`)

func (s *Speech) do(req *http.Request) (*http.Response, error) {
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	res, err := speechClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		_ = res.Body.Close()
		return nil, fmt.Errorf("speech server: %s: %s", res.Status, bytes.TrimSpace(msg))
	}
	return res, nil
}

// Voices lists the server's voices (Kokoro's GET /v1/audio/voices).
func (s *Speech) Voices(ctx context.Context) ([]Voice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL+"/v1/audio/voices", nil)
	if err != nil {
		return nil, err
	}
	res, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var body struct {
		Voices []string `json:"voices"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body); err != nil {
		return nil, err
	}
	out := []Voice{}
	for _, id := range body.Voices {
		if !voiceRe.MatchString(id) {
			continue
		}
		v := Voice{ID: id}
		if len(id) > 2 && id[2] == '_' {
			v.Lang = kokoroLangs[id[0]]
		}
		out = append(out, v)
	}
	return out, nil
}

// Say speaks text in a voice at a speed, as MP3.
func (s *Speech) Say(ctx context.Context, text, voice string, speed float64) ([]byte, error) {
	payload, _ := json.Marshal(map[string]any{"model": s.Model, "input": text, "voice": voice, "response_format": "mp3", "speed": speed})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL+"/v1/audio/speech", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	audio, err := io.ReadAll(io.LimitReader(res.Body, maxSpeech+1))
	if err == nil && len(audio) > maxSpeech {
		err = fmt.Errorf("speech server: more than %d bytes", maxSpeech)
	}
	return audio, err
}
