package access_test

import (
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
)

// The whole rule, as a table. A decision spread across tests is as hard to read
// as one spread across handlers.
func TestWhoMayDoWhat(t *testing.T) {
	cases := []struct {
		name     string
		standing access.Standing
		action   access.Action
		want     bool
	}{
		// Nothing is public: a stranger reaches nothing, not even to read
		// (product.md §8.1).
		{"a stranger cannot read", access.Standing{}, access.ReadProject, false},
		{"a stranger cannot write", access.Standing{}, access.WriteProject, false},

		{"a reader reads", access.Standing{Rights: access.Reader}, access.ReadProject, true},
		{"a reader does not write", access.Standing{Rights: access.Reader}, access.WriteProject, false},

		{"a member reads", access.Standing{Rights: access.Member}, access.ReadProject, true},
		{"a member writes", access.Standing{Rights: access.Member}, access.WriteProject, true},

		// The line that matters: administration reaches accounts, never content
		// (product.md §8.2).
		{"an admin manages accounts", access.Standing{Admin: true}, access.ManageAccounts, true},
		{"an admin creates a project", access.Standing{Admin: true}, access.CreateProject, true},
		{"an admin does not read a project they are not in", access.Standing{Admin: true}, access.ReadProject, false},
		{"an admin does not write a project they are not in", access.Standing{Admin: true}, access.WriteProject, false},

		// Being an administrator neither adds to nor removes from a membership.
		{"an admin who is a member writes", access.Standing{Admin: true, Rights: access.Member}, access.WriteProject, true},

		{"a member does not manage accounts", access.Standing{Rights: access.Member}, access.ManageAccounts, false},
		{"a member does not create a project", access.Standing{Rights: access.Member}, access.CreateProject, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := access.Allows(c.standing, c.action); got != c.want {
				t.Errorf("Allows(%+v, %q) = %v, want %v", c.standing, c.action, got, c.want)
			}
		})
	}
}

func TestAnActionNobodyWroteARuleForIsRefused(t *testing.T) {
	// A default that allowed would leave every new endpoint open until somebody
	// remembered to close it — and nobody remembers.
	everything := access.Standing{Admin: true, Rights: access.Member}
	if access.Allows(everything, access.Action("delete-the-instance")) {
		t.Error("an unknown action was allowed")
	}
}

func TestRightsNobodyGrantedAreNoRights(t *testing.T) {
	// A value the database cannot hold, arriving anyway: refused rather than
	// treated as the nearest thing.
	nonsense := access.Standing{Rights: access.Rights("owner")}
	if access.Allows(nonsense, access.ReadProject) || access.Allows(nonsense, access.WriteProject) {
		t.Error("an unknown rights value was honoured")
	}
}
