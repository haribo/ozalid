package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	appintake "github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

// WriteEdition writes a whole manifest in one transaction.
//
// Everything the manifest asserts is checked first — the project, the cases,
// the content addresses — and only then is anything written. A failure at any
// point rolls the lot back, so a half-written edition never exists.
func (r *Repository) WriteEdition(ctx context.Context, projectSlug string, m contract.Manifest) (appintake.Result, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return appintake.Result{}, fmt.Errorf("beginning the intake: %w", err)
	}
	defer func() {
		// best-effort: rolling back an already-committed transaction is
		// expected to fail and means the write went through.
		_ = tx.Rollback(ctx)
	}()

	q := r.q.WithTx(tx)

	project, err := q.GetProjectBySlug(ctx, projectSlug)
	if err != nil {
		return appintake.Result{}, translate("reading the project", err)
	}

	if err := checkPolicy(ctx, q, project); err != nil {
		return appintake.Result{}, err
	}
	known, err := knownCases(ctx, q, project.ID, m)
	if err != nil {
		return appintake.Result{}, err
	}
	if err := checkContentIsHeld(ctx, q, m); err != nil {
		return appintake.Result{}, err
	}

	// Axes are declared by first use (ADR 0001). A new one lands after those
	// the project already knows, so declaring one never renumbers the rest —
	// which would silently relabel every variant already stored.
	existing, err := q.ListAxes(ctx, project.ID)
	if err != nil {
		return appintake.Result{}, translate("reading the declared axes", err)
	}
	declared := make(map[string]struct{}, len(existing))
	order := make([]string, 0, len(existing))
	for _, axis := range existing {
		declared[axis.Name] = struct{}{}
		order = append(order, axis.Name)
	}
	next := int32(len(existing))
	for _, axis := range appintake.AxisNames(m) {
		if _, seen := declared[axis]; seen {
			continue
		}
		if _, err := q.UpsertAxis(ctx, sqlcgen.UpsertAxisParams{
			ProjectID: project.ID, Name: axis, Position: next,
		}); err != nil {
			return appintake.Result{}, translate("declaring an axis", err)
		}
		order = append(order, axis)
		next++
	}

	edition, err := q.CreateEdition(ctx, sqlcgen.CreateEditionParams{
		ProjectID: project.ID, Revision: optional(m.Revision),
	})
	if err != nil {
		return appintake.Result{}, translate("creating the edition", err)
	}

	variants := newVariantCache(project.ID, order)
	result := appintake.Result{EditionID: edition.ID, Cases: len(m.Cases)}

	for _, mc := range m.Cases {
		for position, ms := range mc.Steps {
			step, err := q.UpsertStep(ctx, sqlcgen.UpsertStepParams{
				CaseID: known[mc.ID].ID, Name: ms.Name, Position: int32(position),
			})
			if err != nil {
				return appintake.Result{}, translate("recording a step", err)
			}

			for _, mcap := range ms.Captures {
				variantID, err := variants.resolve(ctx, q, mcap.Variant)
				if err != nil {
					return appintake.Result{}, err
				}
				provenance, err := json.Marshal(mcap.Provenance)
				if err != nil {
					return appintake.Result{}, fmt.Errorf("encoding the provenance: %w", err)
				}
				if _, err := q.CreateCapture(ctx, sqlcgen.CreateCaptureParams{
					EditionID: edition.ID, StepID: step.ID, VariantID: variantID,
					BlobHash: mcap.Hash, Provenance: provenance,
				}); err != nil {
					return appintake.Result{}, translate("recording a capture", err)
				}
				result.Captures++
			}
		}

		// Steps the manifest no longer carries are gone from the flow. Their
		// captures go with them; the comments anchored to them do too, which is
		// why a step keeps its identity as long as it keeps its position.
		if err := q.DeleteStepsBeyond(ctx, sqlcgen.DeleteStepsBeyondParams{
			CaseID: known[mc.ID].ID, Position: int32(len(mc.Steps)),
		}); err != nil {
			return appintake.Result{}, translate("pruning vanished steps", err)
		}

		for _, mr := range mc.Recordings {
			variantID, err := variants.resolve(ctx, q, mr.Variant)
			if err != nil {
				return appintake.Result{}, err
			}
			if _, err := q.CreateRecording(ctx, sqlcgen.CreateRecordingParams{
				EditionID: edition.ID, CaseID: known[mc.ID].ID,
				VariantID: variantID, BlobHash: mr.Hash,
			}); err != nil {
				return appintake.Result{}, translate("recording a video", err)
			}
			result.Recordings++
		}
	}

	// A case advances onto the edition that just landed -- unless a reviewer is
	// sitting on it. `to-review` means somebody is looking, and moving the
	// bytes under them would have them judge one image and approve another
	// (product.md §7).
	if _, err := q.AdvanceCurrentEdition(ctx, sqlcgen.AdvanceCurrentEditionParams{
		EditionID: &edition.ID, CaseIds: caseIDs(m),
	}); err != nil {
		return appintake.Result{}, translate("pointing the cases at the edition", err)
	}

	// The evidence has arrived, so the cases that had none leave the edge of
	// the funnel. This is the only transition intake drives: every other one
	// comes from a comment (ADR 0012).
	entered, err := q.EnterReviewOnFirstCaptures(ctx, caseIDs(m))
	if err != nil {
		return appintake.Result{}, translate("opening the reviews", err)
	}
	for _, id := range entered {
		inputs, err := json.Marshal(map[string]any{
			"edition":  edition.ID,
			"revision": m.Revision,
		})
		if err != nil {
			return appintake.Result{}, fmt.Errorf("encoding the transition inputs: %w", err)
		}
		if err := q.RecordTransition(ctx, sqlcgen.RecordTransitionParams{
			ProjectID: project.ID,
			CaseID:    &id,
			FromState: ptr(string(review.CaseNotInstrumented)),
			ToState:   ptr(string(review.CaseToReview)),
			Cause:     "edition-accepted",
			ActorID:   "intake",
			ActorKind: "machine",
			Inputs:    inputs,
			// Which version of the computation produced this, so a replay
			// knows what it is comparing against.
			RuleVersion: 1,
		}); err != nil {
			return appintake.Result{}, translate("journalling the transition", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return appintake.Result{}, fmt.Errorf("committing the intake: %w", err)
	}
	return result, nil
}

// checkPolicy enforces the project's intake gate (ADR 0007).
func checkPolicy(ctx context.Context, q *sqlcgen.Queries, project sqlcgen.Project) error {
	if catalogue.IntakePolicy(project.IntakePolicy) != catalogue.PolicyStrict {
		return nil
	}
	open, err := q.CountCasesToReview(ctx, project.ID)
	if err != nil {
		return translate("counting the open reviews", err)
	}
	if open > 0 {
		return fmt.Errorf("%w: %d case(s) still to review", appintake.ErrBlockedByPolicy, open)
	}
	return nil
}

// knownCases resolves every case the manifest names, refusing the whole
// manifest if one is unknown.
func knownCases(ctx context.Context, q *sqlcgen.Queries, projectID string, m contract.Manifest) (map[string]sqlcgen.Case, error) {
	ids := make([]string, 0, len(m.Cases))
	for _, c := range m.Cases {
		ids = append(ids, c.ID)
	}

	rows, err := q.CasesByIDs(ctx, sqlcgen.CasesByIDsParams{ProjectID: projectID, Column2: ids})
	if err != nil {
		return nil, translate("resolving the cases", err)
	}

	known := make(map[string]sqlcgen.Case, len(rows))
	for _, row := range rows {
		known[row.ID] = row
	}
	for _, c := range m.Cases {
		if _, ok := known[c.ID]; !ok {
			return nil, fmt.Errorf("%w: %s", appintake.ErrUnknownCase, c.ID)
		}
	}
	return known, nil
}

// checkContentIsHeld refuses a manifest that references bytes the store does
// not have, naming them so the client uploads exactly those.
func checkContentIsHeld(ctx context.Context, q *sqlcgen.Queries, m contract.Manifest) error {
	var missing []string
	for _, hash := range appintake.Addresses(m) {
		held, err := q.BlobExists(ctx, hash)
		if err != nil {
			return translate("checking for stored content", err)
		}
		if !held {
			missing = append(missing, hash)
		}
	}
	if len(missing) > 0 {
		return &appintake.MissingContent{Hashes: missing}
	}
	return nil
}

// variantCache resolves a combination of axis values to a variant id, creating
// it the first time it is seen and remembering it for the rest of the intake.
type variantCache struct {
	projectID string
	// The project's declared axis order, so a label reads the way the project
	// says rather than alphabetically.
	order []string
	seen  map[string]string
}

func newVariantCache(projectID string, order []string) *variantCache {
	return &variantCache{projectID: projectID, order: order, seen: map[string]string{}}
}

func (c *variantCache) resolve(ctx context.Context, q *sqlcgen.Queries, values map[string]string) (string, error) {
	label := contract.VariantLabel(values, c.order)
	if id, ok := c.seen[label]; ok {
		return id, nil
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encoding the variant: %w", err)
	}
	row, err := q.UpsertVariant(ctx, sqlcgen.UpsertVariantParams{
		ProjectID: c.projectID, Values: encoded, Label: label,
	})
	if err != nil {
		return "", translate("resolving a variant", err)
	}
	c.seen[label] = row.ID
	return row.ID, nil
}

// RecordBlob remembers that the store holds this content.
func (r *Repository) RecordBlob(ctx context.Context, hash string, size int64) error {
	if err := r.q.UpsertBlob(ctx, sqlcgen.UpsertBlobParams{Hash: hash, SizeBytes: size}); err != nil {
		return translate("recording the content", err)
	}
	return nil
}

// caseIDs lists the cases a manifest names.
func caseIDs(m contract.Manifest) []string {
	out := make([]string, 0, len(m.Cases))
	for _, c := range m.Cases {
		out = append(out, c.ID)
	}
	return out
}

func ptr(s string) *string { return &s }

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
