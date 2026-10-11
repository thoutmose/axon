package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
)

// Food catalogues. Open Food Facts (ODbL) needs no key but allows only 10
// searches and 15 product reads a minute per IP; USDA FoodData Central
// (public domain) needs USDA_API_KEY. Results are cached, so logging keeps
// working when both are down, and typed-in entries never need them.

const (
	lookupTTL = 30 * 24 * time.Hour
	maxBody   = 4 << 20
	// maxWait is how long a search may queue behind the providers' limits
	// before Axon says it is busy instead.
	maxWait = 3 * time.Second
)

// ErrUnavailable means a catalogue cannot be reached or is not set up; ErrBusy
// that its rate limit is used up for now.
var (
	ErrUnavailable = errors.New("food catalogue unavailable")
	ErrBusy        = errors.New("food catalogue busy")
)

// Food is a catalogue food, per 100 g.
type Food struct {
	ID       uuid.UUID `json:"id"`
	Source   string    `json:"source"`
	SourceID string    `json:"source_id"`
	Name     string    `json:"name"`
	Brand    string    `json:"brand"`
	Kcal     *float64  `json:"kcal"`
	Protein  *float64  `json:"protein_g"`
	Carbs    *float64  `json:"carbs_g"`
	Fat      *float64  `json:"fat_g"`
	Fiber    *float64  `json:"fiber_g"`
	ServingG *float64  `json:"serving_g"`
}

const foodColumns = `id, source, source_id, name, brand, kcal::float8, protein_g::float8, carbs_g::float8, fat_g::float8, fiber_g::float8, serving_g::float8`

func scanFood(r pgx.CollectableRow) (Food, error) {
	var f Food
	err := r.Scan(&f.ID, &f.Source, &f.SourceID, &f.Name, &f.Brand, &f.Kcal, &f.Protein, &f.Carbs, &f.Fat, &f.Fiber, &f.ServingG)
	return f, err
}

// GetFood returns a cached food.
func GetFood(ctx context.Context, q db.Querier, id uuid.UUID) (Food, error) {
	rows, err := q.Query(ctx, `SELECT `+foodColumns+` FROM foods WHERE id = $1`, id)
	if err != nil {
		return Food{}, err
	}
	f, err := pgx.CollectExactlyOneRow(rows, scanFood)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}

// Endpoints are the catalogues' base URLs.
type Endpoints struct{ OFFSearch, OFF, USDA string }

// DefaultEndpoints are the real catalogues; tests point them elsewhere.
var DefaultEndpoints = Endpoints{
	OFFSearch: "https://search.openfoodfacts.org", OFF: "https://world.openfoodfacts.org", USDA: "https://api.nal.usda.gov/fdc/v1",
}

// Catalogs looks foods up and caches them.
type Catalogs struct {
	Endpoints
	db        *pgxpool.Pool
	client    *http.Client
	userAgent string
	usdaKey   string

	mu   sync.Mutex
	next map[string]time.Time
}

// NewCatalogs returns the catalogues; contact goes in the User-Agent, as Open Food Facts asks.
func NewCatalogs(pool *pgxpool.Pool, version, contact, usdaKey string) *Catalogs {
	ua := "Axon/" + version
	if contact != "" {
		ua += " (" + contact + ")"
	}
	return &Catalogs{
		Endpoints: DefaultEndpoints, db: pool, userAgent: ua, usdaKey: strings.TrimSpace(usdaKey), next: map[string]time.Time{},
		client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

// USDAEnabled reports whether USDA FoodData Central can be searched.
func (c *Catalogs) USDAEnabled() bool { return c.usdaKey != "" }

// gaps keep each catalogue under its limit: OFF 10 searches and 15 product
// reads a minute; USDA 1,000 an hour.
var gaps = map[string]time.Duration{"off-search": 6 * time.Second, "off-product": 4 * time.Second, "usda": 4 * time.Second}

// slot reserves the next request to a catalogue, or says it is busy.
func (c *Catalogs) slot(ctx context.Context, bucket string) error {
	c.mu.Lock()
	now := time.Now()
	at := c.next[bucket]
	if at.Before(now) {
		at = now
	}
	if at.Sub(now) > maxWait {
		c.mu.Unlock()
		return ErrBusy
	}
	c.next[bucket] = at.Add(gaps[bucket])
	c.mu.Unlock()
	select {
	case <-time.After(time.Until(at)):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Catalogs) get(ctx context.Context, bucket, rawURL string, out any) error {
	if err := c.slot(ctx, bucket); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil) //nolint:gosec // fixed catalogue hosts; only validated query values vary
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req) //nolint:gosec // fixed catalogue hosts; redirects are not followed
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err) //nolint:errorlint // the cause is for logs; callers check ErrUnavailable
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return ErrBusy
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: %s answered %d", ErrUnavailable, bucket, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(out)
}

// Search finds foods by name in Open Food Facts, or in USDA when source is usda.
func (c *Catalogs) Search(ctx context.Context, source, query string) ([]Food, error) {
	query = strings.ToLower(strings.Join(strings.Fields(query), " "))
	if query == "" || len([]rune(query)) > 100 {
		return nil, ValidationError{"Search for 1 to 100 characters."}
	}
	switch source {
	case "", "off":
		source = "off"
	case "usda":
		if !c.USDAEnabled() {
			return nil, fmt.Errorf("%w: USDA needs a key on this server", ErrUnavailable)
		}
	default:
		return nil, ValidationError{"Source is off or usda."}
	}
	var ids []uuid.UUID
	err := c.db.QueryRow(ctx, `SELECT food_ids FROM food_lookups WHERE source = $1 AND query = $2 AND fetched_at > now() - $3::interval`,
		source, query, fmt.Sprintf("%d seconds", int(lookupTTL.Seconds()))).Scan(&ids)
	if err == nil {
		return c.foods(ctx, ids)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var found []Food
	if source == "off" {
		found, err = c.searchOFF(ctx, query)
	} else {
		found, err = c.searchUSDA(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	ids, err = c.store(ctx, found)
	if err != nil {
		return nil, err
	}
	if _, err := c.db.Exec(ctx, `INSERT INTO food_lookups (source, query, food_ids) VALUES ($1, $2, $3)
		ON CONFLICT (source, query) DO UPDATE SET food_ids = EXCLUDED.food_ids, fetched_at = now()`, source, query, ids); err != nil {
		return nil, err
	}
	return c.foods(ctx, ids)
}

// Barcode looks a packaged food up in Open Food Facts by its EAN or UPC.
func (c *Catalogs) Barcode(ctx context.Context, code string) (Food, error) {
	if len(code) < 8 || len(code) > 14 || strings.Trim(code, "0123456789") != "" {
		return Food{}, ValidationError{"A barcode is 8 to 14 digits."}
	}
	rows, err := c.db.Query(ctx, `SELECT `+foodColumns+` FROM foods WHERE source = 'off' AND source_id = $1 AND fetched_at > now() - $2::interval`,
		code, fmt.Sprintf("%d seconds", int(lookupTTL.Seconds())))
	if err != nil {
		return Food{}, err
	}
	if f, err := pgx.CollectExactlyOneRow(rows, scanFood); err == nil {
		return f, nil
	}
	var res struct {
		Status  int        `json:"status"`
		Product offProduct `json:"product"`
	}
	if err := c.get(ctx, "off-product", c.OFF+"/api/v2/product/"+code+"?fields=code,product_name,brands,nutriments,serving_quantity", &res); err != nil {
		return Food{}, err
	}
	f, ok := res.Product.food()
	if res.Status != 1 || !ok {
		return Food{}, ErrNotFound
	}
	ids, err := c.store(ctx, []Food{f})
	if err != nil {
		return Food{}, err
	}
	return GetFood(ctx, c.db, ids[0])
}

type offProduct struct {
	Code       string          `json:"code"`
	Name       string          `json:"product_name"`
	Brands     json.RawMessage `json:"brands"` // a string in the product API, a list in search
	Serving    json.RawMessage `json:"serving_quantity"`
	Nutriments map[string]any  `json:"nutriments"`
}

func (p offProduct) food() (Food, bool) {
	name := strings.TrimSpace(p.Name)
	if name == "" || p.Code == "" || len(p.Code) > 40 {
		return Food{}, false
	}
	f := Food{Source: "off", SourceID: p.Code, Name: cut(name, 300), Brand: cut(brandOf(p.Brands), 200)}
	n := func(key string, max float64) *float64 {
		v, ok := number(p.Nutriments[key])
		if !ok || v < 0 || v > max {
			return nil
		}
		return &v
	}
	f.Kcal, f.Protein, f.Carbs, f.Fat, f.Fiber = n("energy-kcal_100g", 1000), n("proteins_100g", 100), n("carbohydrates_100g", 100), n("fat_100g", 100), n("fiber_100g", 100)
	var serving any
	if json.Unmarshal(p.Serving, &serving) == nil {
		if v, ok := number(serving); ok && v > 0 && v <= 5000 {
			f.ServingG = &v
		}
	}
	return f, f.Kcal != nil
}

func brandOf(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		b, _, _ := strings.Cut(s, ",")
		return strings.TrimSpace(b)
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil && len(list) > 0 {
		return strings.TrimSpace(list[0])
	}
	return ""
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

func (c *Catalogs) searchOFF(ctx context.Context, query string) ([]Food, error) {
	var res struct {
		Hits []offProduct `json:"hits"`
	}
	q := url.Values{"q": {query}, "page_size": {"15"}, "fields": {"code,product_name,brands,nutriments,serving_quantity"}}
	if err := c.get(ctx, "off-search", c.OFFSearch+"/search?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	var out []Food
	for _, h := range res.Hits {
		if f, ok := h.food(); ok {
			out = append(out, f)
		}
	}
	return out, nil
}

func (c *Catalogs) searchUSDA(ctx context.Context, query string) ([]Food, error) {
	var res struct {
		Foods []struct {
			ID        int    `json:"fdcId"`
			Name      string `json:"description"`
			Brand     string `json:"brandOwner"`
			Nutrients []struct {
				Number string  `json:"nutrientNumber"`
				Unit   string  `json:"unitName"`
				Value  float64 `json:"value"`
			} `json:"foodNutrients"`
		} `json:"foods"`
	}
	q := url.Values{"api_key": {c.usdaKey}, "query": {query}, "pageSize": {"15"}, "dataType": {"Foundation,SR Legacy"}}
	if err := c.get(ctx, "usda", c.USDA+"/foods/search?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	var out []Food
	for _, u := range res.Foods {
		if u.ID <= 0 || strings.TrimSpace(u.Name) == "" {
			continue
		}
		f := Food{Source: "usda", SourceID: strconv.Itoa(u.ID), Name: cut(strings.TrimSpace(u.Name), 300), Brand: cut(u.Brand, 200)}
		for _, n := range u.Nutrients {
			v := n.Value
			if v < 0 {
				continue
			}
			// Nutrient numbers: 208 energy (kcal), 203 protein, 205 carbohydrate, 204 fat, 291 fibre; values are per 100 g.
			switch {
			case n.Number == "208" && strings.EqualFold(n.Unit, "KCAL") && v <= 1000:
				f.Kcal = &v
			case n.Number == "203" && v <= 100:
				f.Protein = &v
			case n.Number == "205" && v <= 100:
				f.Carbs = &v
			case n.Number == "204" && v <= 100:
				f.Fat = &v
			case n.Number == "291" && v <= 100:
				f.Fiber = &v
			}
		}
		if f.Kcal != nil {
			out = append(out, f)
		}
	}
	return out, nil
}

// store upserts foods and returns their ids in order.
func (c *Catalogs) store(ctx context.Context, found []Food) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	for _, f := range found {
		var id uuid.UUID
		if err := c.db.QueryRow(ctx, `INSERT INTO foods (id, source, source_id, name, brand, kcal, protein_g, carbs_g, fat_g, fiber_g, serving_g)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (source, source_id) DO UPDATE SET name = EXCLUDED.name, brand = EXCLUDED.brand, kcal = EXCLUDED.kcal,
				protein_g = EXCLUDED.protein_g, carbs_g = EXCLUDED.carbs_g, fat_g = EXCLUDED.fat_g, fiber_g = EXCLUDED.fiber_g,
				serving_g = EXCLUDED.serving_g, fetched_at = now()
			RETURNING id`, uuid.Must(uuid.NewV7()), f.Source, f.SourceID, f.Name, f.Brand, f.Kcal, f.Protein, f.Carbs, f.Fat, f.Fiber, f.ServingG).Scan(&id); err != nil {
			return nil, err
		}
		if !contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (c *Catalogs) foods(ctx context.Context, ids []uuid.UUID) ([]Food, error) {
	rows, err := c.db.Query(ctx, `SELECT `+foodColumns+` FROM foods WHERE id = ANY($1) ORDER BY array_position($1, id)`, ids)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, scanFood)
	if out == nil {
		out = []Food{}
	}
	return out, err
}

func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
