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

// ListAccounts returns every account, deactivated ones included.
func (s *Server) ListAccounts(ctx context.Context, _ openapi.ListAccountsRequestObject) (openapi.ListAccountsResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.ListAccounts401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.ListAccounts403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	found, err := s.account.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.Account, 0, len(found))
	for _, a := range found {
		out = append(out, toAPIAccount(a))
	}
	return openapi.ListAccounts200JSONResponse(out), nil
}

// CreateAccount makes an account for a person.
func (s *Server) CreateAccount(ctx context.Context, request openapi.CreateAccountRequestObject) (openapi.CreateAccountResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.CreateAccount401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.CreateAccount403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	admin := request.Body.IsAdmin != nil && *request.Body.IsAdmin
	made, err := s.account.Create(ctx, request.Body.Name, string(request.Body.Email), admin)
	switch {
	case errors.Is(err, account.ErrNameRequired):
		return badAccount("account-needs-a-name", "An account needs a name",
			"It is what names them in the journal, so it cannot be blank."), nil
	case errors.Is(err, account.ErrEmailRequired):
		return badAccount("account-needs-an-address", "An account needs an address",
			"There is no password: the address is how they sign in."), nil
	case errors.Is(err, account.ErrEmailTaken):
		return openapi.CreateAccount409ApplicationProblemPlusJSONResponse(
			problem("address-taken", "That address already has an account", http.StatusConflict,
				"Deactivate the existing account rather than making a second one for the same person."),
		), nil
	case err != nil:
		return nil, err
	}
	return openapi.CreateAccount201JSONResponse(toAPIAccount(made)), nil
}

// DeactivateAccount stops an account from signing in.
func (s *Server) DeactivateAccount(ctx context.Context, request openapi.DeactivateAccountRequestObject) (openapi.DeactivateAccountResponseObject, error) {
	if why, no := s.mayNotOnInstance(ctx, access.ManageAccounts); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.DeactivateAccount401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.DeactivateAccount403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	err := s.account.Deactivate(ctx, request.AccountId)
	if errors.Is(err, account.ErrNotFound) {
		return openapi.DeactivateAccount404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("account"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.DeactivateAccount204Response{}, nil
}

func badAccount(kind, title, detail string) openapi.CreateAccount400ApplicationProblemPlusJSONResponse {
	return openapi.CreateAccount400ApplicationProblemPlusJSONResponse{
		BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
			problem(kind, title, http.StatusBadRequest, detail),
		),
	}
}

func toAPIAccount(a account.Account) openapi.Account {
	out := openapi.Account{
		Id: a.ID, Name: a.Name, Email: openapitypes.Email(a.Email),
		IsAdmin: a.IsAdmin, CreatedAt: a.CreatedAt,
	}
	out.DeactivatedAt = a.DeactivatedAt
	return out
}
