package http

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Rate limits for asking a sign-in link, and where the numbers come from.
//
// **Per address, three in fifteen minutes** — the lifetime of a link
// (`credential.LinkLifetime`). Asking for a fourth inside that window means the
// first three still work, so the extra one is not somebody who needs a link.
//
// **Per source, twenty in fifteen minutes** — an office behind one address
// signing in after a deploy fits; walking a list of addresses to learn which
// have accounts does not.
const (
	perAddress = 3
	perSource  = 20
	limitAfter = 15 * time.Minute
)

// window counts what one key did inside a fixed stretch of time.
//
// Kept in memory rather than in the database, deliberately. ozalid runs as one
// process (see the Dockerfile), and a database round-trip on every attempt
// would make this endpoint a way to load the database — the opposite of what a
// limiter is for. It resets on restart, which costs a window of minutes, and
// the day this runs as more than one process the sharing has to be decided
// again anyway.
type window struct {
	mu    sync.Mutex
	seen  map[string]*counted
	limit int
	every time.Duration
	// now is the clock, handed in rather than read: this package receives its
	// I/O through interfaces so a use case stays testable without one
	// (backend ADR 0001). It is also what lets a test watch a window turn over
	// without waiting for it.
	now func() time.Time
}

type counted struct {
	count int
	until time.Time
}

func newWindow(limit int, every time.Duration, now func() time.Time) *window {
	if now == nil {
		// Loudly, at startup, rather than as a nil dereference on the first
		// sign-in: a rate limit that is not there is a rate limit nobody
		// notices is missing.
		panic("a rate limit needs a clock: pass Deps.Now")
	}
	return &window{seen: map[string]*counted{}, limit: limit, every: every, now: now}
}

// allow records one attempt and reports whether it is within the limit. When it
// is not, it also says how long until the window turns over, which is what the
// caller is told rather than left to guess.
func (w *window) allow(key string) (time.Duration, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := w.now()
	c, held := w.seen[key]
	if !held || now.After(c.until) {
		w.seen[key] = &counted{count: 1, until: now.Add(w.every)}
		w.sweep(now)
		return 0, true
	}
	if c.count >= w.limit {
		return c.until.Sub(now), false
	}
	c.count++
	return 0, true
}

// sweep drops windows that have turned over. Called on the way past rather than
// on a timer: a map that only grows is a slow leak, and a goroutine that only
// exists to empty one is a moving part nobody watches.
func (w *window) sweep(now time.Time) {
	for key, c := range w.seen {
		if now.After(c.until) {
			delete(w.seen, key)
		}
	}
}

type sourceKey struct{}

// withSource remembers where a request came from, so a handler that has the
// parsed body can limit on both at once.
//
// `X-Forwarded-For` is read only when the deployment says a proxy is in front
// (`OZALID_TRUSTED_PROXY`). Trusting it otherwise would let anybody claim any
// source and walk straight past the limit; ignoring it behind a real proxy
// makes every request look like one source, which turns the per-source limit
// into a per-instance one. Neither is safe by default, so the deployment says
// which it is.
func withSource(trustProxy bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := r.RemoteAddr
		if host, _, err := net.SplitHostPort(source); err == nil {
			source = host
		}
		if trustProxy {
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				source, _, _ = strings.Cut(forwarded, ",")
				source = strings.TrimSpace(source)
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sourceKey{}, source)))
	})
}

func sourceOf(ctx context.Context) string {
	source, _ := ctx.Value(sourceKey{}).(string)
	return source
}
