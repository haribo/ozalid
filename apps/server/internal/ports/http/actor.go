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
type sessionKey struct{}

// withActor resolves who is calling, once per request, and carries the answer
// down.
//
// One place decides. Two credentials are read, and only one of them can be
// presented at a time in practice: a program sends a bearer token, a browser
// carries a session cookie. A header that is absent, malformed, or names
// something nobody knows all resolve the same way — to nobody — so a caller
// cannot tell a real credential from an invented one by how the server behaves.
//
// The token wins if both arrive. A request carrying both is a client doing
// something deliberate, and the explicit header is the more deliberate of the
// two.
//
// Resolving is not deciding: this says who is calling, and `access.Allows` says
// what they may do. Keeping them apart is what lets an unauthenticated call
// reach an endpoint that does not require one.
func withActor(tokens Tokens, sessions SignIn, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		by := anonymous

		if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
			ctx = context.WithValue(ctx, sessionKey{}, c.Value)
			if resolved, found, err := sessions.UserBySession(ctx, c.Value); err == nil && found {
				by = resolved
			}
		}
		if token, ok := credential.FromHeader(r.Header.Get("Authorization")); ok {
			if resolved, found, err := tokens.ServiceAccountByToken(ctx, token); err == nil && found {
				by = resolved
			}
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, actorKey{}, by)))
	})
}

// sessionFrom returns the session the browser presented, whether or not it
// resolved to anybody. Signing out has to forget a session that has expired
// just as much as one that has not.
func sessionFrom(ctx context.Context) string {
	token, _ := ctx.Value(sessionKey{}).(string)
	return token
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
