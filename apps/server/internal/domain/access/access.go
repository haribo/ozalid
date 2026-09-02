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
	// Admin manages accounts, creates projects, and reaches every one of them
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
// An administrator reaches everything. Membership decides what everybody else
// may do; administering the instance is a way past it, on every project,
// whether or not they were ever added to one (product.md §8.2).
//
// What that gives up is written down where the decision lives: the running of
// the instance can no longer be handed to somebody without also handing them
// every team's work, and an administrator reading a project leaves no trace —
// they used to have to be added to it, and being a member was visible to the
// team. Reads are not journalled; only writes are.
func Allows(s Standing, a Action) bool {
	switch a {
	case ManageAccounts, CreateProject:
		return s.Admin
	case ReadProject:
		return s.Admin || s.Rights == Reader || s.Rights == Member
	case WriteProject:
		// Reading without writing would give an administrator screens where
		// everything shows and every button fails — worse than either whole
		// answer, and useless to one who is also the reviewer.
		return s.Admin || s.Rights == Member
	default:
		// An action nobody wrote a rule for is refused. A default that allowed
		// would make every new endpoint open until someone remembered.
		return false
	}
}
