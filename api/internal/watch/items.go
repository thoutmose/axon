package watch

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notes"
)

// Item is a news item as a feed note.
type Item struct {
	ID          uuid.UUID  `json:"id"`
	SourceID    *uuid.UUID `json:"source_id"`
	SourceTitle string     `json:"source_title"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	Author      string     `json:"author"`
	PublishedAt time.Time  `json:"published_at"`
	FetchedAt   time.Time  `json:"fetched_at"`
	Summary     string     `json:"summary"`
	Content     *string    `json:"content,omitempty"` // only for one item
	Image       string     `json:"image"`
	Audio       string     `json:"audio"`
	Lang        string     `json:"lang"`
	Words       int        `json:"words"`
	Minutes     int        `json:"reading_minutes"`
	Keywords    []string   `json:"keywords"`
	Labels      []string   `json:"labels"`
	Categories  []string   `json:"categories"`
	TopicIDs    []string   `json:"topic_ids"`
	Score       float64    `json:"score"`
	Outlier     string     `json:"outlier"`
	AlsoIn      []string   `json:"also_in"`
	Status      string     `json:"status"`
	NoteID      *uuid.UUID `json:"note_id"`
	Kind        string     `json:"kind"` // article, video, podcast
}

// Kinds of news, for filters and exports.
var Kinds = []string{"article", "video", "podcast"}

// videoURL matches the video sites whose links are videos.
const videoURL = `^https?://(www\.|m\.)?(youtube\.com|youtu\.be|vimeo\.com)/`

var videoLink = regexp.MustCompile(`(?i)` + videoURL)

// kindOf tells an article from a video or a podcast episode.
func kindOf(url, audio string) string {
	switch {
	case audio != "":
		return "podcast"
	case videoLink.MatchString(url):
		return "video"
	}
	return "article"
}

const itemColumns = `i.id, i.source_id, i.source_title, i.url, i.title, i.author, i.published_at, i.fetched_at, i.summary, i.image, i.audio,
	i.lang, i.words, i.keywords, i.labels, i.categories, i.topic_ids::text[], i.score, i.outlier, i.also_in, i.status, i.note_id`

func scanItem(r pgx.Row, content *string) (Item, error) {
	var it Item
	var score float32
	dst := []any{&it.ID, &it.SourceID, &it.SourceTitle, &it.URL, &it.Title, &it.Author, &it.PublishedAt, &it.FetchedAt, &it.Summary, &it.Image, &it.Audio,
		&it.Lang, &it.Words, &it.Keywords, &it.Labels, &it.Categories, &it.TopicIDs, &score, &it.Outlier, &it.AlsoIn, &it.Status, &it.NoteID}
	if content != nil {
		dst = append(dst, content)
	}
	err := r.Scan(dst...)
	it.Score = float64(score)
	it.Kind = kindOf(it.URL, it.Audio)
	// About 230 words a minute.
	it.Minutes = max(1, (it.Words+115)/230)
	return it, err
}

// Filter selects news.
type Filter struct {
	Status   string     // unread, read, saved, archived; empty for all but archived
	SourceID *uuid.UUID //
	TopicID  string     //
	Label    string     //
	Query    string     // words in the title or summary
	From, To *time.Time // published
	Outliers bool       // include items flagged as outliers
	Kind     string     // article, video, podcast
	Limit    int
	Before   *time.Time // published before (paging)
}

func (f Filter) where(userID uuid.UUID) (string, []any) {
	conds := []string{"i.user_id = $1"}
	args := []any{userID}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	switch f.Status {
	case "":
		conds = append(conds, "i.status <> 'archived'")
	case "all":
	default:
		add("i.status = ?", f.Status)
	}
	if f.SourceID != nil {
		add("i.source_id = ?", *f.SourceID)
	}
	if f.TopicID != "" {
		add("?::uuid = ANY (i.topic_ids)", f.TopicID)
	}
	if f.Label != "" {
		add("? = ANY (i.labels)", strings.ToLower(f.Label))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
		args = append(args, like)
		conds = append(conds, fmt.Sprintf("(i.title ILIKE $%d OR i.summary ILIKE $%d)", len(args), len(args)))
	}
	if f.From != nil {
		add("i.published_at >= ?", *f.From)
	}
	if f.To != nil {
		add("i.published_at < ?", *f.To)
	}
	if f.Before != nil {
		add("i.published_at < ?", *f.Before)
	}
	switch f.Kind {
	case "podcast":
		conds = append(conds, "i.audio <> ''")
	case "video":
		conds = append(conds, "i.audio = '' AND i.url ~* '"+videoURL+"'")
	case "article":
		conds = append(conds, "i.audio = '' AND i.url !~* '"+videoURL+"'")
	}
	if !f.Outliers {
		conds = append(conds, "i.outlier = ''")
	}
	return strings.Join(conds, " AND "), args
}

// ListItems returns news, newest first.
func ListItems(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter) ([]Item, error) {
	where, args := f.where(userID)
	limit := ""
	switch {
	case f.Limit < 0: // everything: exports
	case f.Limit == 0:
		limit = " LIMIT 50"
	default:
		limit = fmt.Sprintf(" LIMIT %d", f.Limit)
	}
	rows, err := q.Query(ctx, `SELECT `+itemColumns+` FROM watch_items i WHERE `+where+` ORDER BY i.published_at DESC, i.id DESC`+limit, args...)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) { return scanItem(r, nil) })
	if out == nil {
		out = []Item{}
	}
	return out, err
}

// GetItem returns one item with its text.
func GetItem(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Item, error) {
	var content string
	it, err := scanItem(q.QueryRow(ctx, `SELECT `+itemColumns+`, i.content FROM watch_items i WHERE i.id = $1 AND i.user_id = $2`, id, userID), &content)
	if errors.Is(err, pgx.ErrNoRows) {
		return it, ErrNotFound
	}
	it.Content = &content
	return it, err
}

// Statuses an item can have.
var Statuses = []string{"unread", "read", "saved", "archived"}

// UpdateItem sets an item's status and the labels the person gives it.
func UpdateItem(ctx context.Context, q db.Querier, userID, id uuid.UUID, status *string, labels []string) error {
	if status != nil && !slices.Contains(Statuses, *status) {
		return ValidationError{"The status is unread, read, saved, or archived."}
	}
	var clean []string
	for _, l := range labels {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" || utf8.RuneCountInString(l) > 60 {
			return ValidationError{"A label has 1 to 60 characters."}
		}
		if !slices.Contains(clean, l) {
			clean = append(clean, l)
		}
	}
	if len(clean) > 20 {
		return ValidationError{"An item has up to 20 labels."}
	}
	tag, err := q.Exec(ctx, `UPDATE watch_items SET status = coalesce($3, status), labels = CASE WHEN $4::text[] IS NULL THEN labels ELSE $4 END
		WHERE id = $1 AND user_id = $2`, id, userID, status, clean)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// MarkAll marks every unread item matching f as read.
func MarkAll(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter) (int64, error) {
	f.Status = "unread"
	where, args := f.where(userID)
	tag, err := q.Exec(ctx, `UPDATE watch_items i SET status = 'read' WHERE `+where, args...)
	return tag.RowsAffected(), err
}

// FeedNote is an item as a Markdown note, the feed-note template: front
// matter for its fields, the summary, key points, the article, and a free
// section for the person's own thoughts.
func FeedNote(it Item, topics map[string]string) string {
	var b strings.Builder
	q := func(s string) string {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", " ").Replace(s) + `"`
	}
	b.WriteString("---\ntype: feed\n")
	fmt.Fprintf(&b, "title: %s\nsource: %s\nurl: %s\n", q(it.Title), q(it.SourceTitle), it.URL)
	if it.Author != "" {
		fmt.Fprintf(&b, "author: %s\n", q(it.Author))
	}
	fmt.Fprintf(&b, "published: %s\n", it.PublishedAt.Format(time.DateOnly))
	var tn []string
	for _, id := range it.TopicIDs {
		if n, ok := topics[id]; ok {
			tn = append(tn, q(n))
		}
	}
	fmt.Fprintf(&b, "topics: [%s]\n", strings.Join(tn, ", "))
	var tags []string
	for _, l := range it.Labels {
		tags = append(tags, q(l))
	}
	fmt.Fprintf(&b, "labels: [%s]\nreading_minutes: %d\n---\n# %s\n\n", strings.Join(tags, ", "), it.Minutes, it.Title)
	if it.Summary != "" {
		fmt.Fprintf(&b, "> %s\n\n", strings.ReplaceAll(it.Summary, "\n", "\n> "))
	}
	if len(it.Keywords) > 0 {
		b.WriteString("## Key points\n")
		for _, k := range it.Keywords {
			fmt.Fprintf(&b, "- %s\n", k)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Why it matters\n\n\n")
	if it.Content != nil && strings.TrimSpace(*it.Content) != "" {
		fmt.Fprintf(&b, "## Article\n%s\n\n", strings.TrimSpace(*it.Content))
	}
	b.WriteString("## My notes\n\n\n")
	fmt.Fprintf(&b, "---\nFrom [%s](%s)", it.SourceTitle, it.URL)
	if len(it.AlsoIn) > 0 {
		fmt.Fprintf(&b, ", also in %s", strings.Join(it.AlsoIn, ", "))
	}
	b.WriteString(" · kept with Axon's tech watch\n")
	return b.String()
}

// Keep makes the item a note in Feeds/, linked to it, and saves the item.
func Keep(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) (uuid.UUID, error) {
	it, err := GetItem(ctx, tx, userID, id)
	if err != nil {
		return uuid.Nil, err
	}
	if it.NoteID != nil {
		return *it.NoteID, nil
	}
	topics, err := ListTopics(ctx, tx, userID)
	if err != nil {
		return uuid.Nil, err
	}
	names := map[string]string{}
	for _, t := range topics {
		names[t.ID] = t.Name
	}
	base := fmt.Sprintf("Feeds/%s/%s", it.PublishedAt.Format("2006/01"), notes.Slug(it.Title))
	path := base + ".md"
	for i := 2; ; i++ {
		// Look before creating: a failed insert would end the transaction.
		if _, err := notes.GetByPath(ctx, tx, userID, path); errors.Is(err, graph.ErrNotFound) {
			break
		} else if err != nil {
			return uuid.Nil, err
		}
		path = fmt.Sprintf("%s-%d.md", base, i)
	}
	n, err := notes.Create(ctx, tx, userID, graph.UserActor, path, FeedNote(it, names))
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := graph.CreateLink(ctx, tx, userID, graph.UserActor, n.ID, graph.RelatesTo, id); err != nil {
		return uuid.Nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE watch_items SET note_id = $2, status = 'saved' WHERE id = $1`, id, n.ID)
	return n.ID, err
}

// Purge forgets news older than the retention that the person did not
// save, keep as a note, or link to anything.
func Purge(ctx context.Context, q db.Querier, olderThan time.Time) (int64, error) {
	tag, err := q.Exec(ctx, `DELETE FROM entities e USING watch_items i WHERE e.id = i.id AND i.fetched_at < $1
		AND i.status <> 'saved' AND i.note_id IS NULL
		AND NOT EXISTS (SELECT 1 FROM links l WHERE l.source_id = i.id OR l.target_id = i.id)`, olderThan)
	if err != nil {
		return 0, err
	}
	_, err = q.Exec(ctx, `DELETE FROM watch_seen WHERE seen_at < $1`, olderThan.Add(-30*24*time.Hour))
	return tag.RowsAffected(), err
}

// exportItems is the tech watch part of a personal export: topics, sources
// (with their schedule), news without their full text, and share links
// (never their tokens, which are not kept).
func exportItems(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	topics, err := ListTopics(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	sources, err := ListSources(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	items, err := ListItems(ctx, q, userID, Filter{Status: "all", Outliers: true, Limit: -1})
	if err != nil {
		return nil, err
	}
	shares, err := ListShares(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"topics": topics, "sources": sources, "items": items, "shares": shares}, nil
}
