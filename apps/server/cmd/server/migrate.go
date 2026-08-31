package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/haribo/ozalid/apps/server/db"
)

// migrate brings the schema up to date, before the listener opens.
//
// At boot and unconditionally. A server that starts against a schema it does
// not know fails on the first request that touches the missing column — an
// outage discovered by a user, minutes after a deploy that looked fine. Failing
// at startup means the deploy fails instead, which is where a failure belongs.
//
// Applying nothing is the common case and costs one query. Running it twice is
// harmless: goose records what it has done.
func migrate(ctx context.Context, dsn string) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("opening the database to migrate it: %w", err)
	}
	defer func() { _ = conn.Close() }()

	goose.SetBaseFS(db.Migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting the dialect: %w", err)
	}

	before, err := goose.GetDBVersionContext(ctx, conn)
	if err != nil {
		return fmt.Errorf("reading the schema version: %w", err)
	}
	if err := goose.UpContext(ctx, conn, "migrations"); err != nil {
		return fmt.Errorf("migrating: %w", err)
	}
	after, err := goose.GetDBVersionContext(ctx, conn)
	if err != nil {
		return fmt.Errorf("reading the schema version: %w", err)
	}

	if after != before {
		slog.Info("schema migrated", "from", before, "to", after)
	}
	return nil
}
