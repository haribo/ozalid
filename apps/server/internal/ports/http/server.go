// Package http exposes the API.
//
// Handler signatures come from the OpenAPI document (backend ADR 0002): the
// generated StrictServerInterface is what Server must satisfy, so a contract
// change that this package has not caught up with fails to compile.
package http

import (
	"context"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/app/catalogue"
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
	catalogue    *catalogue.Service
	intake       *intake.Service
	evidence     *evidence.Service
	session      *session.Service
}

// Compile-time proof that every operation in the contract is implemented.
var _ openapi.StrictServerInterface = (*Server)(nil)

// Deps is what the API needs to answer. Passing them as one struct keeps the
// assembly in cmd/server readable as the surface grows.
type Deps struct {
	Version      string
	Blobs        blobstore.Store
	BlobRecorder BlobRecorder
	Catalogue    *catalogue.Service
	Intake       *intake.Service
	Evidence     *evidence.Service
	Session      *session.Service
}

// New returns a Server wired to deps.
func New(deps Deps) *Server {
	return &Server{
		version:      deps.Version,
		blobs:        deps.Blobs,
		blobRecorder: deps.BlobRecorder,
		catalogue:    deps.Catalogue,
		intake:       deps.Intake,
		evidence:     deps.Evidence,
		session:      deps.Session,
	}
}

// Handler returns the routes described by the contract, mounted under /api.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", openapi.HandlerFromMux(
		openapi.NewStrictHandler(s, nil), http.NewServeMux(),
	)))
	return mux
}
