package intake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/haribo/ozalid/internal/contract"
)

// ErrNotAPNG means a capture is in a format that cannot be compared.
//
// It is refused rather than stored, and that is the whole point. A lossy format
// re-encodes the same screen into different pixels every run, so a reviewer who
// approved one would hold a reference that never matches again — a trap they
// would spring months later. The refusal reaches the run's log, read by the
// person who can fix it.
var ErrNotAPNG = errors.New("intake: a capture is not a PNG")

// pngSignature is the eight bytes every PNG starts with.
var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

// NotPNG carries the addresses that were refused, so the client knows which
// captures to fix rather than which run to re-read.
type NotPNG struct {
	Hashes []string
}

func (n *NotPNG) Error() string {
	return fmt.Sprintf("%v: %d capture(s)", ErrNotAPNG, len(n.Hashes))
}

func (n *NotPNG) Unwrap() error { return ErrNotAPNG }

// checkCaptures reads the first bytes of every capture the manifest names.
//
// One pass, two possible refusals, and a settled order between them. What
// cannot be read is missing content; what reads and is not a PNG is a format
// problem; **missing wins**. Uploading is what a client has to do before a
// format can be judged at all, and pushing new captures is the ordinary case
// while a bad format is a mistake — the ordinary case must not be interrupted
// by the rare one (#64).
//
// The adapter checks for missing content too, inside the transaction where the
// answer cannot change under it. This is the early, useful answer; that one is
// the guard.
//
// Recordings are not checked: they are never compared, so their format is
// nobody's business (ADR 0013).
func (s *Service) checkCaptures(ctx context.Context, m contract.Manifest) error {
	seen := map[string]struct{}{}
	var absent, notPNG []string
	for _, c := range m.Cases {
		for _, st := range c.Steps {
			for _, capture := range st.Captures {
				if _, done := seen[capture.Hash]; done {
					continue
				}
				seen[capture.Hash] = struct{}{}

				held, ok, err := s.readsAsPNG(ctx, capture.Hash)
				if err != nil {
					return err
				}
				switch {
				case !held:
					absent = append(absent, capture.Hash)
				case !ok:
					notPNG = append(notPNG, capture.Hash)
				}
			}
		}
	}
	if len(absent) > 0 {
		return &MissingContent{Hashes: absent}
	}
	if len(notPNG) > 0 {
		return &NotPNG{Hashes: notPNG}
	}
	return nil
}

func (s *Service) readsAsPNG(ctx context.Context, hash string) (held, isPNG bool, err error) {
	body, err := s.blobs.Get(ctx, hash)
	if err != nil {
		// Absent, not malformed. The caller decides what that means; here it is
		// simply not something to judge the format of.
		return false, false, nil
	}
	defer func() { _ = body.Close() }()

	head := make([]byte, len(pngSignature))
	if _, err := io.ReadFull(body, head); err != nil {
		// Too short to be a PNG is simply not a PNG.
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return true, false, nil
		}
		return true, false, fmt.Errorf("reading %s: %w", hash, err)
	}
	return true, bytes.Equal(head, pngSignature), nil
}
