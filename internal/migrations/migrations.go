package migrations

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var files embed.FS

func Run(ctx context.Context, db *pgxpool.Pool, logger *slog.Logger) error {
	if _, err := db.Exec(ctx, `
		create table if not exists schema_migrations (
			version text primary key,
			applied_at timestamptz not null default now()
		)`); err != nil {
		return err
	}

	entries, err := files.ReadDir("sql")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := migrationApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sql, err := files.ReadFile(path.Join("sql", name))
		if err != nil {
			return err
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, "insert into schema_migrations (version) values ($1)", name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		logger.Info("migration applied", slog.String("version", name))
	}

	return nil
}

func migrationApplied(ctx context.Context, db *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, "select exists(select 1 from schema_migrations where version = $1)", name).Scan(&exists)
	return exists, err
}
