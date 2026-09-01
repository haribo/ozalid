// Package webui serves the built web client from inside the binary.
//
// One binary, one container: no second process to run, no proxy to configure,
// and no origin for the client to know — it calls `/api` on whatever host it
// was served from, which is what makes CORS a question nobody has to ask.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// dist holds the client. A placeholder is committed so that `go build` works in
// a checkout where the client was never built — the real files are copied over
// it when the image is made. Without it, every Go build and every test would
// depend on npm having run first.
//
//go:embed all:dist
var dist embed.FS

// Handler serves the client, falling back to its entry point.
//
// The client owns its own routes: a browser loading /cases/abc123 directly must
// be given the application, not a 404. So a path the client did not ship is
// answered with the entry point, and the client reads the path itself.
//
// With one exception: a path carrying a file extension is not a route, it is a
// file that is missing. Handing back HTML there would answer a request for
// JavaScript with a document, and the browser fails on the first `<` with a
// message that says nothing about the real cause.
//
// Requests under /api never reach here — they are routed first.
func Handler() http.Handler {
	files, err := fs.Sub(dist, "dist")
	if err != nil {
		// Only reachable if the embed directive and this path disagree, which
		// is a build-time mistake rather than a runtime condition.
		panic(err)
	}
	serve := http.FileServer(http.FS(files))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked := strings.TrimPrefix(r.URL.Path, "/")
		if asked == "" {
			asked = "index.html"
		}
		if _, err := fs.Stat(files, asked); err != nil {
			if path.Ext(asked) != "" {
				// A missing file, not a route. Say so.
				http.NotFound(w, r)
				return
			}
			// Not a file the client shipped, and shaped like a route: hand over
			// the entry point and let the client decide what it means.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			asked = "index.html"
		}
		w.Header().Set("Cache-Control", cacheFor(asked))
		serve.ServeHTTP(w, r)
	})
}

// cacheFor says how long an answer may be kept.
//
// Two tiers, and the client's own build decides which: Vite writes a content
// hash into every name under `assets/`, so the bytes behind one of those URLs
// never change and revalidating is wasted. The entry point is the opposite — it
// is what names the hashed files, so a stale copy points at a bundle that is
// gone.
//
// A header rather than a deployment concern: a browser, a proxy and a CDN all
// read the same answer, which is why serving the files from somewhere else does
// not remove the need for it.
func cacheFor(asked string) string {
	if strings.HasPrefix(asked, "assets/") {
		return immutable
	}
	return revalidate
}

const (
	// A year, the longest max-age worth writing, plus `immutable` so a reload
	// does not revalidate what cannot have changed.
	immutable = "public, max-age=31536000, immutable"
	// Kept, but never used without asking. `no-store` would be wrong: the copy
	// is worth holding, it is only worth checking before it is trusted.
	revalidate = "no-cache"
)

// Built reports whether a real client was built into this binary.
//
// The placeholder answers no, so an operator learns it at startup rather than
// from a blank page.
func Built() bool {
	entry, err := dist.ReadFile("dist/index.html")
	return err == nil && !strings.Contains(string(entry), placeholderMark)
}

// placeholderMark is what the committed stand-in carries and a real build never
// does.
const placeholderMark = "ozalid-client-not-built"
