package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Transcripts of YouTube videos (EP-123). A video's words are what its
// news is: Axon reads the captions YouTube publishes for it, the author's
// in the spoken language first, else the automatic ones; with none, and
// when the operator runs a Whisper server, it sends the video's smallest
// audio stream there. Nothing is kept but the text.

// Transcript is a video's words, their language, and where they come from:
// captions, auto_captions, or whisper.
type Transcript struct {
	Text   string
	Lang   string
	Source string
}

// ErrNoTranscript means the video has no captions and no Whisper server
// is set up, or it is too long to transcribe.
var ErrNoTranscript = errors.New("no transcript")

// YouTubeBase is where YouTube answers; tests point it elsewhere.
var YouTubeBase = "https://www.youtube.com"

// Limits on what is sent to Whisper.
const (
	maxAudio        = 60 << 20
	maxVideoSeconds = 3 * 60 * 60
	maxTranscript   = 150000 // characters kept
)

type track struct {
	BaseURL string `json:"baseUrl"`
	Lang    string `json:"languageCode"`
	Kind    string `json:"kind"` // "asr" for automatic captions
}

type player struct {
	VideoDetails struct {
		LengthSeconds string `json:"lengthSeconds"`
	} `json:"videoDetails"`
	Captions struct {
		R struct {
			Tracks []track `json:"captionTracks"`
			// With dubbed audio, every language has automatic captions: this
			// names the track of the language spoken in the video.
			Source []int `json:"defaultTranslationSourceTrackIndices"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
	Streaming struct {
		Adaptive []struct {
			URL      string `json:"url"`
			MimeType string `json:"mimeType"`
			Bitrate  int    `json:"bitrate"`
		} `json:"adaptiveFormats"`
	} `json:"streamingData"`
}

// playerInfo asks YouTube's player, as its Android app does, for a video's
// captions and streams.
func playerInfo(ctx context.Context, id string) (player, error) {
	var p player
	body, _ := json.Marshal(map[string]any{
		"context": map[string]any{"client": map[string]any{
			"clientName": "ANDROID", "clientVersion": "20.10.38", "androidSdkVersion": 34, "hl": "en", "gl": "US"}},
		"videoId": id,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, YouTubeBase+"/youtubei/v1/player?prettyPrint=false", bytes.NewReader(body))
	if err != nil {
		return p, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "com.google.android.youtube/20.10.38 (Linux; U; Android 14) gzip")
	res, err := Client.Do(req)
	if err != nil {
		return p, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return p, fmt.Errorf("YouTube answered %d", res.StatusCode)
	}
	return p, json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(&p)
}

// pickTrack prefers the author's captions in the spoken language (the
// language of the automatic ones), then any of the author's, then the
// automatic ones.
func pickTrack(tracks []track) (track, string) { return pickTrackFrom(tracks, -1) }

// primary is a language tag without its region: "en-US" is "en".
func primary(lang string) string {
	l, _, _ := strings.Cut(strings.ToLower(lang), "-")
	return l
}

// pickTrackFrom is pickTrack when YouTube names the spoken language's
// track (source, or -1): with dubbed audio, every dubbed language has its
// automatic captions, and only that one is what was said.
func pickTrackFrom(tracks []track, source int) (track, string) {
	spoken := ""
	if source >= 0 && source < len(tracks) {
		spoken = primary(tracks[source].Lang)
	} else {
		for _, t := range tracks {
			if t.Kind == "asr" && spoken == "" {
				spoken = primary(t.Lang)
			}
		}
	}
	for _, t := range tracks {
		if t.Kind != "asr" && (spoken == "" || primary(t.Lang) == spoken) {
			return t, "captions"
		}
	}
	if source >= 0 && source < len(tracks) {
		return tracks[source], "auto_captions"
	}
	for _, t := range tracks {
		if t.Kind != "asr" {
			return t, "captions"
		}
	}
	for _, t := range tracks {
		if primary(t.Lang) == spoken {
			return t, "auto_captions"
		}
	}
	return track{}, ""
}

// timedText reads YouTube's caption XML (format 3, or the older one) as
// plain text.
func timedText(body []byte) string {
	d := xml.NewDecoder(bytes.NewReader(body))
	var b strings.Builder
	depth := 0
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "p" || t.Name.Local == "text" {
				depth++
				b.WriteByte(' ')
			} else if depth > 0 && t.Name.Local == "s" {
				depth++
			}
		case xml.EndElement:
			if depth > 0 && (t.Name.Local == "p" || t.Name.Local == "text" || t.Name.Local == "s") {
				depth--
			}
		case xml.CharData:
			if depth > 0 {
				b.Write(t)
			}
		}
	}
	return strings.Join(strings.Fields(html.UnescapeString(b.String())), " ")
}

func download(ctx context.Context, raw string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %d", req.URL.Host, res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if int64(len(b)) > limit {
		return nil, ErrNoTranscript
	}
	return b, err
}

// VideoTranscript reads a video's words: its captions, else Whisper.
func VideoTranscript(ctx context.Context, id string, w *Whisper) (Transcript, error) {
	p, err := playerInfo(ctx, id)
	if err != nil {
		return Transcript{}, err
	}
	src := -1
	if len(p.Captions.R.Source) > 0 {
		src = p.Captions.R.Source[0]
	}
	if t, source := pickTrackFrom(p.Captions.R.Tracks, src); source != "" {
		body, err := download(ctx, t.BaseURL, maxBody)
		if err != nil {
			return Transcript{}, err
		}
		if text := timedText(body); text != "" {
			return Transcript{Text: cut(text, maxTranscript), Lang: t.Lang, Source: source}, nil
		}
	}
	if w == nil {
		return Transcript{}, ErrNoTranscript
	}
	if s, _ := strconv.Atoi(p.VideoDetails.LengthSeconds); s > maxVideoSeconds {
		return Transcript{}, ErrNoTranscript
	}
	// The smallest audio stream that can be read without deciphering.
	best, bitrate, mime := "", 0, ""
	for _, f := range p.Streaming.Adaptive {
		if f.URL != "" && strings.HasPrefix(f.MimeType, "audio/") && (best == "" || f.Bitrate < bitrate) {
			best, bitrate, mime = f.URL, f.Bitrate, f.MimeType
		}
	}
	if best == "" {
		return Transcript{}, ErrNoTranscript
	}
	audio, err := download(ctx, best, maxAudio)
	if err != nil {
		return Transcript{}, err
	}
	name := "audio.m4a"
	if strings.Contains(mime, "webm") {
		name = "audio.webm"
	}
	return w.Transcribe(ctx, audio, name)
}

// Whisper is the operator's speech-to-text server, one that speaks the
// OpenAI transcription API (faster-whisper-server, speaches, LocalAI…).
type Whisper struct {
	URL, Model, Token string
}

// WhisperFromEnv is the server the operator set (AXON_WHISPER_URL, and
// AXON_WHISPER_MODEL, AXON_WHISPER_TOKEN), or nil. It is the operator's
// own, so it may be on a private network.
func WhisperFromEnv() *Whisper {
	u := strings.TrimRight(os.Getenv("AXON_WHISPER_URL"), "/")
	if u == "" {
		return nil
	}
	model := os.Getenv("AXON_WHISPER_MODEL")
	if model == "" {
		model = "whisper-1"
	}
	return &Whisper{URL: u, Model: model, Token: os.Getenv("AXON_WHISPER_TOKEN")}
}

var whisperClient = &http.Client{Timeout: 15 * time.Minute}

// whisperLangs maps the languages Whisper names to their codes.
var whisperLangs = map[string]string{"english": "en", "french": "fr", "spanish": "es", "german": "de", "italian": "it",
	"portuguese": "pt", "dutch": "nl", "russian": "ru", "swedish": "sv", "norwegian": "no", "hungarian": "hu",
	"japanese": "ja", "chinese": "zh", "korean": "ko", "arabic": "ar", "hindi": "hi", "polish": "pl", "turkish": "tr"}

// Transcribe sends audio to the Whisper server.
func (w *Whisper) Transcribe(ctx context.Context, audio []byte, name string) (Transcript, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	f, err := mw.CreateFormFile("file", name)
	if err != nil {
		return Transcript{}, err
	}
	if _, err := f.Write(audio); err != nil {
		return Transcript{}, err
	}
	_ = mw.WriteField("model", w.Model)
	_ = mw.WriteField("response_format", "verbose_json")
	if err := mw.Close(); err != nil {
		return Transcript{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL+"/v1/audio/transcriptions", &body)
	if err != nil {
		return Transcript{}, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if w.Token != "" {
		req.Header.Set("Authorization", "Bearer "+w.Token)
	}
	res, err := whisperClient.Do(req)
	if err != nil {
		return Transcript{}, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return Transcript{}, fmt.Errorf("the Whisper server answered %d", res.StatusCode)
	}
	var out struct {
		Text     string `json:"text"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(&out); err != nil {
		return Transcript{}, err
	}
	lang := strings.ToLower(out.Language)
	if code, ok := whisperLangs[lang]; ok {
		lang = code
	} else if len(lang) > 3 {
		lang = ""
	}
	text := strings.TrimSpace(out.Text)
	if text == "" {
		return Transcript{}, ErrNoTranscript
	}
	return Transcript{Text: cut(text, maxTranscript), Lang: lang, Source: "whisper"}, nil
}
