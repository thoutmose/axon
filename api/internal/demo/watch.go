package demo

import (
	"context"
	_ "embed"
	"encoding/json"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/watch"
)

// The demo's tech watch (EP-110, EP-111): topics on the roadmap's domains,
// a few sources, and a month of their news, stored as a fetch would store
// them, so the list, the knowledge graph, the digest, and the podcast all
// have something to show. The demo never fetches (anyone can use it), so
// the articles are in watch.json: real, lasting pages, each described in a
// few paragraphs. The texts of a domain share its concepts, and some
// articles bridge two domains (etcd runs Raft for Kubernetes), so the graph
// has communities and links between them. Hacker News carries a story the
// Go blog has too, to show a story kept once with the sources that carried it.

//go:embed watch.json
var watchJSON []byte

type demoWatch struct {
	Topics  []watch.Topic `json:"topics"`
	Sources []struct {
		URL      string `json:"url"`
		Site     string `json:"site"`
		Title    string `json:"title"`
		Articles []struct {
			Hours  int    `json:"hours"` // published this many hours ago
			URL    string `json:"url"`
			Title  string `json:"title"`
			Author string `json:"author"`
			Text   string `json:"text"` // paragraphs, separated by a blank line
		} `json:"articles"`
	} `json:"sources"`
}

// asHTML gives plain paragraphs as a feed gives an article.
func asHTML(text string) string {
	var b strings.Builder
	for _, p := range strings.Split(text, "\n\n") {
		b.WriteString("<p>" + html.EscapeString(p) + "</p>\n")
	}
	return b.String()
}

func seedWatch(ctx context.Context, tx pgx.Tx, user uuid.UUID, now time.Time) error {
	var w demoWatch
	if err := json.Unmarshal(watchJSON, &w); err != nil {
		return err
	}
	for _, t := range w.Topics {
		if _, err := watch.SaveTopic(ctx, tx, user, "", t.Name, t.Keywords); err != nil {
			return err
		}
	}
	for _, s := range w.Sources {
		f := watch.Found{URL: s.URL, Feed: watch.Feed{Title: s.Title, SiteURL: s.Site}}
		for _, a := range s.Articles {
			f.Feed.Items = append(f.Feed.Items, watch.Entry{GUID: a.URL, URL: a.URL, Title: a.Title, Author: a.Author,
				Published: now.Add(-time.Duration(a.Hours) * time.Hour), Content: asHTML(a.Text)})
		}
		if _, _, err := watch.CreateSource(ctx, tx, user, watch.SourceInput{Origin: "manual"}, f, now); err != nil {
			return err
		}
	}
	// Some news read, some saved, one kept as a note.
	item := func(url string) (id uuid.UUID, err error) {
		err = tx.QueryRow(ctx, `SELECT id FROM watch_items WHERE user_id = $1 AND url_hash = $2`, user, watch.URLHash(url)).Scan(&id)
		return id, err
	}
	for url, status := range map[string]string{
		"https://go.dev/blog/slog": "read", "https://blog.rust-lang.org/2023/12/28/Rust-1.75.0.html": "read",
		"https://martinfowler.com/articles/microservices.html": "read", "https://go.dev/blog/range-functions": "read",
		"https://www.postgresql.org/docs/current/wal-intro.html": "read", "https://raft.github.io/": "saved",
		"https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html": "saved",
	} {
		id, err := item(url)
		if err == nil {
			err = watch.UpdateItem(ctx, tx, user, id, &status, nil)
		}
		if err != nil {
			return err
		}
	}
	id, err := item("https://www.postgresql.org/docs/current/mvcc-intro.html")
	if err == nil {
		_, err = watch.Keep(ctx, tx, user, id)
	}
	return err
}
