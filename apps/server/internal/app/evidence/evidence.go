// Package evidence reads back what a run put in the book.
//
// A reviewer judges a case from a grid: its steps in order, the variants that
// exist, and the capture sitting at each cell.
package evidence

import (
	"context"
	"errors"
	"time"

	"github.com/haribo/ozalid/internal/contract"
)

// ErrNoEdition means the project has taken nothing in yet, so there is nothing
// to read. It is not a failure — a book starts empty (ADR 0008).
var ErrNoEdition = errors.New("evidence: the project has no edition yet")

// Variant is a combination of axis values, named.
type Variant struct {
	ID     string
	Label  string
	Values map[string]string
}

// Cell is one capture: a variant, the address of its bytes, and where the
// review stands on it.
type Cell struct {
	// ID is the capture. It is what the bytes are fetched through, since a
	// content address names no project and cannot be authorised
	// (product.md §8.1).
	ID        string
	VariantID string
	Hash      string
	Status    string
	// Freshness is empty when there is nothing to compare against: nobody has
	// approved this square in this capture's environment. That is not the same
	// as unchanged (ADR 0017).
	Freshness string
	// MovedPixels is nil when no pixel reading happened — identical addresses
	// need none, mismatched dimensions admit none.
	MovedPixels *int
	Provenance  contract.Provenance
}

// Step is a named business moment and the captures taken at it.
type Step struct {
	ID       string
	Name     string
	Position int
	Cells    []Cell
}

// Recording is the flow video for one variant. Optional, never compared
// (ADR 0013).
type Recording struct {
	ID        string
	VariantID string
	Hash      string
}

// Grid is what a case is judged from.
type Grid struct {
	CaseID     string
	EditionID  string
	Revision   string
	TakenAt    time.Time
	Variants   []Variant
	Steps      []Step
	Recordings []Recording
}

// Repository is the outbound port this package needs.
type Repository interface {
	CaseGrid(ctx context.Context, slug, caseID string, editionID *string) (Grid, error)
	CaptureBlob(ctx context.Context, slug, captureID string) (string, error)
	RecordingBlob(ctx context.Context, slug, recordingID string) (string, error)
}

// Service reads evidence.
type Service struct{ repo Repository }

// New returns a Service backed by repo.
func New(repo Repository) *Service { return &Service{repo: repo} }

// Grid returns the evidence for a case at one edition, defaulting to the
// project's most recent.
//
// A case with no capture answers with an empty grid rather than an error: not
// being instrumented is a legitimate state, not a failure (ADR 0012).
func (s *Service) Grid(ctx context.Context, slug, caseID string, editionID *string) (Grid, error) {
	return s.repo.CaseGrid(ctx, slug, caseID, editionID)
}

// CaptureBlob answers where one capture's bytes are stored, or ErrNotFound
// when that capture is not in that project.
//
// The address it returns is not something the caller may ask for directly:
// resolving it here is what ties bytes shared across projects (ADR 0004) to a
// membership somebody actually holds.
func (s *Service) CaptureBlob(ctx context.Context, slug, captureID string) (string, error) {
	return s.repo.CaptureBlob(ctx, slug, captureID)
}

// RecordingBlob does the same for a recording's video.
func (s *Service) RecordingBlob(ctx context.Context, slug, recordingID string) (string, error) {
	return s.repo.RecordingBlob(ctx, slug, recordingID)
}
