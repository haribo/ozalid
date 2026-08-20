// Package postgres implements the outbound ports declared by the app layer.
//
// Transactions are opened here, never in a use case: app cannot import an
// adapter (backend ADR 0001), so a write spanning several tables becomes one
// method on this type.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
)

// Repository is the Postgres-backed store.
type Repository struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

// Open connects to the database and verifies it answers.
func Open(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("opening the pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("reaching the database: %w", err)
	}
	return &Repository{pool: pool, q: sqlcgen.New(pool)}, nil
}

// Close releases the pool.
func (r *Repository) Close() { r.pool.Close() }

// Ping reports whether the store is reachable.
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Queries exposes the generated accessors.
func (r *Repository) Queries() *sqlcgen.Queries { return r.q }

// Begin starts a transaction. Multi-statement writes whose all-or-nothing
// outcome is part of the contract run inside one (backend ADR 0003).
func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}
