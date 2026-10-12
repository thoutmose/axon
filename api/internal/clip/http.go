package clip

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/thoutmose/axon/api/internal/httpx"
)

// Routes mounts POST /v1/clip; the caller wraps it in auth.Require.
func Routes(log *slog.Logger) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/v1/clip", func(w http.ResponseWriter, r *http.Request) {
			var b struct {
				URL string `json:"url"`
			}
			if err := httpx.Decode(r, &b); err != nil {
				httpx.Problem(w, http.StatusBadRequest, err.Error())
				return
			}
			p, err := Fetch(r.Context(), b.URL)
			var ep ErrPage
			switch {
			case errors.As(err, &ep):
				httpx.Problem(w, http.StatusUnprocessableEntity, ep.Msg)
			case err != nil:
				log.Error("clip failed", "err", err)
				httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
			default:
				httpx.JSON(w, http.StatusOK, p)
			}
		})
	}
}
