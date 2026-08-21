// Package http exposes the API.
//
// Handler signatures come from the OpenAPI document (backend ADR 0002): the
// generated StrictServerInterface is what Server must satisfy, so a contract
// change that this package has not caught up with fails to compile.
package http

import (
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// Server implements the generated API interface.
type Server struct {
	version   string
	blobs     blobstore.Store
	catalogue *catalogue.Service
}

// Compile-time proof that every operation in the contract is implemented.
var _ openapi.StrictServerInterface = (*Server)(nil)

// New returns a Server reporting the given build version, storing capture
// bytes in blobs and reading the catalogue through cat.
func New(version string, blobs blobstore.Store, cat *catalogue.Service) *Server {
	return &Server{version: version, blobs: blobs, catalogue: cat}
}

// Handler returns the routes described by the contract, mounted under /api.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", openapi.HandlerFromMux(
		openapi.NewStrictHandler(s, nil), http.NewServeMux(),
	)))
	return mux
}
