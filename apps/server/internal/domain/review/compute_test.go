package review_test

import (
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

func cells(ids ...string) []review.Cell {
	out := make([]review.Cell, 0, len(ids))
	for _, id := range ids {
		out = append(out, review.Cell{StepID: "s1", VariantID: id})
	}
	return out
}

func TestACaseWithNoCaptureIsOutsideTheFunnel(t *testing.T) {
	// Never being captured is a legitimate state, not a failure (ADR 0012).
	got := review.Compute(review.Facts{})
	if got.State != review.CaseNotInstrumented {
		t.Errorf("state = %q, want not-instrumented", got.State)
	}
}

func TestACaseWithEverythingValidatedAndNothingOpenIsReviewed(t *testing.T) {
	got := review.Compute(review.Facts{
		Captures:  cells("v1", "v2"),
		Validated: cells("v1", "v2"),
	})
	if got.State != review.CaseReviewed {
		t.Errorf("state = %q, want reviewed — the only clean state", got.State)
	}
	for cell, status := range got.Verdicts {
		if status != review.CaptureValidated {
			t.Errorf("cell %v = %q, want validated", cell, status)
		}
	}
}

func TestOneUnjudgedCaptureKeepsTheWholeCaseWaitingOnTheReviewer(t *testing.T) {
	got := review.Compute(review.Facts{
		Captures:  cells("v1", "v2", "v3"),
		Validated: cells("v1", "v2"),
	})
	if got.State != review.CaseToReview {
		t.Errorf("state = %q, want to-review: a square nobody judged is unfinished work", got.State)
	}
}

func TestAnOpenCommentPutsTheBallInTheDevsCourt(t *testing.T) {
	got := review.Compute(review.Facts{
		Captures:  cells("v1", "v2"),
		Validated: cells("v1"),
		Comments:  []review.Comment{{State: review.CommentToTrack, Cells: cells("v2")}},
	})
	if got.State != review.CaseToFix {
		t.Errorf("state = %q, want to-fix", got.State)
	}
	// The comment's own state says whether it needs tracking or fixing; the
	// case only says whose turn it is (ADR 0012).
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v2"}] != review.CaptureToFix {
		t.Error("the covered cell does not read to-fix")
	}
}

func TestACommentWaitingForJudgmentOutranksOneWaitingForTheDev(t *testing.T) {
	// The reviewer comes first: their verdict can cancel work in progress.
	got := review.Compute(review.Facts{
		Captures:  cells("v1", "v2"),
		Validated: cells("v1", "v2"),
		Comments: []review.Comment{
			{State: review.CommentTracked, Cells: cells("v1")},
			{State: review.CommentToReview, Cells: cells("v2")},
		},
	})
	if got.State != review.CaseToReview {
		t.Errorf("state = %q, want to-review", got.State)
	}
}

func TestASettledCommentStopsCountingButTheCellKeepsItsVerdict(t *testing.T) {
	got := review.Compute(review.Facts{
		Captures:  cells("v1"),
		Validated: cells("v1"),
		Comments: []review.Comment{
			{State: review.CommentDiscarded, Cells: cells("v1")},
			{State: review.CommentValidated, Cells: cells("v1")},
		},
	})
	// Nothing is deleted — a discarded comment stays visible on its case
	// (ADR 0006) — but it no longer holds the case open.
	if got.State != review.CaseReviewed {
		t.Errorf("state = %q, want reviewed once every comment is settled", got.State)
	}
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v1"}] != review.CaptureValidated {
		t.Error("a settled comment should not keep marking its cell")
	}
}

func TestACommentBeatsAValidationOnTheSameCell(t *testing.T) {
	// A square someone reported a problem on is not a square that is fine,
	// whatever was ticked before.
	got := review.Compute(review.Facts{
		Captures:  cells("v1"),
		Validated: cells("v1"),
		Comments:  []review.Comment{{State: review.CommentToTrack, Cells: cells("v1")}},
	})
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v1"}] != review.CaptureToFix {
		t.Error("a validation silenced an open comment")
	}
}

func TestACommentOnACellThatNoLongerExistsIsIgnored(t *testing.T) {
	// A step can lose a variant between two editions. The comment survives —
	// nothing is deleted — but it cannot mark a square that is not there.
	got := review.Compute(review.Facts{
		Captures:  cells("v1"),
		Validated: cells("v1"),
		Comments:  []review.Comment{{State: review.CommentToTrack, Cells: cells("v9")}},
	})
	if len(got.Verdicts) != 1 {
		t.Errorf("got %d verdicts, want one per existing capture", len(got.Verdicts))
	}
	// It still holds the case open: the problem was not solved by the capture
	// disappearing.
	if got.State != review.CaseToFix {
		t.Errorf("state = %q, want to-fix", got.State)
	}
}

func TestTheSameFactsAlwaysProduceTheSameOutcome(t *testing.T) {
	// The whole point of a pure function here: a replay from the journal must
	// be comparable to what was stored (ADR 0002).
	facts := review.Facts{
		Captures:  cells("v1", "v2", "v3"),
		Validated: cells("v1"),
		Comments:  []review.Comment{{State: review.CommentTracked, Cells: cells("v2")}},
	}
	first := review.Compute(facts)
	for range 20 {
		again := review.Compute(facts)
		if again.State != first.State || len(again.Verdicts) != len(first.Verdicts) {
			t.Fatal("the computation is not deterministic")
		}
		for cell, status := range first.Verdicts {
			if again.Verdicts[cell] != status {
				t.Fatalf("cell %v drifted between runs", cell)
			}
		}
	}
}

func TestSettlingACommentCountsAsJudgingItsSquares(t *testing.T) {
	// Accepting a fix, or setting a comment aside, *is* the judgment. Asking
	// the reviewer to then validate the square they just ruled on would be
	// asking twice for the same answer.
	for _, settled := range []review.CommentState{review.CommentValidated, review.CommentDiscarded} {
		got := review.Compute(review.Facts{
			Captures: cells("v1", "v2"),
			// v2 was never validated by hand: only its comment was settled.
			Validated: cells("v1"),
			Comments:  []review.Comment{{State: settled, Cells: cells("v2")}},
		})
		if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v2"}] != review.CaptureValidated {
			t.Errorf("%s: the square reads %q, want validated", settled, got.Verdicts[review.Cell{StepID: "s1", VariantID: "v2"}])
		}
		if got.State != review.CaseReviewed {
			t.Errorf("%s: state = %q, want reviewed", settled, got.State)
		}
	}
}

func TestASquareWithOneSettledAndOneOpenCommentStillNeedsFixing(t *testing.T) {
	// Settling one comment does not clear a square another still holds.
	got := review.Compute(review.Facts{
		Captures:  cells("v1"),
		Validated: nil,
		Comments: []review.Comment{
			{State: review.CommentValidated, Cells: cells("v1")},
			{State: review.CommentToTrack, Cells: cells("v1")},
		},
	})
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v1"}] != review.CaptureToFix {
		t.Error("a settled comment silenced an open one on the same square")
	}
	if got.State != review.CaseToFix {
		t.Errorf("state = %q, want to-fix", got.State)
	}
}

// The production scenario of #150: every ref of the covering comment is
// delivered, and the grid still showed the dev's amber bubble. The ball is the
// reviewer's, and the cell must say so.
func TestADeliveredCommentHandsItsCellsBackToTheReviewer(t *testing.T) {
	got := review.Compute(review.Facts{
		Captures: cells("v1", "v2"),
		Comments: []review.Comment{
			// Delivered: the reviewer holds these cells.
			{State: review.CommentToReview, Cells: cells("v1")},
			// Still with the dev: the bubble stays.
			{State: review.CommentTracked, Cells: cells("v2")},
		},
	})
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v1"}] != review.CaptureToReview {
		t.Error("a delivered comment's cell does not read to-review")
	}
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v2"}] != review.CaptureToFix {
		t.Error("a dev-side comment's cell no longer reads to-fix")
	}
}

// A cell both delivered and still reported by a second, dev-side comment stays
// to-fix: the finest open claim wins, exactly as for the comment itself.
func TestADevSideCommentOutweighsADeliveredOneOnTheSameCell(t *testing.T) {
	got := review.Compute(review.Facts{
		Captures: cells("v1"),
		Comments: []review.Comment{
			{State: review.CommentToReview, Cells: cells("v1")},
			{State: review.CommentRefused, Cells: cells("v1")},
		},
	})
	if got.Verdicts[review.Cell{StepID: "s1", VariantID: "v1"}] != review.CaptureToFix {
		t.Error("the refused comment's claim was outranked by the delivered one")
	}
}
