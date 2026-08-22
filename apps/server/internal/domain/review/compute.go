package review

// This file is the one place a case's state is decided.
//
// ADR 0012 says so, and ADR 0002 says why: the predecessor spread one rule
// across five files, and no two of them agreed for long. Everything here is a
// pure function of facts already recorded — no clock, no database, no I/O — so
// a transition can be replayed from the journal and compared against what was
// stored.

// Cell names one square of a case's grid.
type Cell struct {
	StepID    string
	VariantID string
}

// Comment is what the computation needs to know about a reviewer's report. Its
// text, its author and its history are irrelevant here.
type Comment struct {
	State CommentState
	// Cells the comment covers: its step, crossed with the variants it applies
	// to. One defect over four variants is one comment over four cells.
	Cells []Cell
}

// Facts is everything the computation reads. Nothing else may influence the
// result — that is what makes a replay meaningful.
type Facts struct {
	// Captures present at the edition the case points at.
	Captures []Cell
	// Cells the reviewer has explicitly validated.
	Validated []Cell
	// Every comment on the case, settled ones included: a discarded comment
	// stops counting, but it still exists (ADR 0006).
	Comments []Comment
}

// Outcome is what the facts amount to.
type Outcome struct {
	State CaseState
	// Verdicts is the status of every capture the case has.
	Verdicts map[Cell]CaptureStatus
}

// CaptureStatus is what one square of the grid is waiting for.
type CaptureStatus string

const (
	// CaptureToReview has not been judged.
	CaptureToReview CaptureStatus = "to-review"
	// CaptureToFix is covered by an open comment.
	CaptureToFix CaptureStatus = "to-fix"
	// CaptureValidated was looked at, with nothing to say.
	CaptureValidated CaptureStatus = "validated"
)

// Compute decides a case's state and the status of each of its captures.
//
// The order of the checks is the rule, and it reads as the question "who holds
// the ball": the reviewer first, because their verdict can cancel the dev's
// work; the dev next; nobody last.
func Compute(f Facts) Outcome {
	verdicts := verdictsOf(f)

	out := Outcome{Verdicts: verdicts}

	// No evidence at all: outside the funnel. Not a failure — a case may
	// legitimately never be captured (ADR 0012).
	if len(f.Captures) == 0 {
		out.State = CaseNotInstrumented
		return out
	}

	// Something still awaits the reviewer: a square nobody has judged, or a
	// comment whose delivery has arrived and not been judged.
	for _, status := range verdicts {
		if status == CaptureToReview {
			out.State = CaseToReview
			return out
		}
	}
	for _, c := range f.Comments {
		if c.State == CommentToReview {
			out.State = CaseToReview
			return out
		}
	}

	// Nothing awaits the reviewer. Anything still open awaits the dev — and
	// which of the two it is does not belong on the case: the comment carries
	// that (ADR 0012).
	for _, c := range f.Comments {
		if c.State.Open() {
			out.State = CaseToFix
			return out
		}
	}

	out.State = CaseReviewed
	return out
}

// verdictsOf gives every capture its status.
//
// A comment wins over a validation: a square someone reported a problem on is
// not a square that is fine, whatever was ticked before.
func verdictsOf(f Facts) map[Cell]CaptureStatus {
	verdicts := make(map[Cell]CaptureStatus, len(f.Captures))
	for _, cell := range f.Captures {
		verdicts[cell] = CaptureToReview
	}

	for _, cell := range f.Validated {
		if _, exists := verdicts[cell]; exists {
			verdicts[cell] = CaptureValidated
		}
	}

	for _, c := range f.Comments {
		if !c.State.Open() {
			continue
		}
		for _, cell := range c.Cells {
			if _, exists := verdicts[cell]; exists {
				verdicts[cell] = CaptureToFix
			}
		}
	}

	return verdicts
}
