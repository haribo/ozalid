package http_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
	"github.com/haribo/ozalid/internal/contract"
)

// theToken is the one credential these tests present. Uploading capture bytes
// needs a service account (ADR 0018), so the tests carry one — going through
// the real middleware rather than around it.
const theToken = "ozp_thetokenthesetestspresent"

// oneToken knows exactly that token and nothing else.
type oneToken struct{}

func (oneToken) ServiceAccountByToken(_ context.Context, token string) (actor.Actor, bool, error) {
	if token != theToken {
		return actor.Actor{}, false, nil
	}
	return actor.Actor{ID: "a-service-account", Kind: actor.Machine}, true, nil
}

// theProject is the one project the token above reaches. A token belongs to
// exactly one project (ADR 0018), and now that every address names its project,
// these tests go through the real membership check rather than around it.
const theProject = "atlas"

// oneMembership grants write on theProject and nothing anywhere else.
type oneMembership struct{}

// StandingOf answers about the instance: this account administers nothing.
func (oneMembership) StandingOf(context.Context, actor.Actor, string) (access.Standing, error) {
	return access.Standing{}, nil
}

func (oneMembership) StandingOnSlug(_ context.Context, by actor.Actor, slug string) (access.Standing, error) {
	if by.Zero() || slug != theProject {
		return access.Standing{}, nil
	}
	return access.Standing{Rights: access.Member}, nil
}

// oneCapture resolves a single capture id to whatever address the test put
// behind it. Reading bytes goes through a capture now (product.md §8.1), so
// these tests need a row to read through.
type oneCapture struct{ id, hash string }

func (c oneCapture) CaseGrid(context.Context, string, string, *string) (evidence.Grid, error) {
	return evidence.Grid{}, nil
}

func (c oneCapture) CaptureBlob(_ context.Context, slug, captureID string) (string, error) {
	if slug != theProject || captureID != c.id {
		return "", catalogue.ErrNotFound
	}
	return c.hash, nil
}

func (c oneCapture) RecordingBlob(context.Context, string, string) (string, error) {
	return "", catalogue.ErrNotFound
}

// serverHolding wires a server whose one capture points at hash.
func serverHolding(t *testing.T, hash string) http.Handler {
	t.Helper()
	store, err := blobstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	return ozhttp.New(ozhttp.Deps{
		Version: "test", Now: time.Now, Blobs: store, Tokens: oneToken{}, Standings: oneMembership{},
		Evidence: evidence.New(oneCapture{id: theCapture, hash: hash}),
	}).Handler()
}

// theCapture is the one capture id these tests read through.
const theCapture = "cap-the-one"

func newServer(t *testing.T) http.Handler {
	t.Helper()
	store, err := blobstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	return ozhttp.New(ozhttp.Deps{Version: "test", Now: time.Now, Blobs: store, Tokens: oneToken{}, Standings: oneMembership{}}).Handler()
}

func do(t *testing.T, h http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	return doAs(t, h, method, path, body, "Bearer "+theToken)
}

// doAs makes a request with whatever the caller wants in Authorization.
func doAs(t *testing.T, h http.Handler, method, path string, body io.Reader, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUploadingCaptureBytesNeedsAToken(t *testing.T) {
	h := newServer(t)
	content := []byte("a capture")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	for _, c := range []struct {
		name          string
		authorization string
	}{
		{"nothing at all", ""},
		{"a token nobody minted", "Bearer ozp_neverminted"},
		{"the right token, no scheme", theToken},
		{"another scheme", "Basic " + theToken},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := doAs(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, bytes.NewReader(content), c.authorization)
			if got.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", got.Code)
			}
		})
	}

	// HEAD is closed the same way, and for the same reason: asking whether the
	// store already holds given bytes is part of pushing to a project, so it
	// takes the same membership (#71).
	if got := doAs(t, h, http.MethodHead, "/api/projects/"+theProject+"/blobs/"+hash, nil, ""); got.Code != http.StatusUnauthorized {
		t.Errorf("HEAD without a credential = %d, want 401", got.Code)
	}
}

func TestUploadingTheSameContentTwiceIsIdempotent(t *testing.T) {
	h := newServer(t)
	content := []byte("a capture")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	first := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, bytes.NewReader(content))
	if first.Code != http.StatusCreated {
		t.Fatalf("first upload = %d, want 201", first.Code)
	}

	// The store already holds it: nothing to do, and the client is told so.
	second := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, bytes.NewReader(content))
	if second.Code != http.StatusNoContent {
		t.Errorf("second upload = %d, want 204", second.Code)
	}
}

func TestContentIsRefusedWhenItDoesNotMatchItsAddress(t *testing.T) {
	h := newServer(t)
	announced, _, _ := contract.HashReader(strings.NewReader("what was promised"))

	rec := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+announced, strings.NewReader("something else"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Errorf("content-type = %q, want problem+json", ct)
	}
	if !strings.Contains(rec.Body.String(), "content-address-mismatch") {
		t.Errorf("body = %q, want it to name the problem type", rec.Body.String())
	}

	// And the refused content must not have landed anywhere.
	if head := do(t, h, http.MethodHead, "/api/projects/"+theProject+"/blobs/"+announced, nil); head.Code != http.StatusNotFound {
		t.Errorf("HEAD after a refused upload = %d, want 404", head.Code)
	}
}

func TestAMalformedAddressIsRefusedOnEveryOperation(t *testing.T) {
	h := newServer(t)

	for _, bad := range []string{
		"sha256:tooshort",
		"md5:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		"sha256:" + strings.Repeat("Z", 64),
	} {
		if rec := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+bad, strings.NewReader("x")); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %q = %d, want 400", bad, rec.Code)
		}
	}
}

func TestReadingACaptureReturnsExactlyWhatWasUploaded(t *testing.T) {
	content := []byte("the evidence itself, bytes and all\x00\xff")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))
	h := serverHolding(t, hash)

	if rec := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, bytes.NewReader(content)); rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d, want 201", rec.Code)
	}

	// Read through the capture, never through the address: a hash names no
	// project and cannot be authorised (product.md §8.1).
	rec := do(t, h, http.MethodGet, "/api/projects/"+theProject+"/captures/"+theCapture, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("reading the capture = %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), content) {
		t.Error("the bytes read are not the bytes uploaded")
	}
}

func TestACaptureFromAnotherProjectIsNotFound(t *testing.T) {
	content := []byte("somebody else's evidence")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))
	h := serverHolding(t, hash)

	if rec := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, bytes.NewReader(content)); rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d, want 201", rec.Code)
	}

	// The bytes are in the store, and this caller may not have them: the
	// capture is not theirs. Not found rather than forbidden — a refusal would
	// confirm the capture exists elsewhere.
	rec := do(t, h, http.MethodGet, "/api/projects/"+theProject+"/captures/some-other-capture", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("reading a capture that is not there = %d, want 404", rec.Code)
	}
}

func TestAskingForContentTheStoreDoesNotHold(t *testing.T) {
	h := newServer(t)
	absent, _, _ := contract.HashReader(strings.NewReader("never uploaded"))

	if rec := do(t, h, http.MethodHead, "/api/projects/"+theProject+"/blobs/"+absent, nil); rec.Code != http.StatusNotFound {
		t.Errorf("HEAD = %d, want 404", rec.Code)
	}
}

func TestACaptureWhoseBytesAreGoneIsNotFound(t *testing.T) {
	// The row survived and the store did not. Nothing the caller can act on,
	// so they are told the same thing as for a capture that is not theirs.
	absent, _, _ := contract.HashReader(strings.NewReader("never uploaded"))
	h := serverHolding(t, absent)

	rec := do(t, h, http.MethodGet, "/api/projects/"+theProject+"/captures/"+theCapture, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("reading it = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not-found") {
		t.Errorf("body = %q, want it to name the problem type", rec.Body.String())
	}
}

func TestAnAlreadyHeldAddressShortCircuitsWithoutReadingTheBody(t *testing.T) {
	content := []byte("a capture")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))
	h := serverHolding(t, hash)

	if rec := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, bytes.NewReader(content)); rec.Code != http.StatusCreated {
		t.Fatalf("first upload = %d, want 201", rec.Code)
	}

	// Documented behaviour, not an oversight: the address is already held, so
	// the body is never read and the mismatch is not reported. The store still
	// holds the right bytes; verifying would mean reading every upload in full.
	rec := do(t, h, http.MethodPut, "/api/projects/"+theProject+"/blobs/"+hash, strings.NewReader("entirely different bytes"))
	if rec.Code != http.StatusNoContent {
		t.Errorf("upload over a held address = %d, want 204", rec.Code)
	}

	// What matters is that the stored content did not change.
	got := do(t, h, http.MethodGet, "/api/projects/"+theProject+"/captures/"+theCapture, nil)
	if !bytes.Equal(got.Body.Bytes(), content) {
		t.Error("the stored content changed under a mismatched upload")
	}
}
