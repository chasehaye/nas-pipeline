// Package migrate applies embedded, versioned SQL migrations on startup.
// Files live in sql/NNNN_name.sql; every file with a version above the highest
// recorded in schema_migrations is applied — forward-only, each atomically.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var files embed.FS

// advisoryKey guards the migration run so concurrent instances can't race.
const advisoryKey int64 = 4577320981

type migration struct {
	version int
	name    string
	sql     string
}

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	migs, err := load()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryKey); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryKey)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
    version    integer PRIMARY KEY,
    name       text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	var current int
	if err := conn.QueryRow(ctx,
		"SELECT COALESCE(max(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("read current version: %w", err)
	}

	applied := 0
	for _, m := range migs {
		if m.version <= current {
			continue
		}
		script := fmt.Sprintf("%s\nINSERT INTO schema_migrations (version, name) VALUES (%d, %s);",
			m.sql, m.version, quoteLiteral(m.name))
		if _, err := conn.Conn().PgConn().Exec(ctx, script).ReadAll(); err != nil {
			return fmt.Errorf("migration %04d_%s: %w", m.version, m.name, err)
		}
		log.Printf("migrate: applied %04d_%s", m.version, m.name)
		applied++
	}

	if applied == 0 {
		log.Printf("migrate: schema up to date (version %d)", current)
	} else {
		log.Printf("migrate: applied %d migration(s), now at version %d", applied, migs[len(migs)-1].version)
	}
	return nil
}

func load() ([]migration, error) {
	entries, err := files.ReadDir("sql")
	if err != nil {
		return nil, err
	}

	var migs []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(strings.TrimSuffix(e.Name(), ".sql"), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad migration filename %q (want NNNN_name.sql)", e.Name())
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("bad version in %q: %w", e.Name(), err)
		}
		b, err := files.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, err
		}
		migs = append(migs, migration{version: v, name: parts[1], sql: string(b)})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	return migs, nil
}

func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
