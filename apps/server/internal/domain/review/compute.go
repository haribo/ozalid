package review

// This file is the one place a case's state is decided.
//
// ADR 0012 says so, and ADR 0002 says why: the predecessor spread one rule
// across five files, and no two of them agreed for long. Everything here is a
// pure function of facts already recorded — no clock, no database, no I/O — so
// a transition can be replayed from the journal and compared against what was
// stored.

// Capture names one capture of the grid: a step crossed with a variant.
type Capture struct {
	StepID    string
	VariantID string
}

// Comment is what the computation needs to know about a reviewer's report. Its
// text, its author and its history are irrelevant here.
type Comment struct {
	State CommentState
	// Captures the comment covers: its step, crossed with the variants it applies
	// to. One defect over four variants is one comment over four captures.
	Captures []Capture
}

// Facts is everything the computation reads. Nothing else may influence the
// result — that is what makes a replay meaningful.
type Facts struct {
	// Captures present at the edition the case points at.
	Captures []Capture
	// Captures the reviewer has explicitly accepted.
	Accepted []Capture
	// Every comment on the case, settled ones included: a discarded comment
	// stops counting, but it still exists (ADR 0006).
	Comments []Comment
}

// Outcome is what the facts amount to.
type Outcome struct {
	State CaseState
	// Verdicts is the status of every capture the case has.
	Verdicts map[Capture]CaptureStatus
}

// CaptureStatus is what one capture of the grid is waiting for.
type CaptureStatus string

const (
	// CaptureToReview has not been judged.
	CaptureToReview CaptureStatus = "to-review"
	// CaptureRefused is covered by an open comment.
	CaptureRefused CaptureStatus = "refused"
	// CaptureAccepted was looked at, with nothing to say.
	CaptureAccepted CaptureStatus = "accepted"
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

	// Something still awaits the reviewer: a capture nobody has judged, or a
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
// The order of the passes is the rule:
//
//  1. everything starts unjudged;
//  2. what the reviewer explicitly validated is validated;
//  3. a capture whose comment has been settled counts as judged — settling it
//     *was* the judgment, and asking the reviewer to then validate what they
//     just accepted or set aside would be asking twice;
//  4. an open comment wins over all of it: a capture someone reported a problem
//     on is not a capture that is fine, whatever was ticked before.
func verdictsOf(f Facts) map[Capture]CaptureStatus {
	verdicts := make(map[Capture]CaptureStatus, len(f.Captures))
	for _, capture := range f.Captures {
		verdicts[capture] = CaptureToReview
	}

	for _, capture := range f.Accepted {
		if _, exists := verdicts[capture]; exists {
			verdicts[capture] = CaptureAccepted
		}
	}

	for _, c := range f.Comments {
		if c.State.Open() {
			continue
		}
		for _, capture := range c.Captures {
			if _, exists := verdicts[capture]; exists {
				verdicts[capture] = CaptureAccepted
			}
		}
	}

	// A delivered comment hands its captures back to the reviewer: the ball is
	// theirs, and the grid says so. The dev-side pass runs second, so a capture
	// also claimed by a tracked or refused comment stays to-fix — the finest
	// open claim wins, exactly as for the comment itself (#150).
	for _, c := range f.Comments {
		if c.State != CommentToReview {
			continue
		}
		for _, capture := range c.Captures {
			if _, exists := verdicts[capture]; exists {
				verdicts[capture] = CaptureToReview
			}
		}
	}
	for _, c := range f.Comments {
		if !c.State.Open() || c.State == CommentToReview {
			continue
		}
		for _, capture := range c.Captures {
			if _, exists := verdicts[capture]; exists {
				verdicts[capture] = CaptureRefused
			}
		}
	}

	return verdicts
}
