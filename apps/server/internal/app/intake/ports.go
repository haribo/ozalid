package intake

import (
	"context"
	"io"

	"github.com/haribo/ozalid/internal/contract"
)

// Square names one capture cell across an intake, independently of the ids the
// database will assign it.
//
// A step is identified by its position and a variant by its canonical label,
// because those are what the schema keys them on. The environment is part of
// the name: a reference belongs to one, and comparing across two of them says
// nothing (ADR 0017).
type Square struct {
	CaseID        string
	StepPosition  int
	VariantLabel  string
	EnvironmentID string
}

// Repository is the outbound port this package needs.
//
// WriteEdition is deliberately coarse: the whole edition is one atomic write,
// and the transaction that makes it atomic belongs in the adapter — app cannot
// import one (backend ADR 0001).
type Repository interface {
	// AxisOrder returns the order the project declares its axes in, so a
	// variant label computed here reads the same as the one the write will
	// store.
	AxisOrder(ctx context.Context, projectSlug string) ([]string, error)
	// ApprovedBytes returns, per square, the content address a reviewer last
	// approved. A square absent from the map has none, which is not the same as
	// having one that matches.
	ApprovedBytes(ctx context.Context, projectSlug string, m contract.Manifest) (map[Square]string, error)
	// PixelThreshold is how many differing pixels this project calls noise.
	PixelThreshold(ctx context.Context, projectSlug string) (int, error)
	WriteEdition(ctx context.Context, projectSlug string, m contract.Manifest, fresh map[Square]Verdict) (Result, error)
}

// Blobs reads capture bytes back. Intake needs them for two things: proving a
// capture is a PNG, and comparing it against what was approved.
type Blobs interface {
	Get(ctx context.Context, hash string) (io.ReadCloser, error)
}

// Verdict is what the comparison found for one square, ready to be stored.
type Verdict struct {
	State string
	// Pixels is how many differed, or nil when no pixel reading happened:
	// identical addresses need none, and mismatched dimensions admit none.
	Pixels *int
}
