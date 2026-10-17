package watch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimedText(t *testing.T) {
	srv3 := `<?xml version="1.0" encoding="utf-8" ?><timedtext format="3"><body>
<p t="160" d="2640">Go 1.22 changes <s>how</s><s> loops</s> work</p>
<p t="2800" d="1000">and that&#39;s it.</p></body></timedtext>`
	if got := timedText([]byte(srv3)); got != "Go 1.22 changes how loops work and that's it." {
		t.Fatalf("srv3 = %q", got)
	}
	srv1 := `<transcript><text start="0" dur="1">Bonjour &amp;amp; bienvenue</text><text start="1" dur="1">à tous</text></transcript>`
	if got := timedText([]byte(srv1)); got != "Bonjour & bienvenue à tous" {
		t.Fatalf("srv1 = %q", got)
	}
}

func TestPickTrack(t *testing.T) {
	tr := func(lang, kind string) track { return track{BaseURL: "u-" + lang + kind, Lang: lang, Kind: kind} }
	// The author's captions in the spoken language beat automatic ones and other languages.
	got, src := pickTrack([]track{tr("en", ""), tr("fr", "asr"), tr("fr", "")})
	if got.Lang != "fr" || got.Kind != "" || src != "captions" {
		t.Fatalf("picked %+v %s", got, src)
	}
	// A dubbed video has automatic captions in every dubbed language; the
	// source track YouTube names is the spoken one.
	got, src = pickTrackFrom([]track{tr("nl-NL", "asr"), tr("en", "asr"), tr("fr-FR", "asr")}, 1)
	if got.Lang != "en" || src != "auto_captions" {
		t.Fatalf("dubbed: picked %+v %s", got, src)
	}
	got, src = pickTrackFrom([]track{tr("en-US", ""), tr("nl-NL", "asr"), tr("en", "asr")}, 2)
	if got.Lang != "en-US" || src != "captions" {
		t.Fatalf("author's in the spoken language: picked %+v %s", got, src)
	}
	got, src = pickTrack([]track{tr("de", "asr")})
	if got.Lang != "de" || src != "auto_captions" {
		t.Fatalf("picked %+v %s", got, src)
	}
	if _, src = pickTrack(nil); src != "" {
		t.Fatal("picked from nothing")
	}
}

// A fake YouTube and a fake Whisper server: captions when there are some,
// else the audio, transcribed.
func TestVideoTranscript(t *testing.T) {
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	var whisperGot string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/youtubei/v1/player":
			var body struct {
				VideoID string `json:"videoId"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.VideoID == "withCaption" {
				//nolint:gosec // a test server pointing at itself
				_, _ = w.Write([]byte(`{"videoDetails":{"lengthSeconds":"300"},"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":[
					{"baseUrl":"http://` + r.Host + `/api/timedtext?v=1","languageCode":"en","kind":"asr"}]}}}`))
				return
			}
			//nolint:gosec // a test server pointing at itself
			_, _ = w.Write([]byte(`{"videoDetails":{"lengthSeconds":"60"},"streamingData":{"adaptiveFormats":[
				{"url":"http://` + r.Host + `/video.mp4","mimeType":"video/mp4","bitrate":900000},
				{"url":"http://` + r.Host + `/audio.m4a","mimeType":"audio/mp4; codecs=\"mp4a.40.2\"","bitrate":128000},
				{"url":"http://` + r.Host + `/small.webm","mimeType":"audio/webm; codecs=\"opus\"","bitrate":50000}]}}`))
		case "/api/timedtext":
			_, _ = w.Write([]byte(`<timedtext format="3"><body><p t="0" d="1">Raft elects a leader</p></body></timedtext>`))
		case "/small.webm":
			_, _ = w.Write([]byte("opus-bytes"))
		case "/v1/audio/transcriptions":
			if r.Header.Get("Authorization") != "Bearer s3cret" {
				w.WriteHeader(401)
				return
			}
			f, _, err := r.FormFile("file")
			if err == nil {
				b, _ := io.ReadAll(f)
				whisperGot = string(b) + "|" + r.FormValue("model")
			}
			_, _ = w.Write([]byte(`{"text":" Bonjour, voici Raft. ","language":"french"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := YouTubeBase
	YouTubeBase = srv.URL
	defer func() { YouTubeBase = old }()
	ctx := context.Background()

	tr, err := VideoTranscript(ctx, "withCaption", nil)
	if err != nil || tr.Text != "Raft elects a leader" || tr.Lang != "en" || tr.Source != "auto_captions" {
		t.Fatalf("captions = %+v, %v", tr, err)
	}
	// No captions and no Whisper: nothing to read.
	if _, err := VideoTranscript(ctx, "noCaptions1", nil); !errors.Is(err, ErrNoTranscript) {
		t.Fatalf("without whisper: %v", err)
	}
	// No captions: the smallest audio goes to Whisper.
	w := &Whisper{URL: srv.URL, Model: "small", Token: "s3cret"}
	tr, err = VideoTranscript(ctx, "noCaptions1", w)
	if err != nil || tr.Text != "Bonjour, voici Raft." || tr.Lang != "fr" || tr.Source != "whisper" || whisperGot != "opus-bytes|small" {
		t.Fatalf("whisper = %+v, %v, sent %q", tr, err, whisperGot)
	}
}
