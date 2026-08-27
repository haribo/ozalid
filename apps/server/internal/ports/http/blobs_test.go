package http_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
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

func newServer(t *testing.T) http.Handler {
	t.Helper()
	store, err := blobstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	return ozhttp.New(ozhttp.Deps{Version: "test", Blobs: store, Tokens: oneToken{}}).Handler()
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
			got := doAs(t, h, http.MethodPut, "/api/blobs/"+hash, bytes.NewReader(content), c.authorization)
			if got.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", got.Code)
			}
		})
	}

	// And reading needs nothing, which is where sixteen of the eighteen
	// endpoints still stand until sign-in exists.
	if got := doAs(t, h, http.MethodHead, "/api/blobs/"+hash, nil, ""); got.Code == http.StatusUnauthorized {
		t.Error("reading was refused; only the two writing endpoints are closed so far")
	}
}

func TestUploadingTheSameContentTwiceIsIdempotent(t *testing.T) {
	h := newServer(t)
	content := []byte("a capture")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	first := do(t, h, http.MethodPut, "/api/blobs/"+hash, bytes.NewReader(content))
	if first.Code != http.StatusCreated {
		t.Fatalf("first upload = %d, want 201", first.Code)
	}

	// The store already holds it: nothing to do, and the client is told so.
	second := do(t, h, http.MethodPut, "/api/blobs/"+hash, bytes.NewReader(content))
	if second.Code != http.StatusNoContent {
		t.Errorf("second upload = %d, want 204", second.Code)
	}
}

func TestContentIsRefusedWhenItDoesNotMatchItsAddress(t *testing.T) {
	h := newServer(t)
	announced, _, _ := contract.HashReader(strings.NewReader("what was promised"))

	rec := do(t, h, http.MethodPut, "/api/blobs/"+announced, strings.NewReader("something else"))
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
	if head := do(t, h, http.MethodHead, "/api/blobs/"+announced, nil); head.Code != http.StatusNotFound {
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
		if rec := do(t, h, http.MethodPut, "/api/blobs/"+bad, strings.NewReader("x")); rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %q = %d, want 400", bad, rec.Code)
		}
		if rec := do(t, h, http.MethodGet, "/api/blobs/"+bad, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %q = %d, want 400", bad, rec.Code)
		}
	}
}

func TestDownloadReturnsExactlyWhatWasUploaded(t *testing.T) {
	h := newServer(t)
	content := []byte("the evidence itself, bytes and all\x00\xff")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	if rec := do(t, h, http.MethodPut, "/api/blobs/"+hash, bytes.NewReader(content)); rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d, want 201", rec.Code)
	}

	rec := do(t, h, http.MethodGet, "/api/blobs/"+hash, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("download = %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), content) {
		t.Error("the bytes downloaded are not the bytes uploaded")
	}
}

func TestAskingForContentTheStoreDoesNotHold(t *testing.T) {
	h := newServer(t)
	absent, _, _ := contract.HashReader(strings.NewReader("never uploaded"))

	if rec := do(t, h, http.MethodHead, "/api/blobs/"+absent, nil); rec.Code != http.StatusNotFound {
		t.Errorf("HEAD = %d, want 404", rec.Code)
	}
	rec := do(t, h, http.MethodGet, "/api/blobs/"+absent, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "blob-not-found") {
		t.Errorf("body = %q, want it to name the problem type", rec.Body.String())
	}
}

func TestAnAlreadyHeldAddressShortCircuitsWithoutReadingTheBody(t *testing.T) {
	h := newServer(t)
	content := []byte("a capture")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	if rec := do(t, h, http.MethodPut, "/api/blobs/"+hash, bytes.NewReader(content)); rec.Code != http.StatusCreated {
		t.Fatalf("first upload = %d, want 201", rec.Code)
	}

	// Documented behaviour, not an oversight: the address is already held, so
	// the body is never read and the mismatch is not reported. The store still
	// holds the right bytes; verifying would mean reading every upload in full.
	rec := do(t, h, http.MethodPut, "/api/blobs/"+hash, strings.NewReader("entirely different bytes"))
	if rec.Code != http.StatusNoContent {
		t.Errorf("upload over a held address = %d, want 204", rec.Code)
	}

	// What matters is that the stored content did not change.
	got := do(t, h, http.MethodGet, "/api/blobs/"+hash, nil)
	if !bytes.Equal(got.Body.Bytes(), content) {
		t.Error("the stored content changed under a mismatched upload")
	}
}
