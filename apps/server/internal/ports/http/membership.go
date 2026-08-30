package http

import (
	"context"
	"errors"
	"net/http"

	openapitypes "github.com/oapi-codegen/runtime/types"

	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// ListMembers returns who reaches a project.
//
// Reading it needs `reader`, not administration: seeing who is on a project is
// seeing, and a reader sees everything (product.md §8.1).
func (s *Server) ListMembers(ctx context.Context, request openapi.ListMembersRequestObject) (openapi.ListMembersResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.ReadProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.ListMembers401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.ListMembers403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	found, err := s.account.Members(ctx, request.Slug)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.Membership, 0, len(found))
	for _, m := range found {
		entry := openapi.Membership{
			AccountId: m.AccountID, Name: m.Name, IsPerson: m.IsPerson,
			Rights: openapi.Rights(m.Rights), AddedAt: m.AddedAt,
		}
		if m.Email != "" {
			address := openapitypes.Email(m.Email)
			entry.Email = &address
		}
		out = append(out, entry)
	}
	return openapi.ListMembers200JSONResponse(out), nil
}

// GrantMembership puts a person on a project, or changes what they may do.
//
// An administrator grants memberships — all of them, not only the first
// (product.md §8.2). A member of the project cannot: they hold WriteProject,
// and that is deliberately not enough.
func (s *Server) GrantMembership(ctx context.Context, request openapi.GrantMembershipRequestObject) (openapi.GrantMembershipResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.GrantMembership401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.GrantMembership403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	err := s.account.Grant(ctx, request.Slug, request.AccountId, access.Rights(request.Body.Rights))
	switch {
	case errors.Is(err, account.ErrUnknownRights):
		return openapi.GrantMembership400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("unknown-rights", "Rights are reader or member", http.StatusBadRequest,
					"Two, and no more: a reader sees everything and changes nothing, a member does everything."),
			),
		}, nil
	case errors.Is(err, account.ErrNotFound):
		return openapi.GrantMembership404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("project or account"),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.GrantMembership204Response{}, nil
}

// RevokeMembership takes a person off a project.
func (s *Server) RevokeMembership(ctx context.Context, request openapi.RevokeMembershipRequestObject) (openapi.RevokeMembershipResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.RevokeMembership401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.RevokeMembership403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	if err := s.account.Revoke(ctx, request.Slug, request.AccountId); err != nil {
		return nil, err
	}
	return openapi.RevokeMembership204Response{}, nil
}
