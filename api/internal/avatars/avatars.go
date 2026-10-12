// Package avatars stores profile pictures and draws the default one (EP-33).
// Uploads are decoded, cropped to a centred square, resized to 256×256, and
// re-encoded as PNG, so only pixels are kept: no EXIF, no location, and no
// way to smuggle a script inside an image.
package avatars

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // decoders for uploads
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Size is the stored edge length in pixels.
const Size = 256

const maxUpload = 5 << 20

// ErrNotImage is returned for files that are not PNG or JPEG pictures.
var ErrNotImage = errors.New("not a PNG or JPEG image")

// normalize decodes, crops, resizes, and re-encodes.
func normalize(data []byte) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotImage
	}
	if cfg.Width < 16 || cfg.Height < 16 || cfg.Width*cfg.Height > 40_000_000 {
		return nil, fmt.Errorf("%w: use a picture between 16×16 and 40 megapixels", ErrNotImage)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotImage
	}
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	crop := image.Rect(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2, 0, 0)
	crop.Max = crop.Min.Add(image.Pt(side, side))
	sq := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(sq, sq.Bounds(), src, crop.Min, draw.Src)
	out := resize(sq, Size)
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// resize scales a square image to n×n by averaging the source pixels that
// fall into each target pixel (a box filter: simple and good for shrinking).
func resize(src *image.RGBA, n int) *image.RGBA {
	side := src.Bounds().Dx()
	dst := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		y0, y1 := y*side/n, max((y+1)*side/n, y*side/n+1)
		for x := 0; x < n; x++ {
			x0, x1 := x*side/n, max((x+1)*side/n, x*side/n+1)
			var r, g, b, a, count uint64
			for sy := y0; sy < y1 && sy < side; sy++ {
				for sx := x0; sx < x1 && sx < side; sx++ {
					c := src.RGBAAt(sx, sy)
					r, g, b, a = r+uint64(c.R), g+uint64(c.G), b+uint64(c.B), a+uint64(c.A)
					count++
				}
			}
			if count == 0 {
				continue
			}
			// An average of 8-bit values is at most 255.
			dst.SetRGBA(x, y, color.RGBA{uint8(r / count), uint8(g / count), uint8(b / count), uint8(a / count)}) //nolint:gosec // see above
		}
	}
	return dst
}

// initials are up to two letters from a display name.
func initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		r := []rune(w)
		out = append(out, []rune(strings.ToUpper(string(r[0])))...)
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return string(out)
}

// Default draws the standard avatar: initials on a gradient whose colours
// come from the account id, so everyone gets a stable, distinct picture.
func Default(id uuid.UUID, name string) []byte {
	h := sha256.Sum256(id[:])
	hue1 := int(h[0]) * 360 / 256
	hue2 := (hue1 + 40 + int(h[1])%80) % 360
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" role="img" aria-label="%[1]s">`+
		`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="hsl(%[2]d 70%% 45%%)"/>`+
		`<stop offset="1" stop-color="hsl(%[3]d 70%% 35%%)"/></linearGradient></defs>`+
		`<rect width="256" height="256" fill="url(#g)"/>`+
		`<text x="128" y="128" dy="0.35em" text-anchor="middle" font-family="system-ui, sans-serif" font-size="104" font-weight="600" fill="#fff">%[1]s</text></svg>`,
		html.EscapeString(initials(name)), hue1, hue2))
}

// Handlers serves the avatar endpoints.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

// NewHandlers returns the avatar endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log}
}

// Routes mounts the endpoints; the caller wraps them in auth.RequireSession.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/me/avatar", h.get)
	r.With(auth.NotDemo).Delete("/v1/me/avatar", h.delete)
}

// UploadRoutes mounts the upload, which needs a larger body limit than JSON.
func (h *Handlers) UploadRoutes(r chi.Router) {
	r.With(auth.NotDemo).Put("/v1/me/avatar", h.put)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	h.log.Error("avatar request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var name string
	var pic []byte
	var updated *time.Time
	if err := h.db.QueryRow(r.Context(), `SELECT display_name, avatar, avatar_updated_at FROM users WHERE id = $1`, who.UserID).
		Scan(&name, &pic, &updated); err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	if pic == nil {
		w.Header().Set("Content-Type", "image/svg+xml")
		// The SVG is ours, but keep it inert even if opened directly.
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		_, _ = w.Write(Default(who.UserID, name))
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(pic)
}

func (h *Handlers) put(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	ct := r.Header.Get("Content-Type")
	if ct != "image/png" && ct != "image/jpeg" {
		httpx.Problem(w, http.StatusUnsupportedMediaType, "Upload a PNG or JPEG picture.")
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxUpload+1))
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "The picture could not be read.")
		return
	}
	if len(data) > maxUpload {
		httpx.Problem(w, http.StatusRequestEntityTooLarge, "Pictures are limited to 5 MB.")
		return
	}
	pic, err := normalize(data)
	if errors.Is(err, ErrNotImage) {
		httpx.Problem(w, http.StatusUnprocessableEntity, strings.ToUpper(err.Error()[:1])+err.Error()[1:]+".")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if _, err := h.db.Exec(r.Context(), `UPDATE users SET avatar = $2, avatar_updated_at = now() WHERE id = $1`, who.UserID, pic); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if _, err := h.db.Exec(r.Context(), `UPDATE users SET avatar = NULL, avatar_updated_at = now() WHERE id = $1`, who.UserID); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Files adds the uploaded picture to a personal data export.
func Files(ctx context.Context, q db.Querier, userID uuid.UUID) (map[string][]byte, error) {
	var pic []byte
	if err := q.QueryRow(ctx, `SELECT avatar FROM users WHERE id = $1`, userID).Scan(&pic); err != nil {
		return nil, err
	}
	if pic == nil {
		return map[string][]byte{}, nil
	}
	return map[string][]byte{"avatar.png": pic}, nil
}
