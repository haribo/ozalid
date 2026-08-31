// Package review holds the review lifecycle: the states a case and a comment
// can be in, and the rules that move between them.
//
// This package is pure. It reads no clock, no environment and no database, so
// a transition can be replayed from recorded facts and compared
// (ADR 0002, backend ADR 0001).
package review

// CaseState answers one question: who holds the ball (ADR 0012).
type CaseState string

const (
	// CaseNotInstrumented has no capture and no verdict. Outside the funnel.
	CaseNotInstrumented CaseState = "not-instrumented"
	// CaseToReview has something waiting for the reviewer's judgment.
	CaseToReview CaseState = "to-review"
	// CaseToFix has nothing awaiting the reviewer and at least one comment
	// awaiting the dev.
	CaseToFix CaseState = "to-fix"
	// CaseReviewed has no open comment. The only clean state.
	CaseReviewed CaseState = "reviewed"
)

// CommentState carries the detail the case state deliberately omits.
type CommentState string

const (
	CommentToTrack   CommentState = "to-track"
	CommentTracked   CommentState = "tracked"
	CommentToReview  CommentState = "to-review"
	CommentRefused   CommentState = "refused"
	CommentValidated CommentState = "validated"
	CommentDiscarded CommentState = "discarded"
)

// Open reports whether the comment still counts against its case. Validated and
// discarded are the only terminal states.
func (s CommentState) Open() bool {
	return s != CommentValidated && s != CommentDiscarded
}
