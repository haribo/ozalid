package webui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/ports/http/webui"
)

// cacheOn is what the client's handler says an answer may be kept for.
func cacheOn(t *testing.T, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	webui.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Header().Get("Cache-Control")
}

func TestTheEntryPointIsAlwaysRevalidated(t *testing.T) {
	// It names the hashed assets, so a stale copy points at a bundle that is
	// gone. Everything answered *with* it is the same file under another name
	// and must say the same thing.
	// Not "/index.html": http.FileServer redirects it to "/", which is Go's
	// behaviour and not this package's to assert.
	for _, asked := range []string{"/", "/projects/atlas", "/cases/abc123", "/accounts"} {
		code, cache := cacheOn(t, asked)
		if code != http.StatusOK {
			t.Errorf("%s = %d, want 200", asked, code)
			continue
		}
		if cache != "no-cache" {
			t.Errorf("%s carries %q, want no-cache", asked, cache)
		}
	}
}

func TestAMissingAssetIsNotCachedAtAll(t *testing.T) {
	// A 404 must not be kept: cached for a year, a browser would go on
	// believing a file is absent long after a deployment shipped it.
	code, cache := cacheOn(t, "/assets/index-Whatever.js")
	if code != http.StatusNotFound {
		t.Errorf("a missing asset = %d, want 404", code)
	}
	if cache != "" {
		t.Errorf("a missing asset carries %q, want nothing", cache)
	}
}

func TestAMissingFileIsStillNotFound(t *testing.T) {
	// Caching must not have turned a 404 into an answer: a request for
	// JavaScript answered with a document fails on the first `<`, with a
	// message that says nothing about the cause.
	if code, _ := cacheOn(t, "/nothing-here.js"); code != http.StatusNotFound {
		t.Errorf("a missing file = %d, want 404", code)
	}
}
