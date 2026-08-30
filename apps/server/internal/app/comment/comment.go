// Package comment carries a report through its life: tracked, delivered,
// judged, or set aside.
//
// The case only says whose turn it is; everything about what is actually
// happening lives here (ADR 0012).
package comment

import (
	"context"
	"strings"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// IssueRef is what a client hands over about an external issue.
//
// All three are **supplied**, never fetched: ozalid holds no tracker
// credential and will never know the issue was closed (ADR 0003).
type IssueRef struct {
	ID    string
	URL   string
	Title string
}

// Outcome reports where the comment and its case ended up.
type Outcome struct {
	CommentState review.CommentState
	CaseState    review.CaseState
}

// Repository is the outbound port this package needs.
//
// Each method is one move: applying it, recomputing the case and journalling
// the result happen together, and the transaction that guarantees it belongs
// in the adapter (backend ADR 0001).
type Repository interface {
	Track(ctx context.Context, commentID string, by actor.Actor, issue IssueRef) (Outcome, error)
	Discard(ctx context.Context, commentID string, by actor.Actor, reason string) (Outcome, error)
	Deliver(ctx context.Context, commentID string, by actor.Actor) (Outcome, error)
	Judge(ctx context.Context, commentID string, by actor.Actor, accept bool, remark string) (Outcome, error)
	OfCase(ctx context.Context, slug, caseID string) ([]Record, error)
}

// Service moves comments along.
type Service struct{ repo Repository }

// New returns a Service backed by repo.
func New(repo Repository) *Service { return &Service{repo: repo} }

// Track attaches an external issue.
func (s *Service) Track(ctx context.Context, commentID string, by actor.Actor, issue IssueRef) (Outcome, error) {
	issue.ID = strings.TrimSpace(issue.ID)
	issue.URL = strings.TrimSpace(issue.URL)
	issue.Title = strings.TrimSpace(issue.Title)
	if issue.ID == "" {
		return Outcome{}, ErrIssueRequired
	}
	return s.repo.Track(ctx, commentID, by, issue)
}

// Discard sets a comment aside. The reason is mandatory and kept forever with
// its author: "I reported this three months ago, who removed it?" must always
// have an answer (ADR 0006).
func (s *Service) Discard(ctx context.Context, commentID string, by actor.Actor, reason string) (Outcome, error) {
	return s.repo.Discard(ctx, commentID, by, strings.TrimSpace(reason))
}

// Deliver is the dev saying the work is done and asking for a judgment.
//
// They may do so without having implemented everything else: one issue can
// depend on the verdict given on another (ADR 0012).
func (s *Service) Deliver(ctx context.Context, commentID string, by actor.Actor) (Outcome, error) {
	return s.repo.Deliver(ctx, commentID, by)
}

// Judge accepts a delivery, or refuses it with a remark.
func (s *Service) Judge(ctx context.Context, commentID string, by actor.Actor, accept bool, remark string) (Outcome, error) {
	return s.repo.Judge(ctx, commentID, by, accept, strings.TrimSpace(remark))
}

// Record is a comment as the layers above read it: what was said, where it
// stands, and everything that happened to it.
type Record struct {
	ID            string
	StepID        string
	Kind          string
	Body          string
	State         review.CommentState
	VariantIDs    []string
	Issue         *IssueRef
	DiscardReason string
	AuthorID      string
	CreatedAt     time.Time
	Judgments     []Judgment
}

// Judgment is one verdict rendered on a delivery. They are all kept.
type Judgment struct {
	Verdict string
	Remark  string
	ActorID string
	At      time.Time
}

// OfCase returns what has been said about a case, settled comments included:
// nothing is deleted, and a discarded one stays visible with its reason
// (ADR 0006).
func (s *Service) OfCase(ctx context.Context, slug, caseID string) ([]Record, error) {
	return s.repo.OfCase(ctx, slug, caseID)
}
