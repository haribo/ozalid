package http

import (
	"context"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

// anonymous is who acts when nothing was presented.
//
// It lives here, in the layer that reads requests, rather than in the domain: a
// stand-in for a missing credential is a property of the door, not of the book.
// Every recorded fact still states an actor (ADR 0002), so the journal is
// honest about not knowing who rather than leaving the field empty, and the
// rows written this way stay as they are once identity lands (ADR 0018).
//
// Sixteen of the eighteen endpoints still resolve to it: the web client has no
// way to prove itself until sign-in exists, and closing its door first would
// leave an instance nobody can read.
var anonymous = actor.Actor{ID: "anonymous", Kind: actor.Human}

// Tokens reads which service account a token belongs to.
type Tokens interface {
	ServiceAccountByToken(ctx context.Context, token string) (actor.Actor, bool, error)
}

type actorKey struct{}

// withActor resolves who is calling, once per request, and carries the answer
// down.
//
// One place decides. A header that is absent, malformed, or names a token
// nobody knows all resolve the same way — to nobody — so a caller cannot tell a
// real token from an invented one by how the server behaves.
//
// Resolving is not deciding: this says who is calling, and `access.Allows` says
// what they may do. Keeping them apart is what lets an unauthenticated call
// reach an endpoint that does not require one.
func withActor(tokens Tokens, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		by := anonymous
		if token, ok := credential.FromHeader(r.Header.Get("Authorization")); ok {
			if resolved, found, err := tokens.ServiceAccountByToken(r.Context(), token); err == nil && found {
				by = resolved
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, by)))
	})
}

// actorFrom returns who the request resolved to.
//
// A context with no actor means the handler was reached without passing through
// withActor, which no route does. Returning the zero value rather than
// panicking keeps a wiring mistake from becoming an outage, and `Actor.Zero`
// makes it visible in what gets recorded.
func actorFrom(ctx context.Context) actor.Actor {
	by, _ := ctx.Value(actorKey{}).(actor.Actor)
	return by
}
