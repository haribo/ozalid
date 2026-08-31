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
// Every pixel is counted, even once the answer is settled. Stopping early would
// be cheaper and would make the count useless: it would always report one more
// than the threshold, and `product.md` §3.3 keeps the number precisely so a
// project can judge its threshold rather than guess it. A project whose
// threshold is still the default zero — every project, on its first run — would
// learn nothing at all about the noise its own suite makes.
func Compare(approved, incoming image.Image, threshold int) Comparison {
	a, b := approved.Bounds(), incoming.Bounds()
	if a.Dx() != b.Dx() || a.Dy() != b.Dy() {
		return Comparison{State: ToReReview, Pixels: -1}
	}

	differing := countRGBA(approved, incoming)
	if differing < 0 {
		differing = countAny(approved, incoming)
	}

	state := Current
	if differing > threshold {
		state = ToReReview
	}
	return Comparison{State: state, Pixels: differing}
}

// countRGBA walks two RGBA images through their byte slices, which is what
// decoded PNGs almost always are. Returns -1 when either image is something
// else, leaving the general path to handle it.
//
// The fast path is not an optimisation for its own sake: it is what makes
// counting every pixel affordable, and counting every pixel is what makes the
// number worth keeping.
func countRGBA(approved, incoming image.Image) int {
	a, ok := approved.(*image.RGBA)
	if !ok {
		return -1
	}
	b, ok := incoming.(*image.RGBA)
	if !ok {
		return -1
	}

	const tolerance = Tolerance
	differing, width := 0, a.Rect.Dx()
	for y := 0; y < a.Rect.Dy(); y++ {
		ai, bi := y*a.Stride, y*b.Stride
		for x := 0; x < width; x++ {
			p, q := ai+x*4, bi+x*4
			if diff8(a.Pix[p], b.Pix[q]) > tolerance ||
				diff8(a.Pix[p+1], b.Pix[q+1]) > tolerance ||
				diff8(a.Pix[p+2], b.Pix[q+2]) > tolerance ||
				diff8(a.Pix[p+3], b.Pix[q+3]) > tolerance {
				differing++
			}
		}
	}
	return differing
}

// countAny handles any other image type, one colour at a time.
func countAny(approved, incoming image.Image) int {
	a, b := approved.Bounds(), incoming.Bounds()
	differing := 0
	for y := 0; y < a.Dy(); y++ {
		for x := 0; x < a.Dx(); x++ {
			if !alike(
				approved.At(a.Min.X+x, a.Min.Y+y),
				incoming.At(b.Min.X+x, b.Min.Y+y),
			) {
				differing++
			}
		}
	}
	return differing
}

func diff8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
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
