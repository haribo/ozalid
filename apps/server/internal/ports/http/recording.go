package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// JudgeRecording renders a verdict on the recording on screen (ADR 0023).
func (s *Server) JudgeRecording(ctx context.Context, request openapi.JudgeRecordingRequestObject) (openapi.JudgeRecordingResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.JudgeRecording401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.JudgeRecording403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}
	remark := ""
	if request.Body.Remark != nil {
		remark = *request.Body.Remark
	}

	state, err := s.evidence.JudgeRecording(ctx, request.Slug, request.RecordingId, actorFrom(ctx), request.Body.Accept, remark)
	switch {
	case errors.Is(err, review.ErrRemarkRequired):
		return openapi.JudgeRecording400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("remark-required", "A refusal explains itself", http.StatusBadRequest,
					"Refusing a recording needs the remark the dev will read (ADR 0020).")),
		}, nil
	case errors.Is(err, review.ErrMoveNotAllowed):
		return openapi.JudgeRecording404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("recording"),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.JudgeRecording200JSONResponse{CaseState: openapi.CaseState(state)}, nil
}

// UnjudgeRecording takes the recording's verdict back, symmetrically.
func (s *Server) UnjudgeRecording(ctx context.Context, request openapi.UnjudgeRecordingRequestObject) (openapi.UnjudgeRecordingResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.UnjudgeRecording401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.UnjudgeRecording403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	state, err := s.evidence.UnjudgeRecording(ctx, request.Slug, request.RecordingId, actorFrom(ctx))
	switch {
	case errors.Is(err, review.ErrMoveNotAllowed):
		return openapi.UnjudgeRecording404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("recording"),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.UnjudgeRecording200JSONResponse{CaseState: openapi.CaseState(state)}, nil
}
