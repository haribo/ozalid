package http

import (
	"context"
	"errors"
	"net/http"

	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// SaveReview records what one review session decided.
func (s *Server) SaveReview(ctx context.Context, request openapi.SaveReviewRequestObject) (openapi.SaveReviewResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.SaveReview401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.SaveReview403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}
	save := toSave(*request.Body)

	result, err := s.session.Save(ctx, request.Slug, request.CaseId, actorFrom(ctx), save)
	switch {
	case errors.Is(err, session.ErrEmptyBody):
		return badReview("empty-comment", "A comment needs a body",
			"An empty report is one nobody can act on."), nil
	case errors.Is(err, session.ErrNoVariant):
		return badReview("comment-covers-nothing", "A comment covers no variant",
			"One defect spanning four variants is one comment with four variants checked."), nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.SaveReview404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	case err != nil:
		var held *review.Held
		if errors.As(err, &held) {
			return openapi.SaveReview423ApplicationProblemPlusJSONResponse{
				HeldApplicationProblemPlusJSONResponse: openapi.HeldApplicationProblemPlusJSONResponse(heldProblem(held)),
			}, nil
		}
		return nil, err
	}

	return openapi.SaveReview200JSONResponse(toAPIOutcome(result)), nil
}

func badReview(kind, title, detail string) openapi.SaveReview400ApplicationProblemPlusJSONResponse {
	return openapi.SaveReview400ApplicationProblemPlusJSONResponse{
		BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
			problem(kind, title, http.StatusBadRequest, detail),
		),
	}
}

func toSave(body openapi.ReviewSave) session.Save {
	var save session.Save
	if body.Accepted != nil {
		for _, capture := range *body.Accepted {
			save.Accepted = append(save.Accepted, review.Capture{
				StepID: capture.StepId, VariantID: capture.VariantId,
			})
		}
	}
	if body.Unaccepted != nil {
		for _, capture := range *body.Unaccepted {
			save.Unaccepted = append(save.Unaccepted, review.Capture{
				StepID: capture.StepId, VariantID: capture.VariantId,
			})
		}
	}
	if body.Unrefused != nil {
		for _, capture := range *body.Unrefused {
			save.Unrefused = append(save.Unrefused, review.Capture{
				StepID: capture.StepId, VariantID: capture.VariantId,
			})
		}
	}
	if body.Comments != nil {
		for _, c := range *body.Comments {
			save.Comments = append(save.Comments, session.NewComment{
				StepID: c.StepId, Body: c.Body, VariantIDs: c.VariantIds,
			})
		}
	}
	return save
}

func toAPIOutcome(r session.Result) openapi.ReviewOutcome {
	verdicts := make([]openapi.CaptureVerdict, 0, len(r.Verdicts))
	for capture, status := range r.Verdicts {
		verdicts = append(verdicts, openapi.CaptureVerdict{
			StepId:    capture.StepID,
			VariantId: capture.VariantID,
			Status:    openapi.CaptureVerdictStatus(status),
		})
	}
	return openapi.ReviewOutcome{
		State:    openapi.CaseState(r.State),
		Comments: r.Comments,
		Verdicts: verdicts,
	}
}
