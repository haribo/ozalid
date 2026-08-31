package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
)

// theKnown is the one address that has an account here.
const theKnown = "nina@example.test"

// slowMail takes as long as a real mail server on a bad day. It is what makes
// the difference between waiting for the send and not waiting measurable.
type slowMail struct {
	took chan time.Duration
}

func (m slowMail) SendSignInLink(_ context.Context, _, _ string) error {
	time.Sleep(sendTakes)
	select {
	case m.took <- sendTakes:
	default:
	}
	return nil
}

const sendTakes = 250 * time.Millisecond

// oneAccount knows theKnown and nobody else.
type oneAccount struct{}

func (oneAccount) StartSignIn(_ context.Context, email string) (string, bool, error) {
	return "a-link", strings.EqualFold(email, theKnown), nil
}
func (oneAccount) ClaimSignIn(context.Context, string) (string, bool, error) { return "", false, nil }
func (oneAccount) UserBySession(context.Context, string) (actor.Actor, bool, error) {
	return actor.Actor{}, false, nil
}
func (oneAccount) EndSession(context.Context, string) error { return nil }
func (oneAccount) Person(context.Context, string) (account.Person, bool, error) {
	return account.Person{}, false, nil
}

func signInServer(t *testing.T) (http.Handler, chan time.Duration) {
	t.Helper()
	store, err := blobstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	sent := make(chan time.Duration, 4)
	return ozhttp.New(ozhttp.Deps{
		Version: "test", Now: time.Now, Blobs: store, Tokens: oneToken{}, Standings: oneMembership{},
		SignIn: oneAccount{}, Mail: slowMail{took: sent},
	}).Handler(), sent
}

func askFor(t *testing.T, h http.Handler, email string) (*httptest.ResponseRecorder, time.Duration) {
	t.Helper()
	started := time.Now()
	rec := doAs(t, h, http.MethodPost, "/api/sign-in",
		strings.NewReader(`{"email":"`+email+`"}`), "")
	return rec, time.Since(started)
}

func TestAskingForALinkTellsNobodyWhetherTheAddressIsKnown(t *testing.T) {
	h, sent := signInServer(t)

	// The address with an account, and one without. Both answer 202, which was
	// already true — what this asserts is that they answer at the same speed.
	known, tookKnown := askFor(t, h, theKnown)
	unknown, tookUnknown := askFor(t, h, "nobody@example.test")

	if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted {
		t.Fatalf("statuses = %d and %d, want 202 for both", known.Code, unknown.Code)
	}

	// Neither waited for the send. A synchronous send would put `sendTakes` on
	// the known one and nothing on the other, which is exactly the difference a
	// stopwatch reads.
	for name, took := range map[string]time.Duration{"known": tookKnown, "unknown": tookUnknown} {
		if took >= sendTakes {
			t.Errorf("the %s address took %v, which is at least the %v the send takes: the answer waits for it",
				name, took, sendTakes)
		}
	}

	// And the link really was sent, so the speed above is not the send being
	// skipped.
	select {
	case <-sent:
	case <-time.After(2 * time.Second):
		t.Error("no link was ever sent, so the timing above proves nothing")
	}
}

func TestAskingTooOftenForOneAddressIsRefused(t *testing.T) {
	h, _ := signInServer(t)

	// Three links inside one window are the three that can be good at once; a
	// fourth is not somebody who needs a link.
	for i := range 3 {
		if rec, _ := askFor(t, h, theKnown); rec.Code != http.StatusAccepted {
			t.Fatalf("attempt %d = %d, want 202", i+1, rec.Code)
		}
	}

	rec, _ := askFor(t, h, theKnown)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("the fourth = %d, want 429", rec.Code)
	}
	after := rec.Header().Get("Retry-After")
	if seconds, err := strconv.Atoi(after); err != nil || seconds <= 0 {
		t.Errorf("Retry-After = %q, want a positive number of seconds", after)
	}
	if body := rec.Body.String(); !strings.Contains(body, "too-many-sign-ins") {
		t.Errorf("body = %q, want it to name the problem type", body)
	}
}

func TestOneAddressBeingFloodedDoesNotUseUpAnother(t *testing.T) {
	h, _ := signInServer(t)

	for range 3 {
		askFor(t, h, theKnown)
	}
	if rec, _ := askFor(t, h, theKnown); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("the address under attack = %d, want it refused", rec.Code)
	}

	// Somebody else can still sign in. The per-address limit protects one
	// person; spending it on their behalf would be the flood succeeding.
	if rec, _ := askFor(t, h, "someone.else@example.test"); rec.Code != http.StatusAccepted {
		t.Errorf("another address = %d, want 202", rec.Code)
	}
}

func TestWalkingAListOfAddressesRunsOutOfSource(t *testing.T) {
	h, _ := signInServer(t)

	// Twenty attempts fit; the twenty-first does not, whatever address it
	// names. This is what makes learning which addresses have accounts slow.
	refused := false
	for i := range 25 {
		rec, _ := askFor(t, h, "probe-"+strconv.Itoa(i)+"@example.test")
		if rec.Code == http.StatusTooManyRequests {
			refused = true
			if i < 20 {
				t.Fatalf("refused at attempt %d, want at least 20 to pass", i+1)
			}
			break
		}
	}
	if !refused {
		t.Error("twenty-five attempts from one source all passed, so nothing limits an enumeration")
	}
}
