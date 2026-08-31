package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
	"github.com/haribo/ozalid/internal/contract"
)

// HeadBlob reports whether the store already holds this content.
//
// Intake calls it before sending anything: content already held is never
// uploaded again (ADR 0004).
func (s *Server) HeadBlob(ctx context.Context, request openapi.HeadBlobRequestObject) (openapi.HeadBlobResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.HeadBlob401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.HeadBlob403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}
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

// GetCaptureImage streams the image one capture holds.
//
// Reached through the capture rather than through the content address: a hash
// names no project, and identical bytes are stored once for every project that
// references them (ADR 0004), so there is nothing in an address to check a
// membership against (product.md §8.1).
func (s *Server) GetCaptureImage(ctx context.Context, request openapi.GetCaptureImageRequestObject) (openapi.GetCaptureImageResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.ReadProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.GetCaptureImage401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.GetCaptureImage403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	rc, err := s.bytesOf(ctx, func() (string, error) {
		return s.evidence.CaptureBlob(ctx, request.Slug, request.CaptureId)
	})
	if errors.Is(err, errNoBytes) {
		return openapi.GetCaptureImage404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("capture"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	// The strict handler owns the response from here; closing is its job once
	// the body has been streamed.
	return openapi.GetCaptureImage200ImagepngResponse{Body: rc}, nil
}

// GetRecordingVideo streams the video one recording holds.
func (s *Server) GetRecordingVideo(ctx context.Context, request openapi.GetRecordingVideoRequestObject) (openapi.GetRecordingVideoResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.ReadProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.GetRecordingVideo401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.GetRecordingVideo403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}

	rc, err := s.bytesOf(ctx, func() (string, error) {
		return s.evidence.RecordingBlob(ctx, request.Slug, request.RecordingId)
	})
	if errors.Is(err, errNoBytes) {
		return openapi.GetRecordingVideo404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("recording"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.GetRecordingVideo200ApplicationoctetStreamResponse{Body: rc}, nil
}

// errNoBytes covers both ways bytes fail to arrive: the row is not in this
// project, or the store lost what the row points at. The caller is told the
// same thing either way — which of the two it was is not their business, and
// telling them would map the instance.
var errNoBytes = errors.New("no bytes at that address")

// bytesOf resolves a row to a stored address and opens it.
func (s *Server) bytesOf(ctx context.Context, resolve func() (string, error)) (io.ReadCloser, error) {
	hash, err := resolve()
	if errors.Is(err, app.ErrNotFound) {
		return nil, errNoBytes
	}
	if err != nil {
		return nil, err
	}

	rc, err := s.blobs.Get(ctx, hash)
	if errors.Is(err, blobstore.ErrNotFound) {
		// The row survived and the bytes did not. Nothing a caller can act on,
		// but the operator must see it: this is the store losing evidence.
		slog.ErrorContext(ctx, "a stored address has no bytes behind it", "hash", hash)
		return nil, errNoBytes
	}
	if err != nil {
		return nil, err
	}
	return rc, nil
}

// PutBlob stores content under its address.
//
// Idempotent by construction: an address always denotes the same bytes, so a
// second write of content already held is a no-op.
func (s *Server) PutBlob(ctx context.Context, request openapi.PutBlobRequestObject) (openapi.PutBlobResponseObject, error) {
	// The store is shared across projects by design (ADR 0004), but the upload
	// is not: it is done on behalf of the project being pushed to, and that is
	// what the caller must be a member of (#71). "Holding a token at all" was
	// the bar while no address named a project.
	if why, no := s.mayNot(ctx, request.Slug, access.WriteProject); no {
		if why.Status != http.StatusUnauthorized {
			return openapi.PutBlob403ApplicationProblemPlusJSONResponse{
				ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.PutBlob401ApplicationProblemPlusJSONResponse{
			UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
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
