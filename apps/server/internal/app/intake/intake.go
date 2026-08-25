// Package intake takes a run's evidence into the book.
//
// One manifest, one transaction: an edition is created whole or not at all.
// There is no pending edition to define, clean up or explain — the client asks
// what the store already holds, uploads the rest, and only then pushes.
package intake

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/haribo/ozalid/internal/contract"
)

// Errors a caller is expected to act on.
var (
	// ErrUnknownCase means the manifest named a case this project does not
	// have. The client stores the id ozalid generated; inventing one is a bug
	// on its side (ADR 0014).
	ErrUnknownCase = errors.New("intake: unknown case")
	// ErrDuplicateCase means the same case appeared twice. Two tests writing to
	// one case would corrupt it silently, so the whole manifest is refused.
	ErrDuplicateCase = errors.New("intake: the same case appears twice")
	// ErrMissingBlobs means the manifest referenced content the store does not
	// hold. The caller uploads them and pushes again.
	ErrMissingBlobs = errors.New("intake: missing content")
	// ErrEmptyManifest means there was nothing to take in.
	ErrEmptyManifest = errors.New("intake: the manifest is empty")
	// ErrBlockedByPolicy means the project runs strict and a review is open
	// (ADR 0007).
	ErrBlockedByPolicy = errors.New("intake: refused while a review is open")
)

// MissingContent carries the addresses the store does not hold, so the client
// knows exactly what to upload rather than re-sending everything.
type MissingContent struct {
	Hashes []string
}

func (m *MissingContent) Error() string {
	return fmt.Sprintf("%v: %d address(es)", ErrMissingBlobs, len(m.Hashes))
}

func (m *MissingContent) Unwrap() error { return ErrMissingBlobs }

// Result reports what an accepted edition contains.
type Result struct {
	EditionID  string
	Cases      int
	Captures   int
	Recordings int
}

// Service takes editions in.
type Service struct {
	repo  Repository
	blobs Blobs
}

// New returns a Service backed by repo and blobs.
func New(repo Repository, blobs Blobs) *Service {
	return &Service{repo: repo, blobs: blobs}
}

// Take validates a manifest and, if it holds, writes the whole edition.
//
// Validation happens before anything is written: a manifest that names an
// unknown case, names one twice, or references content the store does not hold
// is refused without a trace.
func (s *Service) Take(ctx context.Context, projectSlug string, m contract.Manifest) (Result, error) {
	if len(m.Cases) == 0 {
		return Result{}, ErrEmptyManifest
	}

	seen := make(map[string]struct{}, len(m.Cases))
	for _, c := range m.Cases {
		if _, dup := seen[c.ID]; dup {
			return Result{}, fmt.Errorf("%w: %s", ErrDuplicateCase, c.ID)
		}
		seen[c.ID] = struct{}{}
	}

	if err := validateAddresses(m); err != nil {
		return Result{}, err
	}

	// A capture that cannot be compared is not a capture (product.md §2). This
	// runs before anything is written, like every other refusal.
	if err := s.checkCapturesArePNG(ctx, m); err != nil {
		return Result{}, err
	}

	threshold, err := s.repo.PixelThreshold(ctx, projectSlug)
	if err != nil {
		return Result{}, err
	}
	fresh, err := s.compareAgainstApproved(ctx, projectSlug, m, threshold)
	if err != nil {
		return Result{}, err
	}

	return s.repo.WriteEdition(ctx, projectSlug, m, fresh)
}

// validateAddresses rejects anything that is not a content address before it
// can reach the store or the database.
func validateAddresses(m contract.Manifest) error {
	for _, c := range m.Cases {
		for _, st := range c.Steps {
			for _, cap := range st.Captures {
				if !contract.ValidHash(cap.Hash) {
					return fmt.Errorf("intake: %q is not a content address", cap.Hash)
				}
			}
		}
		for _, r := range c.Recordings {
			if !contract.ValidHash(r.Hash) {
				return fmt.Errorf("intake: %q is not a content address", r.Hash)
			}
		}
	}
	return nil
}

// Addresses returns every distinct content address a manifest references, in a
// stable order.
func Addresses(m contract.Manifest) []string {
	set := make(map[string]struct{})
	for _, c := range m.Cases {
		for _, st := range c.Steps {
			for _, cap := range st.Captures {
				set[cap.Hash] = struct{}{}
			}
		}
		for _, r := range c.Recordings {
			set[r.Hash] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// AxisNames returns every axis a manifest mentions, sorted, so the same run
// always declares them in the same order.
func AxisNames(m contract.Manifest) []string {
	set := make(map[string]struct{})
	for _, c := range m.Cases {
		for _, st := range c.Steps {
			for _, cap := range st.Captures {
				for axis := range cap.Variant {
					set[strings.TrimSpace(axis)] = struct{}{}
				}
			}
		}
		for _, r := range c.Recordings {
			for axis := range r.Variant {
				set[strings.TrimSpace(axis)] = struct{}{}
			}
		}
	}
	delete(set, "")
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
