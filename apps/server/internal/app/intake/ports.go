package intake

import (
	"context"

	"github.com/haribo/ozalid/internal/contract"
)

// Repository is the outbound port this package needs.
//
// WriteEdition is deliberately coarse: the whole edition is one atomic write,
// and the transaction that makes it atomic belongs in the adapter — app cannot
// import one (backend ADR 0001).
type Repository interface {
	WriteEdition(ctx context.Context, projectSlug string, m contract.Manifest) (Result, error)
}
