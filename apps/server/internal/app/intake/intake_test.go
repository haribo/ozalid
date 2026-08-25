package intake_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/internal/contract"
)

// refusingRepo fails the test if intake ever reaches it: every case below must
// be refused before anything is written.
type refusingRepo struct{ t *testing.T }

func (r refusingRepo) WriteEdition(
	context.Context, string, contract.Manifest, map[intake.Square]intake.Verdict,
) (intake.Result, error) {
	r.t.Error("the manifest reached the repository, want it refused first")
	return intake.Result{}, nil
}

func (r refusingRepo) AxisOrder(context.Context, string) ([]string, error) {
	r.t.Error("the manifest reached the repository, want it refused first")
	return nil, nil
}

func (r refusingRepo) ApprovedBytes(
	context.Context, string, contract.Manifest,
) (map[intake.Square]string, error) {
	r.t.Error("the manifest reached the repository, want it refused first")
	return nil, nil
}

func (r refusingRepo) PixelThreshold(context.Context, string) (int, error) {
	r.t.Error("the manifest reached the repository, want it refused first")
	return 0, nil
}

// refusingBlobs fails the test if intake ever reads bytes: these manifests are
// refused on their shape, before any content is touched.
type refusingBlobs struct{ t *testing.T }

func (b refusingBlobs) Get(context.Context, string) (io.ReadCloser, error) {
	b.t.Error("intake read a blob, want the manifest refused first")
	return nil, nil
}

func validHash(b byte) string {
	h := "sha256:"
	for i := 0; i < 64; i++ {
		h += string("0123456789abcdef"[int(b)%16])
		b++
	}
	return h
}

func TestAnEmptyManifestIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	svc := intake.New(refusingRepo{t}, refusingBlobs{t})

	_, err := svc.Take(context.Background(), "demo", contract.Manifest{})
	if !errors.Is(err, intake.ErrEmptyManifest) {
		t.Errorf("err = %v, want ErrEmptyManifest", err)
	}
}

func TestTheSameCaseTwiceIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	svc := intake.New(refusingRepo{t}, refusingBlobs{t})

	// Two sources writing to one case would corrupt it with no error to notice
	// (ADR 0014).
	m := contract.Manifest{Cases: []contract.ManifestCase{{ID: "abc"}, {ID: "abc"}}}
	_, err := svc.Take(context.Background(), "demo", m)
	if !errors.Is(err, intake.ErrDuplicateCase) {
		t.Errorf("err = %v, want ErrDuplicateCase", err)
	}
}

func TestAnAddressThatIsNotAHashIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	svc := intake.New(refusingRepo{t}, refusingBlobs{t})

	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: "abc",
		Steps: []contract.ManifestStep{{
			Name:     "opens the form",
			Captures: []contract.ManifestCapture{{Hash: "../../etc/passwd"}},
		}},
	}}}
	if _, err := svc.Take(context.Background(), "demo", m); err == nil {
		t.Error("a manifest carrying a path instead of an address was accepted")
	}
}

func TestAddressesAreDedupedAndOrdered(t *testing.T) {
	a, b := validHash(1), validHash(2)
	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: "one",
		Steps: []contract.ManifestStep{
			{Name: "first", Captures: []contract.ManifestCapture{{Hash: b}, {Hash: a}}},
			{Name: "second", Captures: []contract.ManifestCapture{{Hash: a}}},
		},
		Recordings: []contract.ManifestRecording{{Hash: b}},
	}}}

	got := intake.Addresses(m)
	// Deduped, because the same image appearing four times is one upload.
	if len(got) != 2 {
		t.Fatalf("got %d addresses, want 2 distinct", len(got))
	}
	if got[0] >= got[1] {
		t.Error("addresses are not in a stable order")
	}
}

func TestAxisNamesAreCollectedFromEveryCaptureAndSorted(t *testing.T) {
	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: "one",
		Steps: []contract.ManifestStep{
			{Name: "first", Captures: []contract.ManifestCapture{
				{Variant: map[string]string{"viewport": "desktop", "theme": "light"}},
			}},
			{Name: "second", Captures: []contract.ManifestCapture{
				{Variant: map[string]string{"locale": "fr"}},
			}},
		},
	}}}

	got := intake.AxisNames(m)
	want := []string{"locale", "theme", "viewport"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}
