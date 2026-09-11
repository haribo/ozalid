package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

// pushEdition takes in one capture of one step, from one named environment,
// and returns the address of its bytes.
func pushEdition(
	t *testing.T, ctx context.Context, repo *postgres.Repository,
	project sqlcgen.Project, kase sqlcgen.Case, body, environment string,
) string {
	t.Helper()
	hash := storeBlob(t, ctx, repo, body)
	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant:    map[string]string{"theme": "light"},
					Hash:       hash,
					Provenance: contract.Provenance{EnvironmentID: environment},
				}},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}
	return hash
}

func onlyCapture(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string) review.Capture {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if len(grid.Steps) != 1 || len(grid.Steps[0].Captures) != 1 {
		t.Fatalf("want one step with one capture, got %d steps", len(grid.Steps))
	}
	return review.Capture{StepID: grid.Steps[0].ID, VariantID: grid.Steps[0].Captures[0].VariantID}
}

func TestAnIntakeDoesNotMoveTheBytesUnderAReviewer(t *testing.T) {
	// Somebody is looking means somebody holds the lock (ADR 0024): a run
	// landing now must not change what they are judging (product.md §7).
	ctx, repo, project, kase := intakeFixture(t)
	first := pushEdition(t, ctx, repo, project, kase, "the form, first run", "ci")
	nina := reviewer(t, ctx, repo, "nina")
	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, nina, true); err != nil {
		t.Fatalf("claiming: %v", err)
	}

	pushEdition(t, ctx, repo, project, kase, "the form, second run", "ci")

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if grid.Steps[0].Captures[0].Hash != first {
		t.Errorf("grid shows %q, want the bytes the holder opened", grid.Steps[0].Captures[0].Hash)
	}
}

func TestACaseCatchesUpAsSoonAsNobodyHoldsIt(t *testing.T) {
	// The displayed edition is derived (ADR 0024): a held case keeps the
	// bytes its lock stamped, and the moment the hold ends the next read
	// answers the latest — no write in between.
	ctx, repo, project, kase := intakeFixture(t)
	first := pushEdition(t, ctx, repo, project, kase, "the form, first run", "ci")
	nina := reviewer(t, ctx, repo, "nina")
	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, nina, true); err != nil {
		t.Fatalf("claiming: %v", err)
	}
	second := pushEdition(t, ctx, repo, project, kase, "the form, second run", "ci")

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the held grid: %v", err)
	}
	if grid.Steps[0].Captures[0].Hash != first {
		t.Fatalf("the bytes moved under the holder")
	}

	if err := repo.ReleaseCase(ctx, project.Slug, kase.ID, nina); err != nil {
		t.Fatalf("releasing: %v", err)
	}
	grid, err = repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the free grid: %v", err)
	}
	if grid.Steps[0].Captures[0].Hash != second {
		t.Errorf("grid still shows the old edition; the free case never caught up")
	}
}

// The dd9c192d95e0 case (#248): a to-review case nobody holds is not frozen —
// it reads at the latest edition and shows a recording that landed after it
// was born.
func TestAFreeCaseReadsAtTheLatestEdition(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the form, before any video", "ci")

	video := storeBlob(t, ctx, repo, "webm "+t.Name())
	still := storeBlob(t, ctx, repo, "the form, with a video")
	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant: map[string]string{"theme": "light"}, Hash: still,
					Provenance: contract.Provenance{EnvironmentID: "ci"},
				}},
			}},
			Recordings: []contract.ManifestRecording{{
				Variant: map[string]string{"theme": "light"}, Hash: video,
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("the recording edition: %v", err)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if len(grid.Recordings) != 1 {
		t.Errorf("recordings = %d, want the video that landed after the case was born", len(grid.Recordings))
	}
	if grid.Steps[0].Captures[0].Hash != still {
		t.Errorf("grid shows %q, want the latest edition's bytes", grid.Steps[0].Captures[0].Hash)
	}
}

// A held case keeps the bytes stamped at claim while a push lands, and reads
// the latest once the lock dies — no write in between (ADR 0024).
func TestAHeldCaseKeepsItsBytesUntilTheLockDies(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	first := pushEdition(t, ctx, repo, project, kase, "the form, first run", "ci")
	nina := reviewer(t, ctx, repo, "nina")
	repo.SetLockWindow(1 * time.Second)
	if _, err := repo.ClaimCase(ctx, project.Slug, kase.ID, nina, true); err != nil {
		t.Fatalf("claiming: %v", err)
	}
	second := pushEdition(t, ctx, repo, project, kase, "the form, second run", "ci")

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the held grid: %v", err)
	}
	if grid.Steps[0].Captures[0].Hash != first {
		t.Fatalf("the bytes moved under a live lock")
	}

	// The heartbeat goes silent; the next read answers the latest, and no
	// write happened anywhere.
	time.Sleep(1200 * time.Millisecond)
	grid, err = repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading after the silence: %v", err)
	}
	if grid.Steps[0].Captures[0].Hash != second {
		t.Errorf("grid still shows the dead lock's edition")
	}
}
