// Package session saves what a reviewer decided in one sitting.
//
// One save carries everything: the verdicts given and the remarks written.
// Splitting it would let a case sit half-judged between two calls, which is
// the state the whole product exists to avoid.
package session

import (
	"context"
	"errors"
	"strings"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
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
)

// NewComment is a remark the reviewer wrote while refusing. It carries no
// kind: qualifying into fix or feature belongs to whoever writes the issues
// (ADR 0020).
type NewComment struct {
	StepID     string
	Body       string
	VariantIDs []string
}

// Save is what one sitting produced.
type Save struct {
	// Accepted are the captures the reviewer looked at with nothing to say.
	Accepted []review.Capture
	// Unaccepted are acceptances taken back — a misclick, or a second look.
	// The verdict is a toggle until the review ends (#156, ADR 0020).
	Unaccepted []review.Capture
	// Unrefused are draft refusals withdrawn: the reviewer's own remarks with
	// no issue attached go with them (ADR 0020's explicit exception).
	Unrefused []review.Capture
	// Comments are the remarks of this sitting's refusals.
	Comments []NewComment
}

// Result reports what the save amounted to.
type Result struct {
	State    review.CaseState
	Verdicts map[review.Capture]review.CaptureStatus
	Comments int
}

// Repository is the outbound port this package needs.
//
// SaveReview is deliberately coarse: writing the comments, recomputing every
// verdict they cover and moving the case happen together or not at all, and
// the transaction that guarantees it belongs in the adapter (backend ADR 0001).
type Repository interface {
	SaveReview(ctx context.Context, slug, caseID string, by actor.Actor, save Save) (Result, error)
}

// Service saves review sessions.
type Service struct{ repo Repository }

// New returns a Service backed by repo.
func New(repo Repository) *Service { return &Service{repo: repo} }

// Save validates the session and records it.
//
// Validation happens before anything is written: a session carrying one
// unusable comment is refused whole, rather than half-saved.
func (s *Service) Save(ctx context.Context, slug, caseID string, by actor.Actor, save Save) (Result, error) {
	cleaned := make([]NewComment, 0, len(save.Comments))
	for _, c := range save.Comments {
		body := strings.TrimSpace(c.Body)
		if body == "" {
			return Result{}, ErrEmptyBody
		}
		if len(c.VariantIDs) == 0 {
			return Result{}, ErrNoVariant
		}
		cleaned = append(cleaned, NewComment{
			StepID: c.StepID, Body: body, VariantIDs: c.VariantIDs,
		})
	}
	save.Comments = cleaned

	return s.repo.SaveReview(ctx, slug, caseID, by, save)
}
