// Package cases holds the use cases operating on a case.
//
// It depends only on the interfaces declared here; the concrete adapters are
// wired in cmd/server (backend ADR 0001).
package cases

import "context"

// Clock is injected so a time-dependent transition can be tested without a
// wall clock.
type Clock interface {
	Now() int64
}

// Repository is the outbound port this package needs. Its implementation lives
// under internal/adapters and is never imported from here.
type Repository interface {
	Ping(ctx context.Context) error
}
