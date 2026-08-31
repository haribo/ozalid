package intake

import (
	"context"
	"fmt"
	"image"
	"image/png"

	"github.com/haribo/ozalid/apps/server/internal/domain/freshness"
	"github.com/haribo/ozalid/internal/contract"
)

// compareAgainstApproved works out, for every capture in the manifest, whether
// it still shows what a reviewer approved.
//
// The lookup and the write are two transactions: a reference could change in
// between. That is accepted — freshness is an overlay, never a state
// (product.md §3.3), so the worst case is a mark that the next intake corrects,
// rather than a case sitting in a wrong place.
func (s *Service) compareAgainstApproved(
	ctx context.Context, projectSlug string, m contract.Manifest, threshold int,
) (map[Square]Verdict, error) {
	approved, err := s.repo.ApprovedBytes(ctx, projectSlug, m)
	if err != nil {
		return nil, err
	}
	if len(approved) == 0 {
		// Nothing has ever been approved on this project. Comparing would read
		// every blob to learn what the empty map already said.
		return nil, nil
	}

	order, err := s.repo.AxisOrder(ctx, projectSlug)
	if err != nil {
		return nil, err
	}

	out := map[Square]Verdict{}
	for _, c := range m.Cases {
		for position, st := range c.Steps {
			for _, capture := range st.Captures {
				square := Square{
					CaseID:        c.ID,
					StepPosition:  position,
					VariantLabel:  contract.VariantLabel(capture.Variant, order),
					EnvironmentID: capture.Provenance.EnvironmentID,
				}
				reference, ok := approved[square]
				if !ok {
					// Nobody has approved this square in this environment.
					// Silence is the honest answer (ADR 0017).
					continue
				}
				verdict, err := s.judge(ctx, reference, capture.Hash, threshold)
				if err != nil {
					return nil, err
				}
				out[square] = verdict
			}
		}
	}
	return out, nil
}

// judge compares one incoming capture against the address that was approved.
func (s *Service) judge(ctx context.Context, reference, incoming string, threshold int) (Verdict, error) {
	// Same address, same bytes, same image. Content addressing makes the common
	// case free: no read, no decode, no comparison (ADR 0004).
	if reference == incoming {
		return Verdict{State: string(freshness.Current)}, nil
	}

	before, err := s.decode(ctx, reference)
	if err != nil {
		return Verdict{}, err
	}
	after, err := s.decode(ctx, incoming)
	if err != nil {
		return Verdict{}, err
	}

	found := freshness.Compare(before, after, threshold)
	verdict := Verdict{State: string(found.State)}
	if found.Pixels >= 0 {
		pixels := found.Pixels
		verdict.Pixels = &pixels
	}
	return verdict, nil
}

func (s *Service) decode(ctx context.Context, hash string) (image.Image, error) {
	body, err := s.blobs.Get(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", hash, err)
	}
	defer func() { _ = body.Close() }()

	img, err := png.Decode(body)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", hash, err)
	}
	return img, nil
}
