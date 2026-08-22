// Package session saves what a reviewer decided in one sitting.
//
// One save carries everything: the squares validated and the comments written.
// Splitting it would let a case sit half-judged between two calls, which is
// the state the whole product exists to avoid.
package session

import (
	"context"
	"errors"
	"strings"

	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// Errors a caller acts on.
var (
	// ErrEmptyBody means a comment was submitted with nothing written in it.
	ErrEmptyBody = errors.New("session: a comment needs a body")
	// ErrNoVariant means a comment covers nothing. One defect over four
	// variants is one comment with four variants checked; zero variants is a
	// comment about nothing (ADR 0006).
	ErrNoVariant = errors.New("session: a comment covers no variant")
	// ErrUnknownKind means the comment is neither a defect nor an improvement.
	ErrUnknownKind = errors.New("session: unknown comment kind")
)

// NewComment is a report the reviewer wrote during the session.
type NewComment struct {
	StepID     string
	Kind       string
	Body       string
	VariantIDs []string
}

// Save is what one sitting produced.
type Save struct {
	// Validated are the squares the reviewer looked at with nothing to say.
	Validated []review.Cell
	Comments  []NewComment
}

// Result reports what the save amounted to.
type Result struct {
	State    review.CaseState
	Verdicts map[review.Cell]review.CaptureStatus
	Comments int
}

// Repository is the outbound port this package needs.
//
// SaveReview is deliberately coarse: writing the comments, recomputing every
// verdict they cover and moving the case happen together or not at all, and
// the transaction that guarantees it belongs in the adapter (backend ADR 0001).
type Repository interface {
	SaveReview(ctx context.Context, caseID, actorID string, save Save) (Result, error)
}

// Service saves review sessions.
type Service struct{ repo Repository }

// New returns a Service backed by repo.
func New(repo Repository) *Service { return &Service{repo: repo} }

// Save validates the session and records it.
//
// Validation happens before anything is written: a session carrying one
// unusable comment is refused whole, rather than half-saved.
func (s *Service) Save(ctx context.Context, caseID, actorID string, save Save) (Result, error) {
	cleaned := make([]NewComment, 0, len(save.Comments))
	for _, c := range save.Comments {
		body := strings.TrimSpace(c.Body)
		if body == "" {
			return Result{}, ErrEmptyBody
		}
		if c.Kind != "defect" && c.Kind != "improvement" {
			return Result{}, ErrUnknownKind
		}
		if len(c.VariantIDs) == 0 {
			return Result{}, ErrNoVariant
		}
		cleaned = append(cleaned, NewComment{
			StepID: c.StepID, Kind: c.Kind, Body: body, VariantIDs: c.VariantIDs,
		})
	}
	save.Comments = cleaned

	return s.repo.SaveReview(ctx, caseID, actorID, save)
}
