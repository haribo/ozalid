package http_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// spec is the document the server was generated from, read back out of the
// generated code so the test and the handler cannot be looking at two
// different contracts.
func spec() (*openapi3.T, error) { return openapi.GetSpec() }

// TestEveryOperationThatDeclaresSecurityEnforcesIt reads the OpenAPI document
// and probes the real handler with no credential at all.
//
// It reads the document rather than restating it, deliberately (#55). A list
// written here would be a second copy of the contract, and the two would drift
// the first time somebody added an endpoint — which is exactly the failure this
// exists to catch. An operation that declares `security` and answers anything
// but 401 to an anonymous caller fails here, and so does a new endpoint added
// with a guard forgotten.
func TestEveryOperationThatDeclaresSecurityEnforcesIt(t *testing.T) {
	doc, err := spec()
	if err != nil {
		t.Fatalf("reading the API document: %v", err)
	}
	h := newServer(t)

	probed := 0
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op.Security == nil {
				continue
			}
			probed++
			t.Run(method+" "+path, func(t *testing.T) {
				// Dummy path values: the guard runs before anything looks the
				// row up, so what they name never has to exist.
				target := "/api" + strings.NewReplacer(
					"{slug}", "a-project",
					"{caseId}", "a-case",
					"{commentId}", "a-comment",
					"{categoryId}", "a-category",
					"{captureId}", "a-capture",
					"{recordingId}", "a-recording",
					"{hash}", "sha256:"+strings.Repeat("0", 64),
				).Replace(path)

				// An empty JSON object satisfies the decoder for the operations
				// that take a body, so a 400 never stands in for a 401.
				got := doAs(t, h, method, target, strings.NewReader("{}"), "")
				if got.Code != http.StatusUnauthorized {
					t.Errorf("anonymous %s %s = %d, want 401 — the document declares security, the handler does not enforce it",
						method, path, got.Code)
				}
			})
		}
	}

	if probed == 0 {
		t.Fatal("no operation declares security, so this test proved nothing")
	}
	t.Logf("%d operations probed", probed)
}
