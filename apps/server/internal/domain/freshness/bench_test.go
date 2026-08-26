package freshness_test

import (
	"image/color"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/domain/freshness"
)

// BenchmarkAFullScreenRedraw is the worst case: a 1280×800 capture where every
// pixel differs. Counting all of them is what keeps the number meaningful, so
// it has to stay affordable.
func BenchmarkAFullScreenRedraw(b *testing.B) {
	before := canvas(1280, 800, color.RGBA{A: 255})
	after := canvas(1280, 800, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := freshness.Compare(before, after, 0); got.Pixels != 1280*800 {
			b.Fatalf("pixels = %d", got.Pixels)
		}
	}
}
