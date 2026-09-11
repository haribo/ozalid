package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// reviewer registers a human so the lock can name them (ADR 0005).
func reviewer(t *testing.T, ctx context.Context, repo *postgres.Repository, name string) actor.Actor {
	t.Helper()
	u, err := repo.Queries().CreateUser(ctx, sqlcgen.CreateUserParams{
		Name: name, Email: name + "-" + t.Name() + "@ozalid.test",
	})
	if err != nil {
		t.Fatalf("registering %s: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", u.ID)
	})
	return actor.Actor{ID: u.ID, Kind: actor.Human}
}

// A held case takes no verdict but its holder's (ADR 0005, #95): the server
// refuses, whatever the screen offers — and holding never touches the state,
// because occupancy is not a state.
func TestAHeldCaseRefusesAnotherReviewersVerdict(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	captures := seedGrid(t, ctx, repo, project, kase)
	nina := reviewer(t, ctx, repo, "nina")
	marc := reviewer(t, ctx, repo, "marc")
	stateBefore := caseState(t, ctx, repo, project.Slug, kase.ID)

	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, nina, true); err != nil {
		t.Fatalf("nina claiming: %v", err)
	}
	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != stateBefore {
		t.Errorf("state = %q after the claim, want %q — occupancy is not a state", state, stateBefore)
	}

	// Marc's verdicts are refused, and the refusal names the holder.
	var held *review.Held
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, marc, session.Save{
		Accepted: captures[:1],
	}); !errors.As(err, &held) || held.Name != "nina" {
		t.Errorf("save by marc = %v, want Held naming nina", err)
	}
	// The holder keeps working.
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted: captures[:1],
	}); err != nil {
		t.Errorf("save by nina = %v, want it accepted — she holds the case", err)
	}

	// Releasing somebody else's lock changes nothing; releasing one's own
	// frees the case for the next reviewer.
	if err := repo.ReleaseCase(ctx, project.Slug, kase.ID, marc); err != nil {
		t.Fatalf("marc releasing a lock he does not hold: %v", err)
	}
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, marc, session.Save{
		Accepted: captures[1:2],
	}); !errors.As(err, &held) {
		t.Errorf("save by marc = %v, want still refused — his release was a no-op", err)
	}
	if err := repo.ReleaseCase(ctx, project.Slug, kase.ID, nina); err != nil {
		t.Fatalf("nina releasing: %v", err)
	}
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, marc, session.Save{
		Accepted: captures[1:2],
	}); err != nil {
		t.Errorf("save by marc after the release = %v, want it accepted", err)
	}
}

// A lock whose heartbeat goes silent is released on its own: the next
// reviewer takes it without an administrator involved (ADR 0005).
func TestASilentLockExpiresOnItsOwn(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	seedGrid(t, ctx, repo, project, kase)
	nina := reviewer(t, ctx, repo, "nina")
	marc := reviewer(t, ctx, repo, "marc")
	repo.SetLockWindow(1 * time.Second)

	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, nina, true); err != nil {
		t.Fatalf("nina claiming: %v", err)
	}
	var held *review.Held
	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, marc, true); !errors.As(err, &held) {
		t.Fatalf("marc claiming a live lock = %v, want Held", err)
	}

	time.Sleep(1200 * time.Millisecond)
	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, marc, true); err != nil {
		t.Errorf("marc claiming after the silence = %v, want the expired lock taken", err)
	}
}

// Two reviewers reaching for the lock at the same moment: one gets it, the
// other is told who holds it. In parallel — a check that reads before it
// writes is a check both pass (ADR 0005's validation).
func TestTwoReviewersRaceForTheLock(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	seedGrid(t, ctx, repo, project, kase)
	nina := reviewer(t, ctx, repo, "nina")
	marc := reviewer(t, ctx, repo, "marc")

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i, by := range []actor.Actor{nina, marc} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, results[i] = repo.ClaimCase(ctx, project.Slug, kase.ID, by, true)
		}()
	}
	wg.Wait()

	var wins, refusals int
	for _, err := range results {
		var held *review.Held
		switch {
		case err == nil:
			wins++
		case errors.As(err, &held):
			refusals++
		default:
			t.Fatalf("unexpected claim answer: %v", err)
		}
	}
	if wins != 1 || refusals != 1 {
		t.Errorf("wins = %d, refusals = %d — want exactly one of each", wins, refusals)
	}
}
