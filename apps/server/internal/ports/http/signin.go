package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// sessionCookie is what the browser carries.
//
// HttpOnly so no script can read it, SameSite=Lax so another site cannot spend
// it, Secure so it never crosses plain HTTP. The path is the root because the
// client is served from there and calls /api from the same origin.
const sessionCookie = "ozalid_session"

// SignIn is what the API needs to let a person in.
type SignIn interface {
	StartSignIn(ctx context.Context, email string) (link string, sendIt bool, err error)
	ClaimSignIn(ctx context.Context, link string) (session string, ok bool, err error)
	UserBySession(ctx context.Context, token string) (actor.Actor, bool, error)
	EndSession(ctx context.Context, token string) error
	Person(ctx context.Context, id string) (account.Person, bool, error)
}

// RequestSignIn sends a link to an address.
//
// The answer never says whether the address is known. Whether somebody has an
// account here is not something a stranger gets to learn by asking, and an
// answer that differed would tell them.
func (s *Server) RequestSignIn(ctx context.Context, request openapi.RequestSignInRequestObject) (openapi.RequestSignInResponseObject, error) {
	email := strings.TrimSpace(string(request.Body.Email))
	if email == "" || !strings.Contains(email, "@") {
		return openapi.RequestSignIn400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("no-address", "An address is needed", http.StatusBadRequest,
					"Sign-in sends a link, so it needs somewhere to send it."),
			),
		}, nil
	}

	link, send, err := s.signIn.StartSignIn(ctx, email)
	if err != nil {
		return nil, err
	}
	if send {
		if err := s.mail.SendSignInLink(ctx, email, link); err != nil {
			// The link exists and the message did not leave. Reported here
			// because an operator has to see it, and answered the same way
			// outside because the caller must not learn that the address is
			// known.
			slog.Error("the sign-in link could not be sent", "error", err)
		}
	}
	return openapi.RequestSignIn202Response{}, nil
}

// ClaimSignIn spends a link and sets the session cookie.
func (s *Server) ClaimSignIn(ctx context.Context, request openapi.ClaimSignInRequestObject) (openapi.ClaimSignInResponseObject, error) {
	session, ok, err := s.signIn.ClaimSignIn(ctx, request.Body.Link)
	if err != nil {
		return nil, err
	}
	if !ok {
		return openapi.ClaimSignIn401ApplicationProblemPlusJSONResponse{
			UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(
				problem("link-spent", "This link no longer works", http.StatusUnauthorized,
					"A sign-in link works once and for a few minutes. Ask for another."),
			),
		}, nil
	}
	return signedIn{session: session}, nil
}

// SignOut forgets this browser's session, and no other.
func (s *Server) SignOut(ctx context.Context, _ openapi.SignOutRequestObject) (openapi.SignOutResponseObject, error) {
	if token := sessionFrom(ctx); token != "" {
		if err := s.signIn.EndSession(ctx, token); err != nil {
			return nil, err
		}
	}
	// Signed out and was not signed in give the same answer: there is nothing
	// useful to learn from the difference.
	return signedOut{}, nil
}

// WhoAmI answers what the client needs to decide between a review and a form.
func (s *Server) WhoAmI(ctx context.Context, _ openapi.WhoAmIRequestObject) (openapi.WhoAmIResponseObject, error) {
	by := actorFrom(ctx)
	if by.Kind != actor.Human || by.ID == anonymous.ID || by.Zero() {
		return openapi.WhoAmI401ApplicationProblemPlusJSONResponse{
			UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(
				problem("not-signed-in", "Nobody is signed in", http.StatusUnauthorized, ""),
			),
		}, nil
	}

	person, ok, err := s.signIn.Person(ctx, by.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		// The session resolved and the account is gone: deactivated between one
		// request and the next.
		return openapi.WhoAmI401ApplicationProblemPlusJSONResponse{
			UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(
				problem("not-signed-in", "Nobody is signed in", http.StatusUnauthorized, ""),
			),
		}, nil
	}
	return openapi.WhoAmI200JSONResponse{
		Id: person.ID, Name: person.Name, Email: person.Email, IsAdmin: person.IsAdmin,
	}, nil
}

// signedIn writes the session cookie beside the empty response.
type signedIn struct{ session string }

func (r signedIn) VisitClaimSignInResponse(w http.ResponseWriter) error {
	http.SetCookie(w, cookie(r.session, int(credential.SessionLifetime.Seconds())))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// signedOut clears it.
type signedOut struct{}

func (signedOut) VisitSignOutResponse(w http.ResponseWriter) error {
	http.SetCookie(w, cookie("", -1))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
