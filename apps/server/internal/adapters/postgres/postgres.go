// Package postgres implements the outbound ports declared by the app layer.
package postgres

import "context"

// Repository is the Postgres-backed implementation of cases.Repository.
type Repository struct{}

// Ping reports whether the store is reachable.
func (r *Repository) Ping(ctx context.Context) error { return nil }
