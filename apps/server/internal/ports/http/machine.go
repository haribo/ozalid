package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/app/account"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// CreateServiceAccount makes a program an account on one project, with its
// first token.
//
// Administration, not membership. Creating one is granting a membership, and an
// administrator grants memberships (product.md §8.2). A member of the project
// could otherwise mint a credential that keeps working after their own
// membership is revoked.
func (s *Server) CreateServiceAccount(ctx context.Context, request openapi.CreateServiceAccountRequestObject) (openapi.CreateServiceAccountResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.CreateServiceAccount401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.CreateServiceAccount403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	rights := access.Member
	if request.Body.Rights != nil {
		rights = access.Rights(*request.Body.Rights)
	}

	made, token, err := s.account.CreateServiceAccount(
		ctx, request.Slug, request.Body.Name, actorFrom(ctx).ID, rights, request.Body.TokenLabel,
	)
	switch {
	case errors.Is(err, account.ErrNameRequired):
		return badServiceAccount("service-account-needs-a-name", "A service account needs a name",
			"It is what names the program in the journal."), nil
	case errors.Is(err, account.ErrLabelRequired):
		return badServiceAccount("token-needs-a-label", "A token needs a label",
			"A token nobody can name is a token nobody dares retire."), nil
	case errors.Is(err, account.ErrUnknownRights):
		return badServiceAccount("unknown-rights", "Rights are reader or member",
			"Two, and no more."), nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.CreateServiceAccount404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("project"),
		}, nil
	case err != nil:
		return nil, err
	}

	return openapi.CreateServiceAccount201JSONResponse{
		Account: openapi.ServiceAccount{Id: made.ID, Name: made.Name, CreatedAt: made.CreatedAt},
		Token:   toAPIMintedToken(token),
	}, nil
}

// DeactivateServiceAccount retires a program.
func (s *Server) DeactivateServiceAccount(ctx context.Context, request openapi.DeactivateServiceAccountRequestObject) (openapi.DeactivateServiceAccountResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.DeactivateServiceAccount401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.DeactivateServiceAccount403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	err := s.account.DeactivateServiceAccount(ctx, request.Slug, request.ServiceAccountId)
	if errors.Is(err, account.ErrNotFound) {
		return openapi.DeactivateServiceAccount404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("service account"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.DeactivateServiceAccount204Response{}, nil
}

// ListServiceTokens returns what a program holds, never the tokens themselves.
func (s *Server) ListServiceTokens(ctx context.Context, request openapi.ListServiceTokensRequestObject) (openapi.ListServiceTokensResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.ListServiceTokens401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.ListServiceTokens403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	found, err := s.account.Tokens(ctx, request.Slug, request.ServiceAccountId)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.ListServiceTokens404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("service account"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]openapi.ServiceToken, 0, len(found))
	for _, t := range found {
		out = append(out, openapi.ServiceToken{
			Id: t.ID, Label: t.Label, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt,
		})
	}
	return openapi.ListServiceTokens200JSONResponse(out), nil
}

// MintServiceToken adds a token, and leaves the others working.
func (s *Server) MintServiceToken(ctx context.Context, request openapi.MintServiceTokenRequestObject) (openapi.MintServiceTokenResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.MintServiceToken401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.MintServiceToken403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	token, err := s.account.MintToken(ctx, request.Slug, request.ServiceAccountId, request.Body.Label)
	switch {
	case errors.Is(err, account.ErrLabelRequired):
		return openapi.MintServiceToken400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("token-needs-a-label", "A token needs a label", http.StatusBadRequest,
					"A token nobody can name is a token nobody dares retire."),
			),
		}, nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.MintServiceToken404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("service account"),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.MintServiceToken201JSONResponse(toAPIMintedToken(token)), nil
}

// RetireServiceToken stops one token working, leaving the others alone.
func (s *Server) RetireServiceToken(ctx context.Context, request openapi.RetireServiceTokenRequestObject) (openapi.RetireServiceTokenResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.RetireServiceToken401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.RetireServiceToken403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	if err := s.account.RetireToken(ctx, request.Slug, request.ServiceAccountId, request.TokenId); err != nil {
		return nil, err
	}
	return openapi.RetireServiceToken204Response{}, nil
}

func badServiceAccount(kind, title, detail string) openapi.CreateServiceAccount400ApplicationProblemPlusJSONResponse {
	return openapi.CreateServiceAccount400ApplicationProblemPlusJSONResponse{
		BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
			problem(kind, title, http.StatusBadRequest, detail),
		),
	}
}

func toAPIMintedToken(t account.MintedToken) openapi.MintedToken {
	return openapi.MintedToken{Id: t.ID, Label: t.Label, Token: t.Token, CreatedAt: t.CreatedAt}
}
