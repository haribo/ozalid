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

// CaptureFact is one capture as the computation reads it: its address, the
// bytes on display, and what was last approved there (ADR 0021).
type CaptureFact struct {
	Capture
	// Hash is the content address of the bytes on display.
	Hash string
	// Reference is the content address a reviewer last approved for this
	// capture in its own environment — nil when nobody ever has (ADR 0017).
	Reference *string
	// MovedPixels is the measurement intake recorded against the reference:
	// nil when no comparison could run (no reference, or dimensions differ).
	MovedPixels *int
}

// CommentAnchor is one capture a comment covers, with the bytes the remark
// was written about (#132): what decides whether the pixels on display are
// still the ones that were refused.
type CommentAnchor struct {
	Capture
	// AnchorHash is the content address of the capture the remark was written
	// on — nil when the covered variant had no capture to anchor to.
	AnchorHash *string
}

// Comment is what the computation needs to know about a reviewer's report. Its
// text, its author and its history are irrelevant here.
type Comment struct {
	State CommentState
	// Captures the comment covers: its step, crossed with the variants it applies
	// to. One defect over four variants is one comment over four captures.
	Captures []CommentAnchor
}

// Facts is everything the computation reads. Nothing else may influence the
// result — that is what makes a replay meaningful. Statuses never appear
// here: the computation reads facts and only facts, so it can never eat its
// own output (ADR 0021, the root cause behind #154 and #167).
type Facts struct {
	// Captures present at the edition the case points at.
	Captures []CaptureFact
	// Captures the reviewer has explicitly accepted.
	Accepted []Capture
	// Every comment on the case, settled ones included: a discarded comment
	// stops counting, but it still exists (ADR 0006).
	Comments []Comment
	// PixelThreshold is how many differing pixels this project calls noise.
	PixelThreshold int
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
	// CaptureMoved was accepted, and the image has since changed beyond the
	// project's noise threshold: the pixels on display are not the pixels
	// that were approved (ADR 0021).
	CaptureMoved CaptureStatus = "moved"
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

	// Something still awaits the reviewer: a capture nobody has judged, one
	// that moved since it was judged, or a comment whose delivery has arrived
	// and not been judged. Moved never rises to the case as its own word —
	// the case only says the reviewer is needed (ADR 0021).
	for _, status := range verdicts {
		if status == CaptureToReview || status == CaptureMoved {
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
			out.State = CaseRefused
			return out
		}
	}

	out.State = CaseAccepted
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
//     on is not a capture that is fine, whatever was ticked before — unless
//     the pixels changed under the refusal: a dev-side claim about an image
//     nobody sees any more hands the capture back to the reviewer (ADR 0021);
//  5. moved runs last, on what is otherwise accepted: an image that changed
//     beyond the noise threshold is not the image that was approved. An open
//     comment outranks it — the reason that capture waits is already known.
func verdictsOf(f Facts) map[Capture]CaptureStatus {
	verdicts := make(map[Capture]CaptureStatus, len(f.Captures))
	for _, capture := range f.Captures {
		verdicts[capture.Capture] = CaptureToReview
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
			if _, exists := verdicts[capture.Capture]; exists {
				verdicts[capture.Capture] = CaptureAccepted
			}
		}
	}

	// A delivered comment hands its captures back to the reviewer: the ball is
	// theirs, and the grid says so. The dev-side pass runs second, so a capture
	// also claimed by a tracked or refused comment stays refused — the finest
	// open claim wins, exactly as for the comment itself (#150).
	for _, c := range f.Comments {
		if c.State != CommentToReview {
			continue
		}
		for _, capture := range c.Captures {
			if _, exists := verdicts[capture.Capture]; exists {
				verdicts[capture.Capture] = CaptureToReview
			}
		}
	}
	displayed := make(map[Capture]CaptureFact, len(f.Captures))
	for _, capture := range f.Captures {
		displayed[capture.Capture] = capture
	}
	for _, c := range f.Comments {
		if !c.State.Open() || c.State == CommentToReview {
			continue
		}
		for _, capture := range c.Captures {
			if _, exists := verdicts[capture.Capture]; !exists {
				continue
			}
			// The pixels changed under the refusal: what is on display is not
			// what was refused, and nobody has judged it (ADR 0021). The
			// comment keeps its own cycle — this is the capture talking.
			if capture.AnchorHash != nil && displayed[capture.Capture].Hash != *capture.AnchorHash {
				verdicts[capture.Capture] = CaptureToReview
				continue
			}
			verdicts[capture.Capture] = CaptureRefused
		}
	}

	// Moved, last and only on what is otherwise accepted: an image that
	// changed beyond the noise threshold is not the image that was approved.
	// A nil measurement with differing hashes means the dimensions differ —
	// that is a change, not noise (ADR 0021).
	for _, capture := range f.Captures {
		if verdicts[capture.Capture] != CaptureAccepted {
			continue
		}
		if capture.Reference == nil || capture.Hash == *capture.Reference {
			continue
		}
		if capture.MovedPixels != nil && *capture.MovedPixels <= f.PixelThreshold {
			continue
		}
		verdicts[capture.Capture] = CaptureMoved
	}

	return verdicts
}
