// Package access decides what an actor may do.
//
// One place answers, and the handlers ask. A rule spread across handlers is a
// rule with exceptions nobody wrote down — and the exception is always the one
// that mattered.
//
// Everything here is a pure function over what the caller already knows. No
// clock, no database, no request: the same inputs always give the same answer,
// which is what lets a refusal be replayed and checked.
package access

// Rights is what a membership carries (product.md §8.1).
//
// Two values, and no more. The one distinction met so far is a product owner
// following a review with nothing to validate — a stray click from them would
// falsify the book. Inventing more roles before meeting the need produces roles
// nobody uses and nobody dares remove.
type Rights string

const (
	// Reader sees everything and changes nothing.
	Reader Rights = "reader"
	// Member does everything, on the projects they belong to.
	Member Rights = "member"
)

// Standing is what the server knows about a caller when it decides.
type Standing struct {
	// Admin manages accounts and creates projects. It reaches no content
	// (product.md §8.2).
	Admin bool
	// Rights on the project being reached, or empty when the caller is not a
	// member of it. Empty is the important value: nothing is public.
	Rights Rights
}

// Action is what is being attempted.
type Action string

const (
	// ReadProject covers the catalogue, a case, its grid and its comments.
	ReadProject Action = "read-project"
	// WriteProject covers judging, commenting, taking an edition in — anything
	// that records a fact about a case.
	WriteProject Action = "write-project"
	// ManageAccounts covers creating and deactivating accounts.
	ManageAccounts Action = "manage-accounts"
	// CreateProject covers creating a project and naming its first member.
	CreateProject Action = "create-project"
)

// Allows reports whether a caller of this standing may take this action.
//
// An administrator is deliberately absent from the content answers. On an
// instance carrying several projects, one who could read every review book
// would see every team's work; separating administration from access to content
// is what lets the running of the instance be handed to someone without handing
// them everything. When they do need a project, they are added to it — which
// leaves a trace, exactly what an exceptional access should do.
func Allows(s Standing, a Action) bool {
	switch a {
	case ManageAccounts, CreateProject:
		return s.Admin
	case ReadProject:
		return s.Rights == Reader || s.Rights == Member
	case WriteProject:
		return s.Rights == Member
	default:
		// An action nobody wrote a rule for is refused. A default that allowed
		// would make every new endpoint open until someone remembered.
		return false
	}
}
