package http_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
)

// atNoon is a clock a test moves by hand, so a window can turn over without
// anybody waiting fifteen minutes for it.
type atNoon struct{ at time.Time }

func (c *atNoon) now() time.Time { return c.at }

func TestAWindowTurnsOverAndForgets(t *testing.T) {
	clock := &atNoon{at: time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)}
	store, err := blobstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	h := ozhttp.New(ozhttp.Deps{
		Version: "test", Blobs: store, Tokens: oneToken{}, Standings: oneMembership{},
		SignIn: oneAccount{}, Mail: slowMail{took: make(chan time.Duration, 4)},
		Now: clock.now,
	}).Handler()

	ask := func() int {
		return doAs(t, h, http.MethodPost, "/api/sign-in",
			strings.NewReader(`{"email":"`+theKnown+`"}`), "").Code
	}

	for i := range 3 {
		if got := ask(); got != http.StatusAccepted {
			t.Fatalf("attempt %d = %d, want 202", i+1, got)
		}
	}
	if got := ask(); got != http.StatusTooManyRequests {
		t.Fatalf("the fourth = %d, want 429", got)
	}

	// Fourteen minutes later the window has not turned over: a limit that
	// forgets early is a limit an attacker waits out.
	clock.at = clock.at.Add(14 * time.Minute)
	if got := ask(); got != http.StatusTooManyRequests {
		t.Errorf("after 14 minutes = %d, want it still refused", got)
	}

	// Sixteen, and it has.
	clock.at = clock.at.Add(2 * time.Minute)
	if got := ask(); got != http.StatusAccepted {
		t.Errorf("after 16 minutes = %d, want 202", got)
	}
}
