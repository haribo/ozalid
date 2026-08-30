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
	case errors.Is(err, session.ErrUnknownKind):
		return badReview("unknown-comment-kind", "A comment is neither a defect nor an improvement", ""), nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.SaveReview404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	case err != nil:
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
	if body.Validated != nil {
		for _, cell := range *body.Validated {
			save.Validated = append(save.Validated, review.Cell{
				StepID: cell.StepId, VariantID: cell.VariantId,
			})
		}
	}
	if body.Comments != nil {
		for _, c := range *body.Comments {
			save.Comments = append(save.Comments, session.NewComment{
				StepID: c.StepId, Kind: string(c.Kind), Body: c.Body, VariantIDs: c.VariantIds,
			})
		}
	}
	return save
}

func toAPIOutcome(r session.Result) openapi.ReviewOutcome {
	verdicts := make([]openapi.CellVerdict, 0, len(r.Verdicts))
	for cell, status := range r.Verdicts {
		verdicts = append(verdicts, openapi.CellVerdict{
			StepId:    cell.StepID,
			VariantId: cell.VariantID,
			Status:    openapi.CellVerdictStatus(status),
		})
	}
	return openapi.ReviewOutcome{
		State:    openapi.CaseState(r.State),
		Comments: r.Comments,
		Verdicts: verdicts,
	}
}
