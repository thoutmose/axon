// Package semantic finds notes by meaning (EP-62). Each person may choose
// an embedding service: a local model server (Ollama, LM Studio, anything
// with an OpenAI-style /v1/embeddings), OpenAI, or Voyage AI, and must give
// the ai_embeddings consent. Notes are embedded there; the vectors are kept
// in PostgreSQL (pgvector) and compared by cosine distance. Without both,
// Axon's local TF-IDF similarity (EP-37) is what there is.
package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/privacy"
)

// Purpose is the consent this needs.
const Purpose = "ai_embeddings"

// Providers and their defaults.
var providers = map[string]struct{ base, model string }{
	"local":  {"", "nomic-embed-text"},
	"openai": {"https://api.openai.com", "text-embedding-3-small"},
	"voyage": {"https://api.voyageai.com", "voyage-3.5-lite"},
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Settings are a person's embedding service; the key is never returned.
type Settings struct {
	Provider string  `json:"provider"`
	BaseURL  *string `json:"base_url"`
	Model    string  `json:"model"`
	HasKey   bool    `json:"has_key"`
	Error    string  `json:"error"`
	key      string
}

// Client embeds texts; tests replace it.
var Client = &http.Client{Timeout: 60 * time.Second}

// Load returns the person's settings with the key opened, or nil.
func Load(ctx context.Context, q db.Querier, sealer *auth.Sealer, userID uuid.UUID) (*Settings, error) {
	var s Settings
	var sealed []byte
	err := q.QueryRow(ctx, `SELECT provider, base_url, model, api_key_sealed, error FROM embedding_settings WHERE user_id = $1`, userID).
		Scan(&s.Provider, &s.BaseURL, &s.Model, &sealed, &s.Error)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // not set up
	}
	if err != nil {
		return nil, err
	}
	if sealed != nil {
		s.HasKey = true
		if sealer != nil {
			k, err := sealer.Open(sealed)
			if err != nil {
				return nil, err
			}
			s.key = string(k)
		}
	}
	return &s, nil
}

// Input is what a person sends to set up the service.
type Input struct {
	Provider string  `json:"provider"`
	BaseURL  *string `json:"base_url"`
	Model    string  `json:"model"`
	APIKey   *string `json:"api_key"` // absent: keep the stored key
}

// Save validates and stores the settings. Changing the service or the
// model drops the old vectors: they cannot be compared with new ones.
func Save(ctx context.Context, tx pgx.Tx, sealer *auth.Sealer, userID uuid.UUID, in Input) error {
	def, ok := providers[in.Provider]
	if !ok {
		return ValidationError{"The service is local, openai, or voyage."}
	}
	if in.Model == "" {
		in.Model = def.model
	}
	if len(in.Model) > 100 || strings.ContainsAny(in.Model, " \t\n") {
		return ValidationError{"A model name has no spaces, like nomic-embed-text."}
	}
	if in.Provider == "local" {
		if in.BaseURL == nil {
			return ValidationError{"A local model server needs its address, like http://ollama:11434."}
		}
		u, err := url.Parse(*in.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || len(*in.BaseURL) > 300 {
			return ValidationError{"A local model server's address looks like http://ollama:11434."}
		}
	} else {
		in.BaseURL = nil
	}
	var sealed []byte
	keep := in.APIKey == nil
	if in.APIKey != nil && *in.APIKey != "" {
		if sealer == nil {
			return ValidationError{"This Axon cannot keep keys: its operator has not set AXON_ENCRYPTION_KEY."}
		}
		sealed = sealer.Seal([]byte(strings.TrimSpace(*in.APIKey)))
	}
	if in.Provider != "local" && keep {
		var has bool
		if err := tx.QueryRow(ctx, `SELECT api_key_sealed IS NOT NULL AND provider = $2 FROM embedding_settings WHERE user_id = $1`, userID, in.Provider).Scan(&has); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if !has {
			return ValidationError{"Add your API key for this service."}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO embedding_settings (user_id, provider, base_url, model, api_key_sealed) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET provider = $2, base_url = $3, model = $4,
		api_key_sealed = CASE WHEN $6 THEN embedding_settings.api_key_sealed ELSE $5 END, error = '', updated_at = now()`,
		userID, in.Provider, in.BaseURL, in.Model, sealed, keep); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM note_embeddings WHERE user_id = $1 AND model <> $2`, userID, in.Provider+"/"+in.Model)
	return err
}

// Remove forgets the service and every stored vector.
func Remove(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM note_embeddings WHERE user_id = $1`, userID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM embedding_settings WHERE user_id = $1`, userID)
	return err
}

func init() {
	privacy.OnWithdraw(Purpose, func(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
		_, err := tx.Exec(ctx, `DELETE FROM note_embeddings WHERE user_id = $1`, userID)
		return err
	})
}

// modelKey names the vectors' space: the service and the model.
func (s *Settings) modelKey() string { return s.Provider + "/" + s.Model }

// Embed turns texts into vectors with the person's service. kind is
// "document" or "query" (Voyage embeds them differently).
func (s *Settings) Embed(ctx context.Context, texts []string, kind string) ([][]float32, error) {
	base := providers[s.Provider].base
	if s.BaseURL != nil {
		base = *s.BaseURL
	}
	body := map[string]any{"model": s.Model, "input": texts}
	if s.Provider == "voyage" {
		body["input_type"] = kind
	}
	b, _ := json.Marshal(body)
	// The address is the person's own model server (validated as http(s) when
	// saved), as for the agents' local provider, or a fixed cloud service.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/v1/embeddings", bytes.NewReader(b)) //nolint:gosec // see above
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.key != "" {
		req.Header.Set("Authorization", "Bearer "+s.key)
	}
	res, err := Client.Do(req) //nolint:gosec // see above
	if err != nil {
		return nil, fmt.Errorf("the embedding service could not be reached")
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the embedding service answered %d", res.StatusCode)
	}
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &out); err != nil || len(out.Data) != len(texts) {
		return nil, fmt.Errorf("the embedding service did not answer with %d embeddings", len(texts))
	}
	vecs := make([][]float32, len(texts))
	for i, d := range out.Data {
		at := d.Index
		if at < 0 || at >= len(vecs) {
			at = i
		}
		if len(d.Embedding) == 0 || len(d.Embedding) > 4096 {
			return nil, fmt.Errorf("the embedding service answered an embedding of %d numbers", len(d.Embedding))
		}
		vecs[at] = d.Embedding
	}
	return vecs, nil
}

// literal writes a vector as pgvector reads it.
func literal(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// Ready says whether the person can search by meaning: a service and the consent.
func Ready(ctx context.Context, q db.Querier, sealer *auth.Sealer, userID uuid.UUID) (*Settings, bool, error) {
	s, err := Load(ctx, q, sealer, userID)
	if err != nil || s == nil {
		return s, false, err
	}
	ok, err := privacy.Granted(ctx, q, userID, Purpose)
	return s, ok, err
}
