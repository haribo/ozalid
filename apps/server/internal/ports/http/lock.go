package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// heldOf unwraps the holder from an error, nil when it is something else.
func heldOf(err error) *review.Held {
	var h *review.Held
	if errors.As(err, &h) {
		return h
	}
	return nil
}

// heldProblem is the 423 a held case answers, naming the holder (ADR 0005).
func heldProblem(h *review.Held) openapi.Problem {
	return problem("held", "Somebody is reviewing this case", http.StatusLocked,
		fmt.Sprintf("%s holds this case since %s. A held case takes no verdict but theirs.",
			h.Name, h.Since.UTC().Format("15:04")))
}

// ClaimCase holds the case for the caller, or renews the hold — the same
// call is the heartbeat (ADR 0005, #95).
func (s *Server) ClaimCase(ctx context.Context, request openapi.ClaimCaseRequestObject) (openapi.ClaimCaseResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.ClaimCase401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.ClaimCase403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	hold, err := s.session.Claim(ctx, request.Slug, request.CaseId, actorFrom(ctx))
	var held *review.Held
	switch {
	case errors.As(err, &held):
		return openapi.ClaimCase423ApplicationProblemPlusJSONResponse{
			HeldApplicationProblemPlusJSONResponse: openapi.HeldApplicationProblemPlusJSONResponse(heldProblem(held)),
		}, nil
	case errors.Is(err, review.ErrMoveNotAllowed):
		return openapi.ClaimCase404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.ClaimCase200JSONResponse{By: hold.By, Name: hold.Name, Since: hold.Since}, nil
}

// ReleaseCase lets the case go. Releasing a lock nobody holds, or somebody
// else's, changes nothing.
func (s *Server) ReleaseCase(ctx context.Context, request openapi.ReleaseCaseRequestObject) (openapi.ReleaseCaseResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.ReleaseCase401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.ReleaseCase403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	err := s.session.Release(ctx, request.Slug, request.CaseId, actorFrom(ctx))
	switch {
	case errors.Is(err, review.ErrMoveNotAllowed):
		return openapi.ReleaseCase404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.ReleaseCase204Response{}, nil
}
