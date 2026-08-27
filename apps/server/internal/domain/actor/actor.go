// Package actor names who a recorded fact belongs to.
//
// Every fact ozalid stores carries one (ADR 0002), and an actor is never
// invented: it is resolved from a proven credential and carried down, never
// guessed from what was done (ADR 0018).
package actor

// Kind separates a person from a program.
//
// The journal keeps it beside the id so "was this a human?" is answerable
// without knowing what an id looks like — which is what lets the answer survive
// a change of provider.
type Kind string

const (
	Human   Kind = "human"
	Machine Kind = "machine"
)

// Actor is who did something.
type Actor struct {
	ID   string
	Kind Kind
}

// Zero reports whether nothing was resolved. A handler that reaches an app
// service with one has skipped resolution, which is a bug rather than an
// anonymous caller — anonymity has an id of its own.
func (a Actor) Zero() bool { return a.ID == "" }
