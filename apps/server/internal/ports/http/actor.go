package http

import (
	"context"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
)

// anonymous is who acts until a credential can be proven.
//
// It lives here, in the layer that reads requests, rather than in the domain: a
// stand-in for a missing capability is a property of the door, not of the book.
// Every recorded fact still states an actor (ADR 0002), so the journal is
// honest about not knowing who rather than leaving the field empty, and the
// rows written this way stay as they are once identity lands (ADR 0018).
var anonymous = actor.Actor{ID: "anonymous", Kind: actor.Human}

type actorKey struct{}

// withActor resolves who is calling, once per request, and carries the answer
// down.
//
// One place decides. When credentials arrive this is the function that reads
// them; until then it answers the same thing every time, which is exactly why
// it is worth having before there is anything to read.
func withActor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, anonymous)))
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
