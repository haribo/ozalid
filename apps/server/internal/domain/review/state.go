// Package review holds the review lifecycle: the states a case and a comment
// can be in, and the rules that move between them.
//
// This package is pure. It reads no clock, no environment and no database, so
// a transition can be replayed from recorded facts and compared
// (ADR 0002, backend ADR 0001).
package review

import "time"

// CaseState answers one question: who holds the ball (ADR 0012).
type CaseState string

const (
	// CaseNotInstrumented has no capture and no verdict. Outside the funnel.
	CaseNotInstrumented CaseState = "not-instrumented"
	// CaseToReview has something waiting for the reviewer's judgment.
	CaseToReview CaseState = "to-review"
	// CaseRefused has nothing awaiting the reviewer and at least one comment
	// awaiting the dev — a refusal is where every open comment comes from
	// (ADR 0021).
	CaseRefused CaseState = "refused"
	// CaseAccepted has no open comment. The only clean state.
	CaseAccepted CaseState = "accepted"
)

// CommentState carries the detail the case state deliberately omits.
type CommentState string

const (
	CommentToTrack   CommentState = "to-track"
	CommentTracked   CommentState = "tracked"
	CommentToReview  CommentState = "to-review"
	CommentRefused   CommentState = "refused"
	CommentAccepted  CommentState = "accepted"
	CommentDiscarded CommentState = "discarded"
)

// Open reports whether the comment still counts against its case. Validated and
// discarded are the only terminal states.
func (s CommentState) Open() bool {
	return s != CommentAccepted && s != CommentDiscarded
}

// Hold says who holds a case and since when (ADR 0005): occupancy, never a
// state.
type Hold struct {
	By    string
	Name  string
	Since time.Time
}

// Held is the refusal a held case gives every reviewer but its holder.
type Held struct {
	By    string
	Name  string
	Since time.Time
}

func (h *Held) Error() string { return "review: held by " + h.By }
