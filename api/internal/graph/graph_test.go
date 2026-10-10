package graph

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
)

// Integration tests: AXON_TEST_DATABASE_URL must point at a disposable database.

var pool *pgxpool.Pool

// Two throwaway types stand in for the domain types later Epics add.
func init() {
	export := func(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) { return []string{}, nil }
	RegisterType(Type{Name: "test_note", Table: "test_notes", Export: export})
	RegisterType(Type{Name: "test_goal", Table: "test_goals", Export: export})
	RegisterRelation(Relation{Name: "supports", Sources: []string{"test_note"}, Targets: []string{"test_goal"}})
}

func TestMain(m *testing.M) {
	dsn := os.Getenv("AXON_TEST_DATABASE_URL")
	if dsn == "" {
		os.Exit(0)
	}
	ctx := context.Background()
	if err := db.Migrate(ctx, dsn, "up", io.Discard); err != nil {
		panic(err)
	}
	var err error
	if pool, err = pgxpool.New(ctx, dsn); err != nil {
		panic(err)
	}
	for _, t := range []string{"test_notes", "test_goals"} {
		if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+t+` (id uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE, body text NOT NULL DEFAULT '')`); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	// Leave the schema as migrations define it, so other packages can migrate down.
	_, _ = pool.Exec(ctx, `DROP TABLE test_notes, test_goals`)
	pool.Close()
	os.Exit(code)
}

func newUser(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(context.Background(), `INSERT INTO users (id, display_name) VALUES ($1, 'Test')`, id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id) })
	return id
}

func create(t *testing.T, userID uuid.UUID, typ, title, body string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		var err error
		if id, err = CreateEntity(context.Background(), tx, userID, UserActor, typ, title, body); err != nil {
			return err
		}
		table := map[string]string{"test_note": "test_notes", "test_goal": "test_goals"}[typ]
		_, err = tx.Exec(context.Background(), `INSERT INTO `+table+` (id, body) VALUES ($1, $2)`, id, body)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func inTx(t *testing.T, f func(tx pgx.Tx) error) error {
	t.Helper()
	return pgx.BeginFunc(context.Background(), pool, f)
}

func TestLinksFollowRelationRules(t *testing.T) {
	ctx := context.Background()
	u := newUser(t)
	note := create(t, u, "test_note", "Spaced repetition", "")
	goal := create(t, u, "test_goal", "Learn Japanese", "")
	other := create(t, u, "test_note", "Other note", "")

	link := func(src uuid.UUID, rel string, dst uuid.UUID) error {
		return inTx(t, func(tx pgx.Tx) error { _, err := CreateLink(ctx, tx, u, UserActor, src, rel, dst); return err })
	}
	if err := link(note, "supports", goal); err != nil {
		t.Fatalf("note supports goal: %v", err)
	}
	if err := link(goal, "supports", note); !errors.Is(err, ErrRelationRejected) {
		t.Fatalf("goal supports note = %v, want ErrRelationRejected", err)
	}
	if err := link(note, "invented", other); !errors.Is(err, ErrRelationRejected) {
		t.Fatalf("unknown relation = %v, want ErrRelationRejected", err)
	}
	if err := link(note, RelatesTo, note); !errors.Is(err, ErrRelationRejected) {
		t.Fatalf("self link = %v, want ErrRelationRejected", err)
	}
	if err := link(note, RelatesTo, other); err != nil {
		t.Fatalf("relates_to: %v", err)
	}
	if err := link(note, RelatesTo, other); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate = %v, want ErrDuplicate", err)
	}

	out, _ := Outgoing(ctx, pool, u, note, Page{})
	in, _ := Incoming(ctx, pool, u, goal, Page{})
	if len(out) != 2 || len(in) != 1 || in[0].Other.ID != note {
		t.Fatalf("outgoing %d, incoming %v", len(out), in)
	}
}

func TestLinksCannotCrossOwners(t *testing.T) {
	ctx := context.Background()
	alice, bob := newUser(t), newUser(t)
	a := create(t, alice, "test_note", "Alice's", "")
	b := create(t, bob, "test_note", "Bob's", "")

	err := inTx(t, func(tx pgx.Tx) error { _, err := CreateLink(ctx, tx, alice, UserActor, a, RelatesTo, b); return err })
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner link = %v, want ErrNotFound", err)
	}
	// Even bypassing the service, the schema refuses it.
	_, err = pool.Exec(ctx, `INSERT INTO links (id, user_id, source_id, relation, target_id) VALUES ($1, $2, $3, 'relates_to', $4)`,
		uuid.Must(uuid.NewV7()), alice, a, b)
	if err == nil {
		t.Fatal("database accepted a link to another user's entity")
	}
}

func TestDeletingAnEntityRemovesItsLinksAndDomainRow(t *testing.T) {
	ctx := context.Background()
	u := newUser(t)
	note := create(t, u, "test_note", "Note", "")
	goal := create(t, u, "test_goal", "Goal", "")
	if err := inTx(t, func(tx pgx.Tx) error { _, err := CreateLink(ctx, tx, u, UserActor, note, "supports", goal); return err }); err != nil {
		t.Fatal(err)
	}
	if err := inTx(t, func(tx pgx.Tx) error { return DeleteEntity(ctx, tx, u, goal, UserActor) }); err != nil {
		t.Fatal(err)
	}
	var links, rows int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM links WHERE source_id = $1 OR target_id = $1`, goal).Scan(&links)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM test_goals WHERE id = $1`, goal).Scan(&rows)
	if links != 0 || rows != 0 {
		t.Fatalf("after delete: %d links, %d domain rows", links, rows)
	}
	problems, err := CheckIntegrity(ctx, pool)
	if err != nil || len(problems) != 0 {
		t.Fatalf("integrity: %v %v", problems, err)
	}
	// The timeline keeps the deletion, with field names only.
	events, _ := Timeline(ctx, pool, u, goal, nil, 10)
	if len(events) == 0 || events[0].Kind != "deleted" {
		t.Fatalf("timeline = %+v", events)
	}
}

func TestIntegrityCheckFindsOrphans(t *testing.T) {
	ctx := context.Background()
	u := newUser(t)
	orphan := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(ctx, `INSERT INTO entities (id, user_id, type, title) VALUES ($1, $2, 'test_note', 'no domain row')`, orphan, u); err != nil {
		t.Fatal(err)
	}
	problems, err := CheckIntegrity(ctx, pool)
	if err != nil || len(problems) == 0 {
		t.Fatalf("orphan not detected: %v %v", problems, err)
	}
}

func TestEventsRecordFieldNamesNotValues(t *testing.T) {
	ctx := context.Background()
	u := newUser(t)
	note := create(t, u, "test_note", "Diary", "private thoughts")
	err := inTx(t, func(tx pgx.Tx) error {
		return UpdateEntity(ctx, tx, u, note, UserActor, "Diary", "even more private thoughts", []string{"body"})
	})
	if err != nil {
		t.Fatal(err)
	}
	var leaked bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM events WHERE entity_id = $1 AND array_to_string(fields, ' ') LIKE '%private%')`, note).Scan(&leaked)
	if leaked {
		t.Fatal("event stored content")
	}
	events, _ := Timeline(ctx, pool, u, note, nil, 10)
	if len(events) != 2 || events[0].Kind != "updated" || events[0].Fields[0] != "body" {
		t.Fatalf("timeline = %+v", events)
	}
	if _, err := pool.Exec(ctx, `UPDATE events SET kind = 'created'`); err == nil {
		t.Fatal("events accepted an UPDATE")
	}
}

func TestSearchIsScopedToOwnerAndLanguageNeutral(t *testing.T) {
	ctx := context.Background()
	alice, bob := newUser(t), newUser(t)
	create(t, alice, "test_note", "Réviser le japonais", "kanji et vocabulaire")
	create(t, alice, "test_note", "Groceries", "eggs")
	create(t, bob, "test_note", "Bob's kanji list", "")

	hits, err := Search(ctx, pool, alice, "kanji", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Title != "Réviser le japonais" {
		t.Fatalf("hits = %+v", hits)
	}
	if hits, _ := Search(ctx, pool, alice, `japonais -kanji`, 0, 10); len(hits) != 0 {
		t.Fatalf("exclusion ignored: %+v", hits)
	}
}

func TestPaginationIsBoundedAndOrdered(t *testing.T) {
	ctx := context.Background()
	u := newUser(t)
	hub := create(t, u, "test_goal", "Hub", "")
	for range 5 {
		n := create(t, u, "test_note", "spoke", "")
		if err := inTx(t, func(tx pgx.Tx) error { _, err := CreateLink(ctx, tx, u, UserActor, n, "supports", hub); return err }); err != nil {
			t.Fatal(err)
		}
	}
	first, _ := Incoming(ctx, pool, u, hub, Page{Limit: 3})
	rest, _ := Incoming(ctx, pool, u, hub, Page{After: &first[2].ID, Limit: 3})
	if len(first) != 3 || len(rest) != 2 {
		t.Fatalf("pages of %d and %d", len(first), len(rest))
	}
	if rest[0].ID.String() <= first[2].ID.String() {
		t.Fatal("second page overlaps the first")
	}
}

// Every domain table that hangs off entities must be a registered type, so
// export and integrity checks cover it.
func TestEveryEntityTableIsRegistered(t *testing.T) {
	rows, err := pool.Query(context.Background(), `
		SELECT DISTINCT tc.table_name FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage cu ON cu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND cu.table_name = 'entities' AND tc.table_name <> 'links'`)
	if err != nil {
		t.Fatal(err)
	}
	tables, _ := pgx.CollectRows(rows, pgx.RowTo[string])
	registered := map[string]bool{}
	for _, typ := range Types() {
		registered[typ.Table] = true
	}
	for _, table := range tables {
		if !registered[table] {
			t.Errorf("table %s references entities but no graph.Type registers it", table)
		}
	}
}
