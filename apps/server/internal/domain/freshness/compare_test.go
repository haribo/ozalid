package freshness_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/domain/freshness"
)

// canvas paints a w×h image in one colour.
func canvas(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestTwoIdenticalImagesHaveNotMoved(t *testing.T) {
	a := canvas(20, 10, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	b := canvas(20, 10, color.RGBA{R: 10, G: 20, B: 30, A: 255})

	got := freshness.Compare(a, b, 0)
	if got.State != freshness.Current {
		t.Errorf("state = %q, want current", got.State)
	}
	if got.Pixels != 0 {
		t.Errorf("pixels = %d, want 0", got.Pixels)
	}
}

func TestAColourRoundedDifferentlyIsTheSameColour(t *testing.T) {
	// The whole point of the tolerance: two renderers agreeing on a colour may
	// still write it one step apart. Calling that a change would raise an alarm
	// on every run.
	a := canvas(20, 10, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	b := canvas(20, 10, color.RGBA{R: 102, G: 98, B: 100, A: 255})

	got := freshness.Compare(a, b, 0)
	if got.State != freshness.Current {
		t.Errorf("state = %q, want current — a two-step difference is rounding", got.State)
	}
}

func TestAColourBeyondTheToleranceIsAChange(t *testing.T) {
	a := canvas(20, 10, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	b := canvas(20, 10, color.RGBA{R: 100, G: 100, B: 140, A: 255})

	got := freshness.Compare(a, b, 0)
	if got.State != freshness.ToReReview {
		t.Errorf("state = %q, want to-re-review", got.State)
	}
}

func TestNoiseBelowTheThresholdDoesNotSummonAnyone(t *testing.T) {
	a := canvas(20, 10, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	b := canvas(20, 10, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	b.Set(3, 3, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	b.Set(4, 3, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	got := freshness.Compare(a, b, 5)
	if got.State != freshness.Current {
		t.Errorf("state = %q, want current — two pixels under a threshold of five", got.State)
	}
	// The count is kept even when nothing is raised: it is what makes the
	// threshold judgeable rather than guessed.
	if got.Pixels != 2 {
		t.Errorf("pixels = %d, want 2", got.Pixels)
	}
}

func TestOnePixelPastTheThresholdIsEnough(t *testing.T) {
	a := canvas(20, 10, color.RGBA{A: 255})
	b := canvas(20, 10, color.RGBA{A: 255})
	for i := 0; i < 6; i++ {
		b.Set(i, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}

	got := freshness.Compare(a, b, 5)
	if got.State != freshness.ToReReview {
		t.Errorf("state = %q, want to-re-review — six pixels over a threshold of five", got.State)
	}
}

func TestEveryPixelIsCountedEvenOnceTheAnswerIsSettled(t *testing.T) {
	// Stopping at the threshold would be cheaper and would make the number
	// useless: it would always read one more than the threshold. The count
	// exists so a project can judge its threshold rather than guess it
	// (product.md §3.3), and a project on the default zero would learn nothing
	// about its own noise.
	a := canvas(400, 400, color.RGBA{A: 255})
	b := canvas(400, 400, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	got := freshness.Compare(a, b, 10)
	if got.State != freshness.ToReReview {
		t.Fatalf("state = %q, want to-re-review", got.State)
	}
	if got.Pixels != 400*400 {
		t.Errorf("pixels = %d, want %d — the whole image differs", got.Pixels, 400*400)
	}
}

// slowOnly wraps an image so it cannot be recognised as an *image.RGBA, which
// forces the general path.
type slowOnly struct{ image.Image }

func TestTheFastPathAndTheSlowPathAgree(t *testing.T) {
	// The byte-slice walk exists only to make counting every pixel affordable.
	// A second implementation that disagrees with the first would be worse than
	// no optimisation at all.
	a := canvas(60, 40, color.RGBA{R: 30, G: 60, B: 90, A: 255})
	b := canvas(60, 40, color.RGBA{R: 30, G: 60, B: 90, A: 255})
	for i := 0; i < 37; i++ {
		b.Set(i%60, i/60, color.RGBA{R: 200, G: 10, B: 10, A: 255})
	}
	// One pixel inside the tolerance, which neither path may count.
	b.Set(59, 39, color.RGBA{R: 32, G: 58, B: 90, A: 255})

	fast := freshness.Compare(a, b, 0)
	slow := freshness.Compare(slowOnly{a}, slowOnly{b}, 0)
	if fast.Pixels != slow.Pixels {
		t.Errorf("fast path counted %d, slow path counted %d", fast.Pixels, slow.Pixels)
	}
	if fast.Pixels != 37 {
		t.Errorf("pixels = %d, want 37 — the pixel within tolerance is not a change", fast.Pixels)
	}
}

func TestImagesOfDifferentShapesAreNotCompared(t *testing.T) {
	a := canvas(20, 10, color.RGBA{A: 255})
	b := canvas(20, 11, color.RGBA{A: 255})

	got := freshness.Compare(a, b, 0)
	if got.State != freshness.ToReReview {
		t.Errorf("state = %q, want to-re-review", got.State)
	}
	if got.Pixels != -1 {
		t.Errorf("pixels = %d, want -1 — no pixel reading is possible", got.Pixels)
	}
}

func TestAnImageIsComparedWhateverItsOrigin(t *testing.T) {
	// A decoded PNG does not always start at (0,0). Comparing by absolute
	// coordinates rather than by offset would read past the edge.
	a := canvas(4, 4, color.RGBA{R: 7, G: 7, B: 7, A: 255})
	shifted := image.NewRGBA(image.Rect(100, 50, 104, 54))
	for y := 50; y < 54; y++ {
		for x := 100; x < 104; x++ {
			shifted.Set(x, y, color.RGBA{R: 7, G: 7, B: 7, A: 255})
		}
	}

	got := freshness.Compare(a, shifted, 0)
	if got.State != freshness.Current {
		t.Errorf("state = %q, want current — the same picture, drawn elsewhere", got.State)
	}
}
