package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
	"github.com/haribo/ozalid/internal/contract"
)

// HeadBlob reports whether the store already holds this content.
//
// Intake calls it before sending anything: content already held is never
// uploaded again (ADR 0004).
func (s *Server) HeadBlob(ctx context.Context, request openapi.HeadBlobRequestObject) (openapi.HeadBlobResponseObject, error) {
	if !contract.ValidHash(request.Hash) {
		return openapi.HeadBlob400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: badAddress(),
		}, nil
	}

	held, err := s.blobs.Exists(ctx, request.Hash)
	if err != nil {
		return nil, err
	}
	if !held {
		return openapi.HeadBlob404Response{}, nil
	}
	return openapi.HeadBlob200Response{}, nil
}

// GetBlob streams the content at that address.
func (s *Server) GetBlob(ctx context.Context, request openapi.GetBlobRequestObject) (openapi.GetBlobResponseObject, error) {
	if !contract.ValidHash(request.Hash) {
		return openapi.GetBlob400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: badAddress(),
		}, nil
	}

	rc, err := s.blobs.Get(ctx, request.Hash)
	if errors.Is(err, blobstore.ErrNotFound) {
		return openapi.GetBlob404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: openapi.NotFoundApplicationProblemPlusJSONResponse(
				problem("blob-not-found", "No content at that address", http.StatusNotFound, ""),
			),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	// The strict handler owns the response from here; closing is its job once
	// the body has been streamed.
	return openapi.GetBlob200ApplicationoctetStreamResponse{Body: rc}, nil
}

// PutBlob stores content under its address.
//
// Idempotent by construction: an address always denotes the same bytes, so a
// second write of content already held is a no-op.
func (s *Server) PutBlob(ctx context.Context, request openapi.PutBlobRequestObject) (openapi.PutBlobResponseObject, error) {
	// A token, and nothing more. Content addressing makes the store shared
	// across projects by design (ADR 0004), so there is no project here to be a
	// member of — the meaningful bar is holding a token at all.
	if !isKnown(ctx) {
		return openapi.PutBlob401ApplicationProblemPlusJSONResponse{
			UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(unknownCaller()),
		}, nil
	}

	if !contract.ValidHash(request.Hash) {
		return openapi.PutBlob400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: badAddress(),
		}, nil
	}

	// Deliberate short circuit: an address always denotes the same bytes, so
	// content already held needs neither reading nor verifying. It costs one
	// stat instead of hashing however many megabytes the client is sending.
	//
	// The consequence is that a client sending the wrong bytes under an
	// address the store already holds is answered 204, not 422 — the store
	// still contains the right content, so nothing is at risk, but the client's
	// mistake goes unreported. Catching it would mean reading every upload in
	// full, which is exactly what this saves. Clients call HEAD first.
	held, err := s.blobs.Exists(ctx, request.Hash)
	if err != nil {
		return nil, err
	}
	if held {
		return openapi.PutBlob204Response{}, nil
	}

	err = s.blobs.Put(ctx, request.Hash, request.Body)
	if errors.Is(err, blobstore.ErrHashMismatch) {
		// Not a client formatting error but a truth error: the bytes are not
		// what the address says they are.
		slog.WarnContext(ctx, "refused content that does not match its address", "hash", request.Hash)
		return openapi.PutBlob422ApplicationProblemPlusJSONResponse(
			problem("content-address-mismatch", "The content does not match its address",
				http.StatusUnprocessableEntity,
				"The bytes received do not hash to the address they were sent under."),
		), nil
	}
	if err != nil {
		return nil, err
	}

	// The database holds what it needs to know about the bytes on disk: their
	// address and their size. The bytes themselves never go near it
	// (backend ADR 0004).
	if s.blobRecorder != nil {
		size, err := s.blobs.Size(ctx, request.Hash)
		if err != nil {
			return nil, err
		}
		if err := s.blobRecorder.RecordBlob(ctx, request.Hash, size); err != nil {
			return nil, err
		}
	}
	return openapi.PutBlob201Response{}, nil
}

// badAddress is the one malformed-address payload, shared by every operation
// that takes a content address.
func badAddress() openapi.BadRequestApplicationProblemPlusJSONResponse {
	return openapi.BadRequestApplicationProblemPlusJSONResponse(
		problem("bad-content-address", "Malformed content address", http.StatusBadRequest,
			"An address is 'sha256:' followed by 64 lowercase hexadecimal characters."),
	)
}

// problem builds an RFC 9457 payload. Every failure the API reports carries
// this shape, so a client never has to guess what an error looks like.
func problem(kind, title string, status int, detail string) openapi.Problem {
	p := openapi.Problem{
		Type:   "https://ozalid.dev/problems/" + kind,
		Title:  title,
		Status: status,
	}
	if detail != "" {
		p.Detail = &detail
	}
	return p
}
