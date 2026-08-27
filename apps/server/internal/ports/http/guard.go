package http

import (
	"context"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// Standings reads what an actor may do on the project a caller named.
type Standings interface {
	StandingOnSlug(ctx context.Context, by actor.Actor, slug string) (access.Standing, error)
}

// unknownCaller reports that nothing usable was presented.
//
// Absent, malformed, and naming a token nobody knows are one answer: telling
// them apart would let a caller learn which tokens exist by watching how the
// server replies.
func unknownCaller() openapi.Problem {
	return problem("unauthenticated", "No usable credential was presented",
		http.StatusUnauthorized,
		"This endpoint is reached with a service account's token, sent as `Authorization: Bearer <token>`.")
}

// refused reports that the caller is somebody, and that somebody may not.
//
// Kept apart from `unknownCaller` on purpose: one is fixed by presenting a
// credential, the other by being granted something. A client that cannot tell
// them apart retries forever.
func refused() openapi.Problem {
	return problem("forbidden", "This account may not do that", http.StatusForbidden,
		"A service account belongs to one project and reaches nothing outside it.")
}

// isKnown reports whether the caller resolved to an account.
//
// Some endpoints need nothing more. Content-addressed storage is shared across
// projects by design (ADR 0004), so uploading bytes has no project to be a
// member of — the meaningful bar there is holding a token at all.
func isKnown(ctx context.Context) bool {
	by := actorFrom(ctx)
	return !by.Zero() && by.ID != anonymous.ID
}

// mayNot reports why a caller cannot take an action on a project, or false when
// they can.
//
// Handlers ask; nothing re-derives the rule (product.md §8.1).
func (s *Server) mayNot(ctx context.Context, slug string, a access.Action) (openapi.Problem, bool) {
	if !isKnown(ctx) {
		return unknownCaller(), true
	}
	standing, err := s.standings.StandingOnSlug(ctx, actorFrom(ctx), slug)
	if err != nil || !access.Allows(standing, a) {
		// A lookup that failed and a caller who is not allowed give the same
		// answer. Anything else would let someone map an instance by watching
		// which refusals differ.
		return refused(), true
	}
	return openapi.Problem{}, false
}
