// Package instance moves a whole Axon to another server: every table of the
// database (uploaded files included, since they are stored there) in one
// gzipped tar archive, with a manifest of versions and checksums.
//
// The archive holds manifest.json first, then tables/<name>.copy in
// PostgreSQL's COPY text format. Secrets sealed with AXON_ENCRYPTION_KEY stay
// sealed; the manifest carries a fingerprint of the key, and an import
// refuses a different key rather than leave secrets nobody can open.
package instance

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Format names the archive layout; an import refuses any other.
const Format = "axon-instance/1"

// Manifest describes an archive.
type Manifest struct {
	Format         string    `json:"format"`
	AxonVersion    string    `json:"axon_version"`
	SchemaVersion  int64     `json:"schema_version"`
	CreatedAt      time.Time `json:"created_at"`
	KeyFingerprint string    `json:"key_fingerprint"` // empty when the instance had no encryption key
	Tables         []Table   `json:"tables"`
}

// Table is one table's entry in the manifest.
type Table struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    int64    `json:"rows"`
	SHA256  string   `json:"sha256"`
}

// KeyFingerprint identifies an AXON_ENCRYPTION_KEY without revealing it.
func KeyFingerprint(hexKey string) string {
	if hexKey == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("axon instance key check\x00" + strings.ToLower(hexKey)))
	return hex.EncodeToString(sum[:8])
}

// Export writes the archive of every table to w, read in one snapshot.
func Export(ctx context.Context, pool *pgxpool.Pool, w io.Writer, axonVersion, hexKey string) (Manifest, error) {
	m := Manifest{Format: Format, AxonVersion: axonVersion, CreatedAt: time.Now().UTC(), KeyFingerprint: KeyFingerprint(hexKey)}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return m, err
	}
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return m, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if m.SchemaVersion, err = schemaVersion(ctx, tx); err != nil {
		return m, err
	}
	names, err := tables(ctx, tx)
	if err != nil {
		return m, err
	}
	// Tar needs each size up front, so tables go to temporary files first.
	dir, err := os.MkdirTemp("", "axon-export-")
	if err != nil {
		return m, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	files := make([]string, len(names))
	for i, name := range names {
		cols, err := columns(ctx, tx, name)
		if err != nil {
			return m, err
		}
		f, err := os.CreateTemp(dir, "table-")
		if err != nil {
			return m, err
		}
		h := sha256.New()
		tag, err := tx.Conn().PgConn().CopyTo(ctx, io.MultiWriter(f, h), "COPY "+target(name, cols)+" TO STDOUT")
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return m, fmt.Errorf("export %s: %w", name, err)
		}
		files[i] = f.Name()
		m.Tables = append(m.Tables, Table{Name: name, Columns: cols, Rows: tag.RowsAffected(), SHA256: hex.EncodeToString(h.Sum(nil))})
	}

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	manifest, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return m, err
	}
	if err := writeEntry(tw, "manifest.json", int64(len(manifest)), strings.NewReader(string(manifest))); err != nil {
		return m, err
	}
	for i, t := range m.Tables {
		if err := writeFile(tw, "tables/"+t.Name+".copy", files[i]); err != nil {
			return m, err
		}
	}
	if err := tw.Close(); err != nil {
		return m, err
	}
	return m, gz.Close()
}

// Migrator brings the target schema to exactly the given migration version.
type Migrator func(ctx context.Context, version int64) error

// Import loads an archive into an empty installation. It reads the manifest,
// migrates the target to the export's schema version, checks that every table
// is empty, and copies the tables in one transaction, checking each row count
// and checksum. The caller then applies any newer migrations.
func Import(ctx context.Context, pool *pgxpool.Pool, r io.Reader, hexKey string, latest int64, migrateTo Migrator) (Manifest, error) {
	var m Manifest
	gz, err := gzip.NewReader(r)
	if err != nil {
		return m, fmt.Errorf("not an Axon instance archive: %w", err)
	}
	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil || hdr.Name != "manifest.json" {
		return m, errors.New("not an Axon instance archive: manifest.json must come first")
	}
	if err := json.NewDecoder(io.LimitReader(tr, 16<<20)).Decode(&m); err != nil {
		return m, fmt.Errorf("read manifest: %w", err)
	}
	if m.Format != Format {
		return m, fmt.Errorf("archive format %q is not %q", m.Format, Format)
	}
	if m.SchemaVersion > latest {
		return m, fmt.Errorf("the archive needs schema version %d but this Axon only knows %d: upgrade Axon first", m.SchemaVersion, latest)
	}
	if m.KeyFingerprint != "" && m.KeyFingerprint != KeyFingerprint(hexKey) {
		return m, errors.New("AXON_ENCRYPTION_KEY differs from the exporting instance's, so its sealed secrets (TOTP, AI provider keys, tokens) could not be opened: set the same key and try again")
	}
	// Moving the schema (down too) is only safe on an installation with no
	// one in it; every table is checked again once the schema matches.
	var hasUsers, accounts bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.users') IS NOT NULL").Scan(&hasUsers); err != nil {
		return m, err
	}
	if hasUsers {
		if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users)").Scan(&accounts); err != nil {
			return m, err
		}
	}
	if accounts {
		return m, errors.New("this installation already has accounts: import only into a fresh installation")
	}
	if err := migrateTo(ctx, m.SchemaVersion); err != nil {
		return m, fmt.Errorf("migrate to %d: %w", m.SchemaVersion, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return m, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if v, err := schemaVersion(ctx, tx); err != nil {
		return m, err
	} else if v != m.SchemaVersion {
		return m, fmt.Errorf("the target is at schema version %d, not %d: import into a fresh installation", v, m.SchemaVersion)
	}
	for _, t := range m.Tables {
		var hasRows bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM "+pgx.Identifier{t.Name}.Sanitize()+")").Scan(&hasRows); err != nil {
			return m, fmt.Errorf("check %s: %w", t.Name, err)
		}
		if hasRows {
			return m, fmt.Errorf("table %s already has rows: import only into a fresh installation", t.Name)
		}
	}
	// Foreign keys are checked by the source; replica mode loads tables in
	// any order. It needs a superuser connection.
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		return m, fmt.Errorf("import needs a superuser database connection: %w", err)
	}
	for _, t := range m.Tables {
		hdr, err := tr.Next()
		if err != nil {
			return m, fmt.Errorf("archive ends before table %s: %w", t.Name, err)
		}
		if hdr.Name != "tables/"+t.Name+".copy" {
			return m, fmt.Errorf("archive has %s where tables/%s.copy was expected", hdr.Name, t.Name)
		}
		h := sha256.New()
		tag, err := tx.Conn().PgConn().CopyFrom(ctx, io.TeeReader(tr, h), "COPY "+target(t.Name, t.Columns)+" FROM STDIN")
		if err != nil {
			return m, fmt.Errorf("import %s: %w", t.Name, err)
		}
		if sum := hex.EncodeToString(h.Sum(nil)); sum != t.SHA256 || tag.RowsAffected() != t.Rows {
			return m, fmt.Errorf("table %s does not match its manifest (rows %d of %d, checksum %s): the archive is damaged", t.Name, tag.RowsAffected(), t.Rows, sum[:12])
		}
	}
	return m, tx.Commit(ctx)
}

func schemaVersion(ctx context.Context, q pgx.Tx) (int64, error) {
	var v int64
	err := q.QueryRow(ctx, "SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied").Scan(&v)
	return v, err
}

func tables(ctx context.Context, q pgx.Tx) ([]string, error) {
	rows, err := q.Query(ctx, "SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'goose_db_version' ORDER BY tablename")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// columns lists the stored columns; generated ones are recomputed on import.
func columns(ctx context.Context, q pgx.Tx, table string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT attname FROM pg_attribute
		WHERE attrelid = ('public.' || quote_ident($1))::regclass AND attnum > 0 AND NOT attisdropped AND attgenerated = ''
		ORDER BY attnum`, table)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func target(table string, cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = pgx.Identifier{c}.Sanitize()
	}
	return pgx.Identifier{"public", table}.Sanitize() + " (" + strings.Join(quoted, ", ") + ")"
}

func writeEntry(tw *tar.Writer, name string, size int64, r io.Reader) error {
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: size, ModTime: time.Now()}); err != nil {
		return err
	}
	_, err := io.Copy(tw, r)
	return err
}

func writeFile(tw *tar.Writer, name, path string) error {
	f, err := os.Open(path) //nolint:gosec // a temporary file this package created
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	return writeEntry(tw, name, st.Size(), f)
}
