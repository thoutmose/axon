// Package idem makes POST requests safe to retry with an Idempotency-Key header.
package idem

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Middleware stores the first response for each (user, key) for 24 hours and
// replays it for retries. Reusing a key for a different request is rejected.
// It must run after auth.Require; requests without the header pass through.
func Middleware(pool *pgxpool.Pool, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			id, signedIn := auth.FromContext(r.Context())
			if r.Method != http.MethodPost || key == "" || !signedIn {
				next.ServeHTTP(w, r)
				return
			}
			if len(key) > 200 {
				httpx.Problem(w, http.StatusBadRequest, "Idempotency-Key must be at most 200 characters.")
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, httpx.MaxBody))
			if err != nil {
				httpx.Problem(w, http.StatusBadRequest, "The request body could not be read.")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			sum := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), body...))
			ctx := r.Context()

			_, _ = pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE created_at < now() - interval '24 hours'`)
			tag, err := pool.Exec(ctx,
				`INSERT INTO idempotency_keys (user_id, key, request_hash) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
				id.UserID, key, sum[:])
			if err != nil {
				log.Error("idempotency insert failed", "err", err)
				httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
				return
			}
			if tag.RowsAffected() == 0 {
				replay(w, r, pool, log, id, key, sum[:])
				return
			}

			rec := &recorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status >= 500 {
				// Let the client retry a failure instead of replaying it.
				_, _ = pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE user_id = $1 AND key = $2`, id.UserID, key)
				return
			}
			if _, err := pool.Exec(ctx, `UPDATE idempotency_keys SET status = $3, body = $4 WHERE user_id = $1 AND key = $2`,
				id.UserID, key, rec.status, rec.body.Bytes()); err != nil {
				log.Error("idempotency store failed", "err", err)
			}
		})
	}
}

func replay(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, log *slog.Logger, id auth.Identity, key string, sum []byte) {
	var hash, body []byte
	var status *int
	err := pool.QueryRow(r.Context(), `SELECT request_hash, status, body FROM idempotency_keys WHERE user_id = $1 AND key = $2`,
		id.UserID, key).Scan(&hash, &status, &body)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		httpx.Problem(w, http.StatusConflict, "That request failed; retry it.")
	case err != nil:
		log.Error("idempotency lookup failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	case !bytes.Equal(hash, sum):
		httpx.Problem(w, http.StatusUnprocessableEntity, "This Idempotency-Key was already used for a different request.")
	case status == nil:
		httpx.Problem(w, http.StatusConflict, "A request with this Idempotency-Key is still being processed.")
	default:
		w.Header().Set("Idempotent-Replayed", "true")
		if len(body) > 0 {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(*status)
		_, _ = w.Write(body)
	}
}

type recorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *recorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *recorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
