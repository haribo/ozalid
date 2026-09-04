package review_test

import (
	"errors"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

func TestTheHappyPathFromReportToClosure(t *testing.T) {
	steps := []struct {
		from   review.CommentState
		move   review.Move
		reason string
		want   review.CommentState
	}{
		{review.CommentToTrack, review.MoveTrack, "", review.CommentTracked},
		{review.CommentTracked, review.MoveDeliver, "", review.CommentToReview},
		{review.CommentToReview, review.MoveAccept, "", review.CommentValidated},
	}
	for _, s := range steps {
		got, err := review.Transition(s.from, s.move, s.reason)
		if err != nil {
			t.Fatalf("%s from %s: %v", s.move, s.from, err)
		}
		if got != s.want {
			t.Errorf("%s from %s gave %s, want %s", s.move, s.from, got, s.want)
		}
	}
}

func TestARefusalIsNotAWayToDie(t *testing.T) {
	refused, err := review.Transition(review.CommentToReview, review.MoveRefuse, "still cropped on iPhone SE")
	if err != nil {
		t.Fatalf("refusing: %v", err)
	}
	if refused != review.CommentRefused {
		t.Fatalf("refusing gave %s", refused)
	}
	if !refused.Open() {
		t.Error("a refused comment is settled, but the problem is not solved")
	}

	// The dev reworks and delivers again — as many rounds as it takes.
	again, err := review.Transition(refused, review.MoveDeliver, "")
	if err != nil {
		t.Fatalf("delivering again: %v", err)
	}
	if again != review.CommentToReview {
		t.Errorf("a second delivery gave %s, want to-review", again)
	}
}

func TestDiscardingWithoutAReasonIsRefused(t *testing.T) {
	// The friction is intended: a defect dismissed without a reason is exactly
	// what comes back in six months (ADR 0006).
	_, err := review.Transition(review.CommentToTrack, review.MoveDiscard, "")
	if !errors.Is(err, review.ErrReasonRequired) {
		t.Errorf("err = %v, want ErrReasonRequired", err)
	}
}

func TestRefusingWithoutARemarkIsRefused(t *testing.T) {
	// What the dev has to read is the remark, not the state.
	_, err := review.Transition(review.CommentToReview, review.MoveRefuse, "")
	if !errors.Is(err, review.ErrRemarkRequired) {
		t.Errorf("err = %v, want ErrRemarkRequired", err)
	}
}

func TestNothingCanBeDeliveredBeforeItIsTracked(t *testing.T) {
	_, err := review.Transition(review.CommentToTrack, review.MoveDeliver, "")
	if !errors.Is(err, review.ErrMoveNotAllowed) {
		t.Errorf("err = %v, want ErrMoveNotAllowed", err)
	}
}

func TestNothingCanBeJudgedBeforeItIsDelivered(t *testing.T) {
	for _, from := range []review.CommentState{review.CommentToTrack, review.CommentTracked} {
		if _, err := review.Transition(from, review.MoveAccept, ""); !errors.Is(err, review.ErrMoveNotAllowed) {
			t.Errorf("accepting from %s: err = %v, want ErrMoveNotAllowed", from, err)
		}
		if _, err := review.Transition(from, review.MoveRefuse, "no"); !errors.Is(err, review.ErrMoveNotAllowed) {
			t.Errorf("refusing from %s: err = %v, want ErrMoveNotAllowed", from, err)
		}
	}
}

func TestASettledCommentStaysSettled(t *testing.T) {
	// Accepted and discarded are the only terminal states, and nothing brings
	// them back — that is what makes the book's history trustworthy.
	for _, from := range []review.CommentState{review.CommentValidated, review.CommentDiscarded} {
		for _, move := range []review.Move{
			review.MoveTrack, review.MoveDiscard, review.MoveDeliver,
			review.MoveAccept, review.MoveRefuse,
		} {
			if _, err := review.Transition(from, move, "a reason"); !errors.Is(err, review.ErrNotOpen) {
				t.Errorf("%s from %s: err = %v, want ErrNotOpen", move, from, err)
			}
		}
	}
}

func TestSomethingCanBeSetAsideAtAnyPointBeforeItSettles(t *testing.T) {
	// At triage, or once an issue turned out to be the wrong answer.
	for _, from := range []review.CommentState{
		review.CommentToTrack, review.CommentTracked,
		review.CommentToReview, review.CommentRefused,
	} {
		got, err := review.Transition(from, review.MoveDiscard, "agreed it is fine")
		if err != nil {
			t.Errorf("discarding from %s: %v", from, err)
		}
		if got != review.CommentDiscarded {
			t.Errorf("discarding from %s gave %s", from, got)
		}
	}
}

func TestAMoveIsAFunctionOfTheStateAndNothingElse(t *testing.T) {
	// Replaying the journal has to give the same answer as the day it was
	// recorded (ADR 0002).
	for range 20 {
		got, err := review.Transition(review.CommentTracked, review.MoveDeliver, "")
		if err != nil || got != review.CommentToReview {
			t.Fatal("the transition is not deterministic")
		}
	}
}

// One comment, several issues, each on its own round (#138). The derivation is
// the rule that nothing reads settled while anything is undelivered.
func TestACommentReadsAsItsFinestOpenRef(t *testing.T) {
	cases := []struct {
		name string
		refs []review.RefState
		want review.CommentState
	}{
		{"no ref yet", nil, review.CommentToTrack},
		{"one tracked", []review.RefState{review.RefTracked}, review.CommentTracked},
		{"delivered beats tracked", []review.RefState{review.RefTracked, review.RefToReview}, review.CommentToReview},
		{"refused beats tracked", []review.RefState{review.RefTracked, review.RefRefused}, review.CommentRefused},
		{"delivered beats refused", []review.RefState{review.RefRefused, review.RefToReview}, review.CommentToReview},
		{"one validated, one tracked stays open", []review.RefState{review.RefValidated, review.RefTracked}, review.CommentTracked},
		{"all validated closes", []review.RefState{review.RefValidated, review.RefValidated}, review.CommentValidated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := review.DeriveComment(review.CommentTracked, c.refs); got != c.want {
				t.Errorf("DeriveComment(%v) = %v, want %v", c.refs, got, c.want)
			}
		})
	}
}

func TestADiscardIsSaidNotDerived(t *testing.T) {
	// Whatever the refs say, a discard with its reason stands.
	got := review.DeriveComment(review.CommentDiscarded, []review.RefState{review.RefToReview})
	if got != review.CommentDiscarded {
		t.Errorf("a discarded comment re-opened to %v", got)
	}
}

func TestARefMovesLikeACommentUsedTo(t *testing.T) {
	if _, err := review.TransitionRef(review.RefTracked, review.MoveAccept, ""); err == nil {
		t.Error("accepting an undelivered ref was allowed")
	}
	if _, err := review.TransitionRef(review.RefToReview, review.MoveRefuse, ""); err == nil {
		t.Error("refusing without a remark was allowed")
	}
	to, err := review.TransitionRef(review.RefRefused, review.MoveDeliver, "")
	if err != nil || to != review.RefToReview {
		t.Errorf("redelivering a refused ref: %v -> %v", err, to)
	}
	if _, err := review.TransitionRef(review.RefValidated, review.MoveDeliver, ""); err == nil {
		t.Error("a validated ref moved again")
	}
}
