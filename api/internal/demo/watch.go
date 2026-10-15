package demo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/watch"
)

// The demo's tech watch (EP-110): topics on the roadmap's domains, a few
// sources, and their news, stored as a fetch would store them. The demo
// never fetches (anyone can use it), so the articles are written here:
// real, lasting pages, each summarised. Hacker News carries a story the Go
// blog has too, to show a story kept once with the sources that carried it.

type demoArticle struct {
	hoursAgo int
	url      string
	title    string
	author   string
	summary  string
}

type demoSource struct {
	url, site, title string
	articles         []demoArticle
}

var demoTopics = []watch.Topic{
	{Name: "Go and Rust", Keywords: []string{"golang", "goroutine", "closure", "compiler", "inlining", "standard library", "rust", "cargo"}},
	{Name: "Databases", Keywords: []string{"postgresql", "sqlite", "sql", "mvcc", "query plan", "index"}},
	{Name: "Distributed systems", Keywords: []string{"raft", "consensus", "microservices", "replication", "leader"}},
	{Name: "Security", Keywords: []string{"owasp", "security", "vulnerability", "access control"}},
	{Name: "Cloud native", Keywords: []string{"kubernetes", "pod", "container", "cluster"}},
}

var loopvar = demoArticle{3, "https://go.dev/blog/loopvar-preview", "Fixing For Loops in Go 1.22", "Russ Cox",
	"Go 1.22 gives each iteration of a for loop its own copy of the loop variable, so closures and goroutines started in the loop no longer share one variable by accident. Go 1.21 offers a preview behind GOEXPERIMENT=loopvar, and the change only applies to modules that declare Go 1.22 or later, so old code keeps its meaning."}

var demoSources = []demoSource{
	{"https://go.dev/blog/feed.atom", "https://go.dev/blog/", "The Go Blog", []demoArticle{
		loopvar,
		{30, "https://go.dev/blog/pgo", "Profile-guided optimization in Go 1.21", "Michael Pratt",
			"Profile-guided optimization is ready for production in Go 1.21. Put a CPU profile from production in the main package as default.pgo, and the compiler inlines the hot calls more aggressively and devirtualises interface calls. Typical programs get a few percent faster with no change to their code."},
		{100, "https://go.dev/blog/slog", "Structured Logging with slog", "Jonathan Amsterdam",
			"Go 1.21 adds log/slog to the standard library: structured logs as a message, a level, and key-value pairs, written by a handler as text or JSON. Attributes can be grouped, handlers can be swapped or wrapped, and the API is designed to be fast enough for hot paths."},
	}},
	{"https://blog.rust-lang.org/feed.xml", "https://blog.rust-lang.org/", "Rust Blog", []demoArticle{
		{8, "https://blog.rust-lang.org/2024/10/17/Rust-1.82.0.html", "Announcing Rust 1.82.0", "The Rust Release Team",
			"Rust 1.82 adds cargo info to describe a package from the registry, precise capturing with use&lt;..&gt; bounds on impl Trait, a native syntax for raw pointers with &raw const and &raw mut, and safe items inside unsafe extern blocks. The compiler and standard library gain many smaller fixes."},
		{150, "https://blog.rust-lang.org/2023/12/28/Rust-1.75.0.html", "Announcing Rust 1.75.0", "The Rust Release Team",
			"Rust 1.75 stabilises async fn and return-position impl Trait in traits, a long-awaited step for asynchronous code: a trait can now declare async methods directly. The release also brings pointer byte offset APIs and code layout optimisations for the compiler itself."},
	}},
	{"https://kubernetes.io/feed.xml", "https://kubernetes.io/", "Kubernetes", []demoArticle{
		{20, "https://kubernetes.io/docs/concepts/security/pod-security-standards/", "Pod Security Standards", "Kubernetes",
			"Kubernetes defines three policies for pods: Privileged, unrestricted; Baseline, which blocks the known privilege escalations; and Restricted, which follows current hardening practice. A cluster applies them per namespace with the built-in Pod Security admission controller, to enforce, audit, or only warn."},
	}},
	{"https://www.postgresql.org/news.rss", "https://www.postgresql.org/", "PostgreSQL", []demoArticle{
		{12, "https://www.postgresql.org/docs/current/mvcc-intro.html", "Concurrency control: an introduction to MVCC", "The PostgreSQL Global Development Group",
			"PostgreSQL keeps data consistent with multiversion concurrency control: each statement sees a snapshot of the data as it was some time ago, whatever other transactions are doing. Reading never blocks writing and writing never blocks reading, which is why the database scales well under concurrent load."},
		{75, "https://www.postgresql.org/docs/current/using-explain.html", "Using EXPLAIN", "The PostgreSQL Global Development Group",
			"EXPLAIN shows the plan the query planner chose for a statement: the scans, joins, and sorts, with estimated costs and row counts. EXPLAIN ANALYZE also runs the query and reports actual times and rows, so wrong estimates, missing indexes, and slow nodes become visible."},
	}},
	{"https://hnrss.org/frontpage", "https://news.ycombinator.com/", "Hacker News", []demoArticle{
		loopvar,
		{5, "https://raft.github.io/", "The Raft Consensus Algorithm", "Diego Ongaro and John Ousterhout",
			"Raft is a consensus algorithm designed to be easy to understand, and equivalent to Paxos in fault tolerance and performance. It separates leader election, log replication, and safety, and lets a cluster of servers agree on a replicated log even when some of them fail."},
		{16, "https://owasp.org/www-project-top-ten/", "OWASP Top Ten", "OWASP Foundation",
			"The OWASP Top 10 is the standard awareness document on the most critical security risks of web applications, built from data and a community survey. Broken access control leads the 2021 edition, followed by cryptographic failures and injection; each risk comes with ways to prevent it."},
		{50, "https://www.sqlite.org/whentouse.html", "Appropriate Uses For SQLite", "SQLite",
			"SQLite does not compete with client/server databases; it competes with fopen(). The page lists where an embedded database shines, from devices and application file formats to websites with modest traffic and data analysis, and where a client/server database such as PostgreSQL is the better choice."},
		{125, "https://martinfowler.com/articles/microservices.html", "Microservices", "James Lewis and Martin Fowler",
			"The article that named the microservices style: an application built as a suite of small services, each running in its own process, deployed independently, and organised around business capabilities. It weighs smart endpoints and dumb pipes, decentralised data, and infrastructure automation against the cost of distribution."},
	}},
}

func seedWatch(ctx context.Context, tx pgx.Tx, user uuid.UUID, now time.Time) error {
	for _, t := range demoTopics {
		if _, err := watch.SaveTopic(ctx, tx, user, "", t.Name, t.Keywords); err != nil {
			return err
		}
	}
	for _, s := range demoSources {
		f := watch.Found{URL: s.url, Feed: watch.Feed{Title: s.title, SiteURL: s.site}}
		for _, a := range s.articles {
			f.Feed.Items = append(f.Feed.Items, watch.Entry{GUID: a.url, URL: a.url, Title: a.title, Author: a.author,
				Published: now.Add(-time.Duration(a.hoursAgo) * time.Hour), Summary: a.summary})
		}
		if _, _, err := watch.CreateSource(ctx, tx, user, watch.SourceInput{Origin: "manual"}, f, now); err != nil {
			return err
		}
	}
	// Some news read, one saved, one kept as a note.
	item := func(url string) (id uuid.UUID, err error) {
		err = tx.QueryRow(ctx, `SELECT id FROM watch_items WHERE user_id = $1 AND url = $2`, user, url).Scan(&id)
		return id, err
	}
	for url, status := range map[string]string{
		"https://go.dev/blog/slog": "read", "https://blog.rust-lang.org/2023/12/28/Rust-1.75.0.html": "read",
		"https://martinfowler.com/articles/microservices.html": "read", "https://raft.github.io/": "saved",
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
