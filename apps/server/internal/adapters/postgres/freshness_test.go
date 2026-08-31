package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/freshness"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

// freshnessFixture adds a blob store to the intake fixture. Intake reads bytes
// back to compare them, so these tests need somewhere those bytes actually are.
func freshnessFixture(t *testing.T) (context.Context, *postgres.Repository, *blobstore.FileStore, sqlcgen.Project, sqlcgen.Case) {
	t.Helper()
	ctx, repo, project, kase := intakeFixture(t)
	blobs, err := blobstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("opening the blob store: %v", err)
	}
	return ctx, repo, blobs, project, kase
}

// screen paints a small PNG and stores it, returning its address. dots are
// painted white, one pixel each, so a test can move an image by an exact
// amount.
func screen(t *testing.T, ctx context.Context, repo *postgres.Repository, blobs *blobstore.FileStore, shade uint8, dots int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}
	for i := 0; i < dots; i++ {
		img.Set(i%40, i/40, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding the png: %v", err)
	}
	return storeBlobBytes(t, ctx, repo, blobs, buf.Bytes())
}

func storeBlobBytes(t *testing.T, ctx context.Context, repo *postgres.Repository, blobs *blobstore.FileStore, body []byte) string {
	t.Helper()
	hash, size, err := contract.HashReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("hashing: %v", err)
	}
	if err := repo.RecordBlob(ctx, hash, size); err != nil {
		t.Fatalf("recording the content: %v", err)
	}
	if err := blobs.Put(ctx, hash, bytes.NewReader(body)); err != nil {
		t.Fatalf("storing the bytes: %v", err)
	}
	return hash
}

// takeIn pushes one capture through the real intake service, so the freshness
// path is exercised end to end rather than simulated.
func takeIn(
	t *testing.T, ctx context.Context, repo *postgres.Repository,
	blobs *blobstore.FileStore, project sqlcgen.Project, kase sqlcgen.Case, hash string,
) error {
	t.Helper()
	svc := intake.New(repo, blobs)
	_, err := svc.Take(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant:    map[string]string{"theme": "light"},
					Hash:       hash,
					Provenance: contract.Provenance{EnvironmentID: "ci"},
				}},
			}},
		}},
	})
	return err
}

func freshnessOf(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string) (string, *int) {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	cell := grid.Steps[0].Cells[0]
	return cell.Freshness, cell.MovedPixels
}

func validateOnly(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string) {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if _, err := repo.SaveReview(ctx, slug, caseID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Validated: []review.Cell{{
			StepID: grid.Steps[0].ID, VariantID: grid.Steps[0].Cells[0].VariantID,
		}},
	}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}
}

func TestACaptureNobodyApprovedSaysNothingAboutItsFreshness(t *testing.T) {
	// Silence is the honest answer: "nothing to compare against" is not
	// "unchanged" (ADR 0017).
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 0)); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	state, moved := freshnessOf(t, ctx, repo, project.Slug, kase.ID)
	if state != "" {
		t.Errorf("freshness = %q, want nothing — no reference exists", state)
	}
	if moved != nil {
		t.Errorf("movedPixels = %v, want nothing", *moved)
	}
}

func TestTheSameBytesComeBackCurrentWithoutBeingCompared(t *testing.T) {
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	same := screen(t, ctx, repo, blobs, 10, 0)
	if err := takeIn(t, ctx, repo, blobs, project, kase, same); err != nil {
		t.Fatalf("taking the first edition in: %v", err)
	}
	validateOnly(t, ctx, repo, project.Slug, kase.ID)

	if err := takeIn(t, ctx, repo, blobs, project, kase, same); err != nil {
		t.Fatalf("taking the second edition in: %v", err)
	}

	state, moved := freshnessOf(t, ctx, repo, project.Slug, kase.ID)
	if state != string(freshness.Current) {
		t.Errorf("freshness = %q, want current", state)
	}
	// Content addressing answers this one for free: same address, same bytes,
	// nothing decoded (ADR 0004).
	if moved != nil {
		t.Errorf("movedPixels = %v, want nothing — no comparison should have run", *moved)
	}
}

func TestAnImageThatMovedIsMarkedAndCounted(t *testing.T) {
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 0)); err != nil {
		t.Fatalf("taking the first edition in: %v", err)
	}
	validateOnly(t, ctx, repo, project.Slug, kase.ID)

	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 4)); err != nil {
		t.Fatalf("taking the second edition in: %v", err)
	}

	state, moved := freshnessOf(t, ctx, repo, project.Slug, kase.ID)
	if state != string(freshness.ToReReview) {
		t.Errorf("freshness = %q, want to-re-review", state)
	}
	if moved == nil {
		t.Fatal("movedPixels is nil, want the count that makes the threshold judgeable")
	}
	// Every differing pixel is counted, threshold or no threshold: the number
	// is what lets a project judge its threshold rather than guess it.
	if *moved != 4 {
		t.Errorf("movedPixels = %d, want 4 — every differing pixel is counted", *moved)
	}
}

func TestNoiseUnderTheProjectsThresholdSummonsNobody(t *testing.T) {
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	if _, err := repo.Pool().Exec(ctx,
		"UPDATE projects SET pixel_threshold = 10 WHERE id = $1", project.ID); err != nil {
		t.Fatalf("setting the threshold: %v", err)
	}

	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 0)); err != nil {
		t.Fatalf("taking the first edition in: %v", err)
	}
	validateOnly(t, ctx, repo, project.Slug, kase.ID)

	// Four differing pixels, on a project that calls ten of them noise.
	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 4)); err != nil {
		t.Fatalf("taking the second edition in: %v", err)
	}

	state, moved := freshnessOf(t, ctx, repo, project.Slug, kase.ID)
	if state != string(freshness.Current) {
		t.Errorf("freshness = %q, want current — four pixels under a threshold of ten", state)
	}
	if moved == nil || *moved != 4 {
		t.Errorf("movedPixels = %v, want 4 kept even though nothing was raised", moved)
	}
}

func TestACaptureThatIsNotAPNGIsRefused(t *testing.T) {
	// A lossy format re-encodes the same screen differently every run, so a
	// reviewer approving one would hold a reference that never matches again
	// (product.md §2).
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	notAnImage := storeBlobBytes(t, ctx, repo, blobs, []byte("\xff\xd8\xff\xe0 this pretends to be a jpeg"))

	err := takeIn(t, ctx, repo, blobs, project, kase, notAnImage)
	var refused *intake.NotPNG
	if !errors.As(err, &refused) {
		t.Fatalf("err = %v, want NotPNG", err)
	}
	if len(refused.Hashes) != 1 || refused.Hashes[0] != notAnImage {
		t.Errorf("hashes = %v, want the offending address named", refused.Hashes)
	}

	// Refused before anything is written, like every other refusal.
	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if len(grid.Steps) != 0 {
		t.Errorf("the case has %d steps, want the manifest refused without a trace", len(grid.Steps))
	}
}

func TestAbsentBytesAreReportedAsMissingRatherThanAsAFailure(t *testing.T) {
	// The designed sequence: push, be told exactly what is missing, upload that,
	// push again. The PNG check used to read every capture before the write
	// could speak, so a capture the store did not hold surfaced as a 500 — and
	// the refusal written for this exact case was unreachable (#64).
	ctx, repo, blobs, project, kase := freshnessFixture(t)

	// An address that is well-formed and that nothing was ever stored under.
	absent := "sha256:" + strings.Repeat("ab", 32)

	svc := intake.New(repo, blobs)
	_, err := svc.Take(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant: map[string]string{"theme": "light"}, Hash: absent,
				}},
			}},
		}},
	})

	var missing *intake.MissingContent
	if !errors.As(err, &missing) {
		t.Fatalf("err = %v, want MissingContent naming the address to upload", err)
	}
	if len(missing.Hashes) != 1 || missing.Hashes[0] != absent {
		t.Errorf("hashes = %v, want exactly the absent address", missing.Hashes)
	}
}

func TestAHeldCaptureThatIsNotAPNGIsStillRefusedOnItsFormat(t *testing.T) {
	// Fixing #64 must not stop the format check from working on bytes that are
	// there.
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	notAnImage := storeBlobBytes(t, ctx, repo, blobs, []byte("\xff\xd8\xff\xe0 pretending to be a jpeg"))

	if err := takeIn(t, ctx, repo, blobs, project, kase, notAnImage); !errors.Is(err, intake.ErrNotAPNG) {
		t.Errorf("err = %v, want ErrNotAPNG", err)
	}
}

func TestMissingContentIsReportedBeforeAFormatProblem(t *testing.T) {
	// Uploading is what the client must do before a format can even be judged,
	// so the missing address is the useful half of the answer.
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	notAnImage := storeBlobBytes(t, ctx, repo, blobs, []byte("\xff\xd8\xff\xe0 pretending to be a jpeg"))
	absent := "sha256:" + strings.Repeat("cd", 32)

	svc := intake.New(repo, blobs)
	_, err := svc.Take(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: absent},
					{Variant: map[string]string{"theme": "dark"}, Hash: notAnImage},
				},
			}},
		}},
	})

	var missing *intake.MissingContent
	if !errors.As(err, &missing) {
		t.Fatalf("err = %v, want the missing content reported first", err)
	}
}
