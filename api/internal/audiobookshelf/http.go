package audiobookshelf

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/netsafe"
)

// Routes mounts the endpoints; the caller wraps them in auth.Require.
// Connecting and syncing reach outside Axon, so the demo cannot.
func (s *Store) Routes(r chi.Router) {
	r.Get("/v1/audiobookshelf", s.status)
	r.With(auth.NotDemo).Put("/v1/audiobookshelf", s.connect)
	r.Delete("/v1/audiobookshelf", s.disconnect)
	r.With(auth.NotDemo).Post("/v1/audiobookshelf/sync", s.sync)
}

func (s *Store) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNotConnected):
		httpx.Problem(w, http.StatusNotFound, "No Audiobookshelf server is connected.")
	case errors.Is(err, ErrToken):
		httpx.Problem(w, http.StatusUnprocessableEntity, "Audiobookshelf refused this API token.")
	case errors.Is(err, netsafe.ErrPrivate):
		httpx.Problem(w, http.StatusUnprocessableEntity, "That address is on a private network. The operator can allow it with AXON_CALENDAR_PRIVATE=1.")
	default:
		s.log.Warn("audiobookshelf request failed", "err", err)
		httpx.Problem(w, http.StatusBadGateway, "Audiobookshelf did not answer. Check the address, and that Axon can reach it.")
	}
}

func (s *Store) status(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	st, err := s.Get(r.Context(), who.UserID)
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (s *Store) connect(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var in struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	st, err := s.Connect(r.Context(), who.UserID, in.URL, in.Token, time.Now())
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (s *Store) disconnect(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := s.Disconnect(r.Context(), who.UserID); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) sync(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	out, err := s.Sync(r.Context(), who.UserID, time.Now())
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
