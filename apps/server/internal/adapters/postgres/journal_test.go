package postgres_test

import (
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
)

// The promise of product.md §6 — "I reported this three months ago, who removed
// it?" — needs somebody able to read what the server has always written (#94).
func TestACaseTellsHowItReachedItsState(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	nina := reviewer(t, ctx, repo, "nina")
	seedEdition(t, ctx, repo, project, flow{caseID: kase.ID, steps: []string{"opens"}})

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted: capturesOf(t, ctx, repo, project, kase.ID),
	}); err != nil {
		t.Fatalf("accepting everything: %v", err)
	}

	history, err := repo.CaseHistory(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading the history: %v", err)
	}
	if len(history) < 2 {
		t.Fatalf("history = %d transitions, want the intake and the review", len(history))
	}

	// Oldest first: the run arrived, then somebody judged it.
	first, last := history[0], history[len(history)-1]
	if first.Cause != "edition-accepted" {
		t.Errorf("first cause = %q, want edition-accepted", first.Cause)
	}
	if first.Actor.Kind != string(actor.Machine) {
		t.Errorf("intake actor kind = %q, want machine — the distinction is why the journal is kept", first.Actor.Kind)
	}
	if last.Cause != "review-saved" {
		t.Errorf("last cause = %q, want review-saved", last.Cause)
	}
	// A person, named: an opaque id answers nobody's question.
	if last.Actor.ID != nina.ID || last.Actor.Name != "nina" {
		t.Errorf("review actor = %+v, want nina named", last.Actor)
	}
	if last.ToState == "" {
		t.Error("the last transition names no state it moved to")
	}
	if !last.At.After(first.At) && !last.At.Equal(first.At) {
		t.Errorf("history is not in order: %v then %v", first.At, last.At)
	}
}

// A case nobody has moved answers an empty history, not an error: a case that
// was never captured is a legitimate state (ADR 0012).
func TestACaseNobodyMovedHasAnEmptyHistory(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	q := repo.Queries()
	untouched := openCase(t, ctx, q, project, kase.CategoryID, "never moved")

	history, err := repo.CaseHistory(ctx, project.Slug, untouched.ID)
	if err != nil {
		t.Fatalf("reading the history: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("history = %+v, want empty", history)
	}
}

// Another project's case has no history here rather than one somebody may not
// see (#71).
func TestACaseFromAnotherProjectHasNoHistoryHere(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	if _, err := repo.CaseHistory(ctx, project.Slug+"-elsewhere", kase.ID); err == nil {
		t.Error("a case was given a history under a project it does not belong to")
	}
}
