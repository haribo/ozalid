package webui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/ports/http/webui"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	webui.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestARouteOfTheClientIsAnsweredWithTheClient(t *testing.T) {
	// A browser loading /cases/abc123 directly owns that route, not the server.
	got := get(t, "/cases/abc123")
	if got.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", got.Code)
	}
	if ct := got.Header().Get("Content-Type"); ct != "" && ct[:9] != "text/html" {
		t.Errorf("content type = %q, want html", ct)
	}
}

func TestAMissingFileIsNotAnsweredWithHTML(t *testing.T) {
	// Handing back a document where JavaScript was asked for makes the browser
	// fail on the first `<`, with a message that says nothing about the cause.
	for _, path := range []string{"/assets/never-built.js", "/favicon.svg", "/a/b/c.css"} {
		t.Run(path, func(t *testing.T) {
			if got := get(t, path); got.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404", got.Code)
			}
		})
	}
}

func TestTheRootIsTheEntryPoint(t *testing.T) {
	if got := get(t, "/"); got.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", got.Code)
	}
}

func TestABinaryWithoutTheClientSaysSo(t *testing.T) {
	// This test runs against the committed placeholder, which is what a
	// checkout has when npm never ran. The image replaces it, and there the
	// answer is the opposite — which is why the server logs it at startup
	// rather than leaving it to be discovered as a blank page.
	if webui.Built() {
		t.Error("Built() is true against the placeholder; it can no longer tell the two apart")
	}
}
