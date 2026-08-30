package http

import (
	"context"
	"errors"
	"net/http"

	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// TrackComment attaches an external issue.
func (s *Server) TrackComment(ctx context.Context, request openapi.TrackCommentRequestObject) (openapi.TrackCommentResponseObject, error) {
	issue := comment.IssueRef{ID: request.Body.Id}
	if request.Body.Url != nil {
		issue.URL = *request.Body.Url
	}
	if request.Body.Title != nil {
		issue.Title = *request.Body.Title
	}

	out, err := s.comment.Track(ctx, request.Slug, request.CommentId, actorFrom(ctx), issue)
	switch {
	case errors.Is(err, comment.ErrIssueRequired):
		return openapi.TrackComment400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("issue-required", "Tracking needs an issue reference", http.StatusBadRequest,
					"A comment tracked by nothing is a comment nobody is working on."),
			),
		}, nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.TrackComment404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("comment"),
		}, nil
	case isRefusedMove(err):
		return openapi.TrackComment409ApplicationProblemPlusJSONResponse{MoveRefusedApplicationProblemPlusJSONResponse: openapi.MoveRefusedApplicationProblemPlusJSONResponse(refusedMove(err))}, nil
	case err != nil:
		return nil, err
	}
	return openapi.TrackComment200JSONResponse{MoveAppliedJSONResponse: openapi.MoveAppliedJSONResponse(toAPIMove(out))}, nil
}

// DiscardComment sets a comment aside, with its reason.
func (s *Server) DiscardComment(ctx context.Context, request openapi.DiscardCommentRequestObject) (openapi.DiscardCommentResponseObject, error) {
	out, err := s.comment.Discard(ctx, request.Slug, request.CommentId, actorFrom(ctx), request.Body.Reason)
	switch {
	case errors.Is(err, review.ErrReasonRequired):
		return openapi.DiscardComment400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("reason-required", "Discarding needs a reason", http.StatusBadRequest,
					"A defect dismissed without one is exactly what comes back in six months."),
			),
		}, nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.DiscardComment404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("comment"),
		}, nil
	case isRefusedMove(err):
		return openapi.DiscardComment409ApplicationProblemPlusJSONResponse{MoveRefusedApplicationProblemPlusJSONResponse: openapi.MoveRefusedApplicationProblemPlusJSONResponse(refusedMove(err))}, nil
	case err != nil:
		return nil, err
	}
	return openapi.DiscardComment200JSONResponse{MoveAppliedJSONResponse: openapi.MoveAppliedJSONResponse(toAPIMove(out))}, nil
}

// DeliverComment is the dev asking for a judgment.
func (s *Server) DeliverComment(ctx context.Context, request openapi.DeliverCommentRequestObject) (openapi.DeliverCommentResponseObject, error) {
	out, err := s.comment.Deliver(ctx, request.Slug, request.CommentId, actorFrom(ctx))
	switch {
	case errors.Is(err, app.ErrNotFound):
		return openapi.DeliverComment404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("comment"),
		}, nil
	case isRefusedMove(err):
		return openapi.DeliverComment409ApplicationProblemPlusJSONResponse{MoveRefusedApplicationProblemPlusJSONResponse: openapi.MoveRefusedApplicationProblemPlusJSONResponse(refusedMove(err))}, nil
	case err != nil:
		return nil, err
	}
	return openapi.DeliverComment200JSONResponse{MoveAppliedJSONResponse: openapi.MoveAppliedJSONResponse(toAPIMove(out))}, nil
}

// JudgeComment accepts a delivery, or refuses it with a remark.
func (s *Server) JudgeComment(ctx context.Context, request openapi.JudgeCommentRequestObject) (openapi.JudgeCommentResponseObject, error) {
	remark := ""
	if request.Body.Remark != nil {
		remark = *request.Body.Remark
	}

	out, err := s.comment.Judge(ctx, request.Slug, request.CommentId, actorFrom(ctx), request.Body.Accept, remark)
	switch {
	case errors.Is(err, review.ErrRemarkRequired):
		return openapi.JudgeComment400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("remark-required", "Refusing needs a remark", http.StatusBadRequest,
					"What the dev has to read is the remark, not the state."),
			),
		}, nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.JudgeComment404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("comment"),
		}, nil
	case isRefusedMove(err):
		return openapi.JudgeComment409ApplicationProblemPlusJSONResponse{MoveRefusedApplicationProblemPlusJSONResponse: openapi.MoveRefusedApplicationProblemPlusJSONResponse(refusedMove(err))}, nil
	case err != nil:
		return nil, err
	}
	return openapi.JudgeComment200JSONResponse{MoveAppliedJSONResponse: openapi.MoveAppliedJSONResponse(toAPIMove(out))}, nil
}

// ListComments returns what has been said about a case, settled included.
func (s *Server) ListComments(ctx context.Context, request openapi.ListCommentsRequestObject) (openapi.ListCommentsResponseObject, error) {
	comments, err := s.comment.OfCase(ctx, request.Slug, request.CaseId)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.ListComments404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	out := make([]openapi.Comment, 0, len(comments))
	for _, c := range comments {
		out = append(out, toAPIComment(c))
	}
	return openapi.ListComments200JSONResponse(out), nil
}

// isRefusedMove reports whether the domain turned the move down. That is an
// answer about the state of things, not a failure.
func isRefusedMove(err error) bool {
	return errors.Is(err, review.ErrNotOpen) || errors.Is(err, review.ErrMoveNotAllowed)
}

func refusedMove(err error) openapi.Problem {
	kind, title := "move-not-available", "That move is not available from this state"
	if errors.Is(err, review.ErrNotOpen) {
		kind, title = "comment-settled", "The comment is settled"
	}
	return problem(kind, title, http.StatusConflict, err.Error())
}

func toAPIMove(o comment.Outcome) openapi.MoveOutcome {
	return openapi.MoveOutcome{
		CommentState: openapi.CommentState(o.CommentState),
		CaseState:    openapi.CaseState(o.CaseState),
	}
}

func toAPIComment(c comment.Record) openapi.Comment {
	out := openapi.Comment{
		Id:         c.ID,
		StepId:     c.StepID,
		Kind:       openapi.CommentKind(c.Kind),
		Body:       c.Body,
		State:      openapi.CommentState(c.State),
		VariantIds: c.VariantIDs,
		AuthorId:   c.AuthorID,
		CreatedAt:  c.CreatedAt,
		Judgments:  make([]openapi.Judgment, 0, len(c.Judgments)),
	}
	if c.Issue != nil {
		out.Issue = &openapi.IssueRef{
			Id: c.Issue.ID, Url: nonEmptyPtr(c.Issue.URL), Title: nonEmptyPtr(c.Issue.Title),
		}
	}
	out.DiscardReason = nonEmptyPtr(c.DiscardReason)
	for _, j := range c.Judgments {
		out.Judgments = append(out.Judgments, openapi.Judgment{
			Verdict: openapi.JudgmentVerdict(j.Verdict),
			Remark:  nonEmptyPtr(j.Remark),
			ActorId: j.ActorID,
			At:      j.At,
		})
	}
	return out
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
