package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

// ServiceAccountByToken returns who a token belongs to.
//
// An unknown token resolves to nothing rather than to an error: from the
// caller's side "this is not a token I know" and "this is not a token" are the
// same answer, and giving them different shapes would let someone tell a real
// token from an invented one by the way the server behaves.
//
// The lookup is by hash, so the token itself never reaches a query plan or a
// slow-query log.
func (r *Repository) ServiceAccountByToken(ctx context.Context, token string) (actor.Actor, bool, error) {
	row, err := r.q.ServiceAccountByTokenHash(ctx, credential.Hash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return actor.Actor{}, false, nil
	}
	if err != nil {
		return actor.Actor{}, false, translate("reading a token", err)
	}

	// Recorded on the way through, and deliberately unchecked. Knowing when a
	// token was last used is worth having and not worth refusing a call that was
	// otherwise good — an instance whose intake fails because a bookkeeping
	// write failed is worse than one that forgets a timestamp.
	_ = r.q.TouchServiceToken(ctx, row.TokenID)

	return actor.Actor{ID: row.ID, Kind: actor.Machine}, true, nil
}
