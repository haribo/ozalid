// Package freshness answers one question about one capture: are the bytes on
// display still the bytes the reviewer approved?
//
// Everything here is a pure function over two decoded images. No clock, no
// database, no file — the same two images always give the same answer, which is
// what lets a stored verdict be replayed later and checked (ADR 0002).
package freshness

import "image"

// State is what a capture is worth against what was approved (product.md §3.3).
type State string

const (
	// Current — the bytes the reviewer approved are still the bytes on display.
	Current State = "current"
	// ToReReview — the capture moved, and by more than noise.
	ToReReview State = "to-re-review"
)

// Tolerance is how far two pixels may differ, per channel, before they count as
// different at all.
//
// It is fixed rather than configurable, and deliberately: it describes the same
// colour rounded differently — a property of how screens are rasterised, not of
// any one project. What a project does get to choose is the threshold: how many
// genuinely different pixels are worth summoning a reviewer for. Two dials for
// one symptom would guarantee somebody turns the wrong one.
//
// The value is in 8-bit channel units. Two is the largest step observed between
// renderers agreeing on a colour.
const Tolerance = 2

// Comparison is what looking at two images found.
type Comparison struct {
	State State
	// Pixels is how many differed by more than Tolerance, or -1 when no
	// pixel-by-pixel reading was possible.
	Pixels int
}

// Compare reads two images and says whether the second moved away from the
// first by more than threshold pixels.
//
// Images of different dimensions are moved without being compared: there is no
// pixel-to-pixel reading of two pictures that are not the same shape, and
// pretending otherwise would produce a number nobody could act on.
//
// Counting stops as soon as the threshold is passed. The exact count above it
// changes nothing — the reviewer is summoned either way — and a full-screen
// redraw would otherwise walk millions of pixels to reach a conclusion the
// first thousand already gave.
func Compare(approved, incoming image.Image, threshold int) Comparison {
	a, b := approved.Bounds(), incoming.Bounds()
	if a.Dx() != b.Dx() || a.Dy() != b.Dy() {
		return Comparison{State: ToReReview, Pixels: -1}
	}

	differing := 0
	for y := 0; y < a.Dy(); y++ {
		for x := 0; x < a.Dx(); x++ {
			if !alike(
				approved.At(a.Min.X+x, a.Min.Y+y),
				incoming.At(b.Min.X+x, b.Min.Y+y),
			) {
				differing++
				if differing > threshold {
					return Comparison{State: ToReReview, Pixels: differing}
				}
			}
		}
	}
	return Comparison{State: Current, Pixels: differing}
}

// alike reports whether two colours are the same colour, allowing for the step
// two renderers may disagree by.
func alike(x, y interface{ RGBA() (r, g, b, a uint32) }) bool {
	xr, xg, xb, xa := x.RGBA()
	yr, yg, yb, ya := y.RGBA()
	return within(xr, yr) && within(xg, yg) && within(xb, yb) && within(xa, ya)
}

// within compares two channels. RGBA() returns them pre-multiplied and scaled to
// 16 bits, so the 8-bit tolerance is scaled the same way rather than compared
// against a different unit.
func within(a, b uint32) bool {
	const scaled = Tolerance * 0x101
	if a > b {
		return a-b <= scaled
	}
	return b-a <= scaled
}
