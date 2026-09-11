package postgres

import (
	"context"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// lockWindow is how long a silent heartbeat keeps holding (ADR 0005, #95).
// A field rather than a constant: the window is a product parameter
// (product.md §3.2), and the tests shrink it to prove expiry.
const defaultLockWindow = 2 * time.Minute

// SetLockWindow overrides the expiry window (OZALID_LOCK_WINDOW).
func (r *Repository) SetLockWindow(d time.Duration) { r.lockWindow = d }

func (r *Repository) window() int32 {
	w := r.lockWindow
	if w <= 0 {
		w = defaultLockWindow
	}
	return int32(w / time.Second)
}

// ClaimCase takes the case's lock, or renews it: one atomic statement claims
// a free or expired lock and beats the caller's own. Somebody else's live
// lock answers review.ErrHeld, naming the holder (ADR 0005).
func (r *Repository) ClaimCase(ctx context.Context, slug, caseID string, by actor.Actor, fresh bool) (review.Hold, error) {
	kase, err := r.q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: caseID, Slug: slug})
	if err != nil {
		if isNoRows(err) {
			return review.Hold{}, review.ErrMoveNotAllowed
		}
		return review.Hold{}, translate("finding the case", err)
	}
	// The claim stamps what is current now (ADR 0024): these are the bytes
	// the hold will keep under the reviewer.
	latest, err := r.latestEditionID(ctx, r.q, kase.ProjectID)
	if err != nil {
		return review.Hold{}, err
	}
	row, err := r.q.ClaimCaseLock(ctx, sqlcgen.ClaimCaseLockParams{
		CaseID: kase.ID, AccountID: by.ID, WindowSeconds: r.window(),
		EditionID: latest, Fresh: fresh,
	})
	if err == nil {
		hold := review.Hold{By: row.AccountID, Since: row.ClaimedAt.Time}
		if own, err := r.holderOf(ctx, r.q, kase.ID); err == nil && own != nil {
			hold.Name = own.Name
		}
		return hold, nil
	}
	if !isNoRows(err) {
		return review.Hold{}, translate("claiming the case", err)
	}
	held, err := r.holderOf(ctx, r.q, kase.ID)
	if err != nil {
		return review.Hold{}, err
	}
	if held == nil {
		// The lock moved between the two reads; the caller retries.
		return review.Hold{}, review.ErrMoveNotAllowed
	}
	return review.Hold{}, &review.Held{By: held.By, Name: held.Name, Since: held.Since}
}

// ReleaseCase lets go. Releasing a lock nobody holds, or somebody else's,
// changes nothing (ADR 0005's validation).
func (r *Repository) ReleaseCase(ctx context.Context, slug, caseID string, by actor.Actor) error {
	kase, err := r.q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: caseID, Slug: slug})
	if err != nil {
		if isNoRows(err) {
			return review.ErrMoveNotAllowed
		}
		return translate("finding the case", err)
	}
	if err := r.q.ReleaseCaseLock(ctx, sqlcgen.ReleaseCaseLockParams{
		CaseID: kase.ID, AccountID: by.ID,
	}); err != nil {
		return translate("releasing the case", err)
	}
	return nil
}

// holderOf reads who holds the case right now, nil when it is free — an
// expired heartbeat simply stops counting.
func (r *Repository) holderOf(ctx context.Context, q *sqlcgen.Queries, caseID string) (*review.Held, error) {
	row, err := q.ReadCaseLock(ctx, sqlcgen.ReadCaseLockParams{
		CaseID: caseID, WindowSeconds: r.window(),
	})
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, translate("reading the lock", err)
	}
	return &review.Held{By: row.AccountID, Name: row.HolderName, Since: row.ClaimedAt.Time}, nil
}

// refuseHeld answers review.Held when somebody else holds the case: a held
// case takes no verdict but the holder's (ADR 0005). Every reviewer write
// path calls it; the dev's moves never do.
func (r *Repository) refuseHeld(ctx context.Context, q *sqlcgen.Queries, caseID string, by actor.Actor) error {
	held, err := r.holderOf(ctx, q, caseID)
	if err != nil {
		return err
	}
	if held != nil && held.By != by.ID {
		return held
	}
	return nil
}

// displayedEdition resolves which edition the case shows (ADR 0024): the
// live lock's stamped edition, else the project's latest, else nil — a book
// can start empty (ADR 0008). Derived at read, never stored.
func (r *Repository) displayedEdition(ctx context.Context, q *sqlcgen.Queries, kase sqlcgen.Case) (*string, error) {
	row, err := q.ReadCaseLock(ctx, sqlcgen.ReadCaseLockParams{
		CaseID: kase.ID, WindowSeconds: r.window(),
	})
	if err == nil && row.EditionID != nil {
		return row.EditionID, nil
	}
	if err != nil && !isNoRows(err) {
		return nil, translate("reading the lock", err)
	}
	return r.latestEditionID(ctx, q, kase.ProjectID)
}

// latestEditionID is the free case's answer — and the settle-time one, which
// deliberately looks past the saver's own lock (ADR 0024).
func (r *Repository) latestEditionID(ctx context.Context, q *sqlcgen.Queries, projectID string) (*string, error) {
	edition, err := q.LatestEdition(ctx, projectID)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, translate("reading the edition", err)
	}
	return &edition.ID, nil
}
