package cards

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/fsrs"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/images"
)

// Handlers serves cards, decks, and reviews.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the card endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/decks", h.decks)
	r.Get("/v1/flashcards", h.list)
	r.Post("/v1/flashcards", h.create)
	r.Get("/v1/flashcards/due", h.due)
	r.Get("/v1/flashcards/practice", h.practice)
	r.Get("/v1/flashcards/export.apkg", h.exportAPKG)
	r.Get("/v1/flashcards/{id}", h.get)
	r.Patch("/v1/flashcards/{id}", h.update)
	r.Delete("/v1/flashcards/{id}", h.delete)
	r.Post("/v1/flashcards/{id}/review", h.review)
	r.Get("/v1/flashcards/{id}/image", h.getImage)
	r.Delete("/v1/flashcards/{id}/image", h.deleteImage)
}

// UploadRoutes mounts the picture upload, which needs a larger body limit.
func (h *Handlers) UploadRoutes(r chi.Router) {
	r.Put("/v1/flashcards/{id}/image", h.putImage)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such card.")
	default:
		h.log.Error("cards request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func cardID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, graph.ErrNotFound
	}
	return id, nil
}

func (h *Handlers) tx(ctx context.Context, f func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, h.db, f)
}

func (h *Handlers) decks(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := Decks(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := List(r.Context(), h.db, who.UserID, r.URL.Query().Get("deck"))
	if err != nil {
		h.fail(w, err)
		return
	}
	if r.URL.Query().Get("leech") == "true" {
		items = slices.DeleteFunc(items, func(c Card) bool { return !c.Leech })
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// DueCard is a due card with what each rating would do.
type DueCard struct {
	Card
	Next Preview `json:"next"`
}

func (h *Handlers) due(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			httpx.Problem(w, http.StatusBadRequest, "?limit= is 1 to 500.")
			return
		}
		limit = n
	}
	now := h.now()
	list, err := Due(r.Context(), h.db, who.UserID, r.URL.Query().Get("deck"), now, limit)
	if err != nil {
		h.fail(w, err)
		return
	}
	total, err := DueCount(r.Context(), h.db, who.UserID, now)
	if err != nil {
		h.fail(w, err)
		return
	}
	prefs, err := LoadPrefs(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	items := make([]DueCard, len(list))
	for i, c := range list {
		items[i] = DueCard{c, Next(c, now, prefs)}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "due_total": total})
}

// practice returns cards for deliberate practice; rating them is up to the
// page and does not change their schedule.
func (h *Handlers) practice(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			httpx.Problem(w, http.StatusBadRequest, "?limit= is 1 to 100.")
			return
		}
		limit = n
	}
	items, err := Practice(r.Context(), h.db, who.UserID, h.now(), limit)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type body struct {
	Deck        *string    `json:"deck"`
	Front       *string    `json:"front"`
	Back        *string    `json:"back"`
	Elaboration *string    `json:"elaboration"`
	Example     *string    `json:"example"`
	Mnemonic    *string    `json:"mnemonic"`
	Suspended   *bool      `json:"suspended"`
	PalaceID    *string    `json:"palace_id"`
	Locus       *int       `json:"locus"`
	SourceID    *uuid.UUID `json:"source_id"`
}

func (b body) fields() Fields {
	return Fields{Deck: b.Deck, Front: b.Front, Back: b.Back, Elaboration: b.Elaboration, Example: b.Example, Mnemonic: b.Mnemonic,
		Suspended: b.Suspended, Palace: b.PalaceID, Locus: b.Locus}
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b body
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var c Card
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		c, err = Create(r.Context(), tx, who.UserID, graph.UserActor, b.fields(), b.SourceID, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	c, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var b body
	if err := httpx.Decode(r, &b); err != nil || b.SourceID != nil {
		httpx.Problem(w, http.StatusBadRequest, "Send the fields to change: deck, front, back, elaboration, example, mnemonic, suspended, palace_id, or locus.")
		return
	}
	var c Card
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		c, err = Update(r.Context(), tx, who.UserID, id, graph.UserActor, b.fields())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		if _, err := Get(r.Context(), tx, who.UserID, id); err != nil {
			return err
		}
		return graph.DeleteEntity(r.Context(), tx, who.UserID, id, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) review(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var b struct {
		Rating     int  `json:"rating"`
		DurationMs *int `json:"duration_ms"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var c Card
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		c, err = Review(r.Context(), tx, who.UserID, id, fsrs.Rating(b.Rating), b.DurationMs, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) getImage(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	img, ct, err := Image(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if img == nil {
		httpx.Problem(w, http.StatusNotFound, "This card has no picture.")
		return
	}
	// A drawing is sanitised SVG; keep it inert even when opened on its own.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(img)
}

func (h *Handlers) putImage(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "image/png" && ct != "image/jpeg" {
		httpx.Problem(w, http.StatusUnsupportedMediaType, "Upload a PNG or JPEG picture.")
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 10<<20+1))
	if err != nil || len(data) > 10<<20 {
		httpx.Problem(w, http.StatusRequestEntityTooLarge, "Pictures are limited to 10 MB.")
		return
	}
	png, err := images.Fit(data, 1200)
	if errors.Is(err, images.ErrNotImage) {
		msg := err.Error()
		httpx.Problem(w, http.StatusUnprocessableEntity, strings.ToUpper(msg[:1])+msg[1:]+".")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := SetImage(r.Context(), h.db, who.UserID, id, png); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) deleteImage(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := cardID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := SetImage(r.Context(), h.db, who.UserID, id, nil); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) exportAPKG(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	deck := r.URL.Query().Get("deck")
	list, err := List(r.Context(), h.db, who.UserID, deck)
	if err != nil {
		h.fail(w, err)
		return
	}
	if len(list) == 0 {
		httpx.Problem(w, http.StatusNotFound, "There are no cards to export.")
		return
	}
	data, err := APKG(r.Context(), list, func(id uuid.UUID) ([]byte, error) {
		img, _, err := Image(r.Context(), h.db, who.UserID, id)
		return img, err
	}, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	name := "axon-cards.apkg"
	if deck != "" {
		name = "axon-" + safeName(deck) + ".apkg"
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write(data)
}

func safeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// ExportReviews is the review history in a personal data export.
func ExportReviews(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	type row struct {
		CardID     uuid.UUID `json:"card_id"`
		Rating     int       `json:"rating"`
		ReviewedAt time.Time `json:"reviewed_at"`
		State      int       `json:"state"`
		DurationMs *int      `json:"duration_ms"`
	}
	rows, err := q.Query(ctx, `SELECT r.card_id, r.rating, r.reviewed_at, r.state, r.duration_ms FROM card_reviews r
		JOIN entities e ON e.id = r.card_id WHERE e.user_id = $1 ORDER BY r.reviewed_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[row])
}
