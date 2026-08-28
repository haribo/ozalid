package postgres_test

import (
	"context"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
)

func TestAnAddressNobodyHasProducesNoLinkAndNoError(t *testing.T) {
	// Whether somebody has an account here is not something a stranger gets to
	// learn by asking. The caller answers the same either way, and it can only
	// do that if this does too.
	ctx, repo, _ := accessFixture(t)

	link, send, err := repo.StartSignIn(ctx, "nobody@example.test")
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	if send || link != "" {
		t.Error("a link was minted for an address nobody has")
	}
}

func TestALinkWorksOnceAndOpensASession(t *testing.T) {
	ctx, repo, _ := accessFixture(t)
	person(t, ctx, repo.Queries(), "nina", false)
	email := onlyEmail(t, ctx, repo, "nina")

	link, send, err := repo.StartSignIn(ctx, email)
	if err != nil || !send {
		t.Fatalf("starting: %v (send=%v)", err, send)
	}

	session, ok, err := repo.ClaimSignIn(ctx, link)
	if err != nil || !ok {
		t.Fatalf("claiming: %v (ok=%v)", err, ok)
	}

	// Spent. A second attempt is refused, and it is refused the same way a link
	// that never existed is.
	if _, again, err := repo.ClaimSignIn(ctx, link); err != nil || again {
		t.Errorf("the link worked twice (again=%v, err=%v)", again, err)
	}
	if _, invented, err := repo.ClaimSignIn(ctx, "never-issued"); err != nil || invented {
		t.Errorf("an invented link was honoured (ok=%v, err=%v)", invented, err)
	}

	by, found, err := repo.UserBySession(ctx, session)
	if err != nil || !found {
		t.Fatalf("reading the session: %v (found=%v)", err, found)
	}
	if by.Kind != actor.Human {
		t.Errorf("kind = %q, want human — it is derived from having signed in", by.Kind)
	}
}

func TestDeactivatingAnAccountShutsItsSessionsAtOnce(t *testing.T) {
	// This is why a session is a row and not a signed token: a token the server
	// cannot take back stays valid until it expires, whatever the database
	// says.
	ctx, repo, _ := accessFixture(t)
	gone := person(t, ctx, repo.Queries(), "gone", false)
	email := onlyEmail(t, ctx, repo, "gone")

	link, _, err := repo.StartSignIn(ctx, email)
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	session, ok, err := repo.ClaimSignIn(ctx, link)
	if err != nil || !ok {
		t.Fatalf("claiming: %v", err)
	}
	if _, found, _ := repo.UserBySession(ctx, session); !found {
		t.Fatal("the session did not work before deactivation")
	}

	if err := repo.Queries().DeactivateUser(ctx, gone.ID); err != nil {
		t.Fatalf("deactivating: %v", err)
	}

	if _, found, err := repo.UserBySession(ctx, session); err != nil || found {
		t.Errorf("the session survived deactivation (found=%v, err=%v)", found, err)
	}
}

func TestSigningOutForgetsOneBrowserAndNotTheOthers(t *testing.T) {
	ctx, repo, _ := accessFixture(t)
	person(t, ctx, repo.Queries(), "nina", false)
	email := onlyEmail(t, ctx, repo, "nina")

	open := func() string {
		t.Helper()
		link, _, err := repo.StartSignIn(ctx, email)
		if err != nil {
			t.Fatalf("starting: %v", err)
		}
		session, ok, err := repo.ClaimSignIn(ctx, link)
		if err != nil || !ok {
			t.Fatalf("claiming: %v", err)
		}
		return session
	}
	laptop, phone := open(), open()

	if err := repo.EndSession(ctx, laptop); err != nil {
		t.Fatalf("signing out: %v", err)
	}
	if _, found, _ := repo.UserBySession(ctx, laptop); found {
		t.Error("the session that signed out still works")
	}
	if _, found, _ := repo.UserBySession(ctx, phone); !found {
		t.Error("signing out of one browser signed out of the other")
	}
}

// onlyEmail reads back the address the fixture gave a person, since it carries
// a clock to keep runs from colliding.
func onlyEmail(t *testing.T, ctx context.Context, repo *postgres.Repository, name string) string {
	t.Helper()
	var email string
	if err := repo.Pool().QueryRow(ctx,
		"SELECT email FROM users WHERE name = $1 ORDER BY created_at DESC LIMIT 1", name,
	).Scan(&email); err != nil {
		t.Fatalf("reading the address: %v", err)
	}
	return email
}
