package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

// StartSignIn records a link for the person at this address and returns the
// value to put in the message.
//
// An address nobody has resolves to no link and no error. Whether an address is
// known is not something a stranger gets to learn by asking, and the caller
// answers the same either way.
func (r *Repository) StartSignIn(ctx context.Context, email string) (link string, sendIt bool, err error) {
	user, err := r.q.UserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, translate("reading the account", err)
	}

	value, hash, err := credential.Secret()
	if err != nil {
		return "", false, err
	}
	if err := r.q.CreateSignInLink(ctx, sqlcgen.CreateSignInLinkParams{
		UserID: user.ID, LinkHash: hash,
		LifetimeSeconds: int32(credential.LinkLifetime.Seconds()),
	}); err != nil {
		return "", false, translate("recording the sign-in link", err)
	}
	return value, true, nil
}

// ClaimSignIn spends a link and opens a session.
//
// Expired, already used and never issued are one answer: the browser is told
// the link no longer works, and learns nothing about which of the three it was.
func (r *Repository) ClaimSignIn(ctx context.Context, link string) (session string, ok bool, err error) {
	userID, err := r.q.ClaimSignInLink(ctx, credential.Hash(link))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, translate("claiming the sign-in link", err)
	}

	value, hash, err := credential.Secret()
	if err != nil {
		return "", false, err
	}
	if err := r.q.CreateSession(ctx, sqlcgen.CreateSessionParams{
		UserID: userID, TokenHash: hash,
		LifetimeSeconds: int32(credential.SessionLifetime.Seconds()),
	}); err != nil {
		return "", false, translate("opening the session", err)
	}
	return value, true, nil
}

// UserBySession returns who a session belongs to.
//
// A deactivated account resolves to nothing, so shutting an account shuts every
// session it has in the same instant — which is the whole reason a session is a
// row here rather than a signed token the server cannot take back.
func (r *Repository) UserBySession(ctx context.Context, token string) (actor.Actor, bool, error) {
	row, err := r.q.UserBySessionToken(ctx, credential.Hash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return actor.Actor{}, false, nil
	}
	if err != nil {
		return actor.Actor{}, false, translate("reading the session", err)
	}

	// Bookkeeping, deliberately unchecked: knowing when a session was last used
	// is worth having and not worth refusing a request that was otherwise good.
	_ = r.q.TouchSession(ctx, row.SessionID)

	return actor.Actor{ID: row.ID, Kind: actor.Human}, true, nil
}

// EndSession forgets one session. Signing out of a browser must not sign out of
// the others.
func (r *Repository) EndSession(ctx context.Context, token string) error {
	if err := r.q.EndSession(ctx, credential.Hash(token)); err != nil {
		return translate("ending the session", err)
	}
	return nil
}

// Person returns what the client is told about whoever is signed in.
func (r *Repository) Person(ctx context.Context, id string) (account.Person, bool, error) {
	row, err := r.q.UserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Person{}, false, nil
	}
	if err != nil {
		return account.Person{}, false, translate("reading the account", err)
	}
	return account.Person{
		ID: row.ID, Name: row.Name, Email: row.Email, IsAdmin: row.IsAdmin,
	}, true, nil
}
