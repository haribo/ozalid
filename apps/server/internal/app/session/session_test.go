package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
)

// refusingRepo fails the test if a session ever reaches it: every case below
// must be refused before anything is written.
type refusingRepo struct{ t *testing.T }

func (r refusingRepo) SaveReview(
	context.Context, string, string, actor.Actor, session.Save,
) (session.Result, error) {
	r.t.Error("the session reached the repository, want it refused first")
	return session.Result{}, nil
}

// recordingRepo remembers what the service handed it.
type recordingRepo struct{ got session.Save }

func (r *recordingRepo) SaveReview(
	_ context.Context, _, _ string, _ actor.Actor, save session.Save,
) (session.Result, error) {
	r.got = save
	return session.Result{}, nil
}

func TestACommentWithNothingWrittenInItIsRefused(t *testing.T) {
	svc := session.New(refusingRepo{t})

	_, err := svc.Save(context.Background(), "atlas", "case", actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{{StepID: "s1", Kind: "defect", Body: "  \n ", VariantIDs: []string{"v1"}}},
	})
	if !errors.Is(err, session.ErrEmptyBody) {
		t.Errorf("err = %v, want ErrEmptyBody", err)
	}
}

func TestACommentCoveringNoVariantIsRefused(t *testing.T) {
	// One defect over four variants is one comment with four checked; zero
	// variants is a comment about nothing (ADR 0006).
	svc := session.New(refusingRepo{t})

	_, err := svc.Save(context.Background(), "atlas", "case", actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{{StepID: "s1", Kind: "defect", Body: "misaligned", VariantIDs: nil}},
	})
	if !errors.Is(err, session.ErrNoVariant) {
		t.Errorf("err = %v, want ErrNoVariant", err)
	}
}

func TestAKindTheProductDoesNotKnowIsRefused(t *testing.T) {
	svc := session.New(refusingRepo{t})

	_, err := svc.Save(context.Background(), "atlas", "case", actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{{StepID: "s1", Kind: "wish", Body: "…", VariantIDs: []string{"v1"}}},
	})
	if !errors.Is(err, session.ErrUnknownKind) {
		t.Errorf("err = %v, want ErrUnknownKind", err)
	}
}

func TestOneUnusableCommentRefusesTheWholeSession(t *testing.T) {
	// Half-saving a sitting would leave the case in a state nobody chose.
	svc := session.New(refusingRepo{t})

	_, err := svc.Save(context.Background(), "atlas", "case", actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{
			{StepID: "s1", Kind: "defect", Body: "fine", VariantIDs: []string{"v1"}},
			{StepID: "s2", Kind: "defect", Body: "", VariantIDs: []string{"v1"}},
		},
	})
	if err == nil {
		t.Error("a session carrying one unusable comment was accepted")
	}
}

func TestSurroundingSpaceIsTrimmedBeforeStoring(t *testing.T) {
	repo := &recordingRepo{}
	svc := session.New(repo)

	if _, err := svc.Save(context.Background(), "atlas", "case", actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{
			{StepID: "s1", Kind: "improvement", Body: "  the label is cramped  ", VariantIDs: []string{"v1"}},
		},
	}); err != nil {
		t.Fatalf("saving: %v", err)
	}
	if repo.got.Comments[0].Body != "the label is cramped" {
		t.Errorf("stored body = %q, want it trimmed", repo.got.Comments[0].Body)
	}
}
