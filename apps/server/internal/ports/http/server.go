// Package http exposes the API.
//
// Handler signatures come from the OpenAPI document (backend ADR 0002): the
// generated StrictServerInterface is what Server must satisfy, so a contract
// change that this package has not caught up with fails to compile.
package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/adapters/mail"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// BlobRecorder remembers, in the database, what the blob store now holds. The
// bytes stay on disk; only their address and size are recorded.
type BlobRecorder interface {
	RecordBlob(ctx context.Context, hash string, size int64) error
}

// Server implements the generated API interface.
type Server struct {
	version      string
	blobs        blobstore.Store
	blobRecorder BlobRecorder
	tokens       Tokens
	signIn       SignIn
	mail         mail.Sender
	standings    Standings
	// Asking for a sign-in link is the one thing anybody may do without a
	// credential, so it is the one thing that has to be limited.
	// trustProxy says whether X-Forwarded-For may be believed. False by
	// default: trusting it without a proxy in front lets anybody claim any
	// source and walk past the per-source limit.
	trustProxy bool
	perAddress *window
	perSource  *window
	// sending tracks the deliveries still in flight, so a shutdown can wait for
	// them rather than dropping a link somebody is waiting on.
	sending   sync.WaitGroup
	account   *account.Service
	catalogue *catalogue.Service
	intake    *intake.Service
	evidence  *evidence.Service
	session   *session.Service
	comment   *comment.Service
}

// Compile-time proof that every operation in the contract is implemented.
var _ openapi.StrictServerInterface = (*Server)(nil)

// Deps is what the API needs to answer. Passing them as one struct keeps the
// assembly in cmd/server readable as the surface grows.
type Deps struct {
	Version      string
	Blobs        blobstore.Store
	BlobRecorder BlobRecorder
	Tokens       Tokens
	SignIn       SignIn
	Mail         mail.Sender
	Standings    Standings
	TrustProxy   bool
	// Now is the clock the rate limits are measured against. Handed in from
	// cmd/server, which is where the concrete world is wired (backend ADR 0001).
	Now       func() time.Time
	Account   *account.Service
	Catalogue *catalogue.Service
	Intake    *intake.Service
	Evidence  *evidence.Service
	Session   *session.Service
	Comment   *comment.Service
}

// New returns a Server wired to deps.
func New(deps Deps) *Server {
	return &Server{
		version:      deps.Version,
		blobs:        deps.Blobs,
		blobRecorder: deps.BlobRecorder,
		tokens:       deps.Tokens,
		signIn:       deps.SignIn,
		mail:         deps.Mail,
		standings:    deps.Standings,
		trustProxy:   deps.TrustProxy,
		perAddress:   newWindow(perAddress, limitAfter, deps.Now),
		perSource:    newWindow(perSource, limitAfter, deps.Now),
		account:      deps.Account,
		catalogue:    deps.Catalogue,
		intake:       deps.Intake,
		evidence:     deps.Evidence,
		session:      deps.Session,
		comment:      deps.Comment,
	}
}

// Handler returns the routes described by the contract, mounted under /api.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Every route passes through resolution, so no handler can be reached
	// without an answer to "who is this".
	mux.Handle("/api/", http.StripPrefix("/api", withSource(s.trustProxy, withActor(s.tokens, s.signIn, openapi.HandlerFromMux(
		openapi.NewStrictHandler(s, nil), http.NewServeMux(),
	)))))
	// The client is not in here. It ships as its own archive and whatever
	// terminates TLS serves it — a deployment decides its own shape, and an
	// artefact that carried the client would have decided it instead (#103).
	//
	// Said rather than 404'd: an operator pointing a browser at the API and
	// getting a bare "not found" has no way to tell a misconfiguration from a
	// broken server.
	mux.Handle("/", http.HandlerFunc(elsewhere))
	return mux
}

// WaitForSending blocks until every sign-in link in flight has been handed to
// the mail server or given up.
//
// Sending happens off the request's path, so a shutdown that did not wait would
// drop a link somebody is sitting in front of their inbox waiting for.
func (s *Server) WaitForSending() { s.sending.Wait() }

// elsewhere answers anything that is not the API.
//
// This binary serves `/api` and nothing else. The web client is published as
// `ozalid-web-<version>.zip` beside it, and the deployment serves it — from
// nginx, from a CDN, from wherever it likes.
func elsewhere(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(
		"ozalid serves its API under /api.\n\n" +
			"The web client is not in this binary: it ships as ozalid-web-<version>.zip,\n" +
			"and whatever sits in front of this server is what serves it.\n"))
}
