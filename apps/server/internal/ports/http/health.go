package http

import (
	"context"

	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// GetHealth reports that the process is able to serve requests.
//
// It deliberately checks nothing else: a readiness probe that also pings the
// database turns a slow query into a restart loop.
func (s *Server) GetHealth(_ context.Context, _ openapi.GetHealthRequestObject) (openapi.GetHealthResponseObject, error) {
	return openapi.GetHealth200JSONResponse{
		Status:  openapi.Ok,
		Version: s.version,
	}, nil
}
