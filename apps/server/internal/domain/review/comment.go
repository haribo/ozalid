package review

import "errors"

// The comment's own lifecycle. The case says whose turn it is; everything
// about what is actually happening lives here (ADR 0012).
//
// Like the state computation next to it, this is a pure function: the same
// move on the same state always gives the same answer, and nothing about
// clocks, users or storage enters into it.

// Errors a caller acts on.
var (
	// ErrNotOpen means the comment is already settled. Accepted and discarded
	// are the only terminal states, and nothing brings them back.
	ErrNotOpen = errors.New("review: the comment is settled")
	// ErrMoveNotAllowed means the move does not exist from where the comment
	// stands — delivering something nobody tracked, judging something nobody
	// delivered.
	ErrMoveNotAllowed = errors.New("review: that move is not available from this state")
	// ErrReasonRequired means a discard came without its reason. The friction
	// is intended: a defect dismissed without one is exactly what comes back
	// in six months (ADR 0006).
	ErrReasonRequired = errors.New("review: discarding needs a reason")
	// ErrRemarkRequired means a refusal came without its remark. What the dev
	// has to read is the remark, not the state.
	ErrRemarkRequired = errors.New("review: refusing needs a remark")
)

// Move is something someone does to a comment.
type Move string

const (
	// MoveTrack attaches an external issue.
	MoveTrack Move = "track"
	// MoveDiscard sets the comment aside, with a reason.
	MoveDiscard Move = "discard"
	// MoveDeliver is the dev saying "I finished this, look".
	MoveDeliver Move = "deliver"
	// MoveAccept closes the comment.
	MoveAccept Move = "accept"
	// MoveRefuse sends it back, with a remark.
	MoveRefuse Move = "refuse"
)

// allowed is the whole machine, in one table. Reading it is reading the rules.
var allowed = map[Move]map[CommentState]CommentState{
	MoveTrack: {
		CommentToTrack: CommentTracked,
	},
	MoveDiscard: {
		// A comment can be set aside at any point before it is settled: at
		// triage, or once an issue turned out to be the wrong answer.
		CommentToTrack:  CommentDiscarded,
		CommentTracked:  CommentDiscarded,
		CommentToReview: CommentDiscarded,
		CommentRefused:  CommentDiscarded,
	},
	MoveDeliver: {
		CommentTracked: CommentToReview,
		// A refusal is not a way to die: the dev reworks and delivers again,
		// as many rounds as it takes (ADR 0012).
		CommentRefused: CommentToReview,
	},
	MoveAccept: {
		CommentToReview: CommentValidated,
	},
	MoveRefuse: {
		CommentToReview: CommentRefused,
	},
}

// Transition reports what a move does to a comment, or why it cannot.
//
// reason carries the discard's reason or the refusal's remark; both are
// mandatory where they apply, and the check happens here rather than in a
// handler so no caller can skip it.
func Transition(from CommentState, move Move, reason string) (CommentState, error) {
	if !from.Open() {
		return from, ErrNotOpen
	}

	switch move {
	case MoveDiscard:
		if reason == "" {
			return from, ErrReasonRequired
		}
	case MoveRefuse:
		if reason == "" {
			return from, ErrRemarkRequired
		}
	}

	to, ok := allowed[move][from]
	if !ok {
		return from, ErrMoveNotAllowed
	}
	return to, nil
}
