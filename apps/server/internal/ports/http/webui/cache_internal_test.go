package webui

import "testing"

// The rule the handler applies, tested where the real client's asset names are
// not needed: a clean checkout ships only the placeholder, so nothing under
// `assets/` exists to ask for through the handler.
func TestWhatMayBeKeptAndForHowLong(t *testing.T) {
	for asked, want := range map[string]string{
		"assets/index-BADa5hkn.js":  immutable,
		"assets/index-C0ffee42.css": immutable,
		"index.html":                revalidate,
		"favicon.ico":               revalidate,
		"robots.txt":                revalidate,
	} {
		if got := cacheFor(asked); got != want {
			t.Errorf("cacheFor(%q) = %q, want %q", asked, got, want)
		}
	}
}
