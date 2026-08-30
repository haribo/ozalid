package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
)

// nothingRepo satisfies the port and does nothing. The doubles below embed it
// so each one states only the method its test is about; adding a method to the
// port then breaks the one double that should have had an opinion, not all of
// them.
type nothingRepo struct{}

func (nothingRepo) CreateAccount(context.Context, string, string, bool) (account.Account, error) {
	return account.Account{}, nil
}
func (nothingRepo) ListAccounts(context.Context) ([]account.Account, error)       { return nil, nil }
func (nothingRepo) DeactivateAccount(context.Context, string) (bool, error)       { return true, nil }
func (nothingRepo) Members(context.Context, string) ([]account.Membership, error) { return nil, nil }
func (nothingRepo) Grant(context.Context, string, string, access.Rights) (bool, error) {
	return true, nil
}
func (nothingRepo) Revoke(context.Context, string, string) error { return nil }

// recordingRepo remembers what the service handed it.
type recordingRepo struct {
	nothingRepo
	name, email string
	isAdmin     bool
	rights      access.Rights
}

func (r *recordingRepo) CreateAccount(_ context.Context, name, email string, isAdmin bool) (account.Account, error) {
	r.name, r.email, r.isAdmin = name, email, isAdmin
	return account.Account{Person: account.Person{Name: name, Email: email, IsAdmin: isAdmin}}, nil
}

func (r *recordingRepo) Grant(_ context.Context, _, _ string, rights access.Rights) (bool, error) {
	r.rights = rights
	return true, nil
}

// refusingRepo fails the test if anything reaches it.
type refusingRepo struct {
	nothingRepo
	t *testing.T
}

func (r refusingRepo) CreateAccount(context.Context, string, string, bool) (account.Account, error) {
	r.t.Error("the account reached the repository, want it refused first")
	return account.Account{}, nil
}

func (r refusingRepo) Grant(context.Context, string, string, access.Rights) (bool, error) {
	r.t.Error("the grant reached the repository, want it refused first")
	return true, nil
}

// absentRepo reports that the row was not there.
type absentRepo struct{ nothingRepo }

func (absentRepo) DeactivateAccount(context.Context, string) (bool, error) { return false, nil }
func (absentRepo) Grant(context.Context, string, string, access.Rights) (bool, error) {
	return false, nil
}

func TestAnAccountWithoutANameIsRefused(t *testing.T) {
	svc := account.New(refusingRepo{t: t})

	for _, blank := range []string{"", "   ", "\t\n"} {
		if _, err := svc.Create(context.Background(), blank, "nina@example.test", false); !errors.Is(err, account.ErrNameRequired) {
			t.Errorf("Create(%q) = %v, want ErrNameRequired", blank, err)
		}
	}
}

func TestAnAccountWithoutAnAddressIsRefused(t *testing.T) {
	svc := account.New(refusingRepo{t: t})

	// There is no password, so the address is the only way in. An account
	// without one could never sign in (ADR 0019).
	if _, err := svc.Create(context.Background(), "nina", "  ", false); !errors.Is(err, account.ErrEmailRequired) {
		t.Errorf("Create with a blank address = %v, want ErrEmailRequired", err)
	}
}

func TestAnAddressIsStoredLowercased(t *testing.T) {
	repo := &recordingRepo{}
	svc := account.New(repo)

	// The unique index is on the lowercased address. Storing two casings of one
	// address would let the database refuse a row the caller believed was new.
	if _, err := svc.Create(context.Background(), "  Nina  ", " Nina@Example.TEST ", true); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if repo.email != "nina@example.test" {
		t.Errorf("email stored = %q, want it lowercased and trimmed", repo.email)
	}
	if repo.name != "Nina" {
		t.Errorf("name stored = %q, want it trimmed", repo.name)
	}
	if !repo.isAdmin {
		t.Error("isAdmin was not carried through")
	}
}

func TestDeactivatingAnAccountNobodyHas(t *testing.T) {
	svc := account.New(absentRepo{})

	if err := svc.Deactivate(context.Background(), "nobody"); !errors.Is(err, account.ErrNotFound) {
		t.Errorf("Deactivate = %v, want ErrNotFound", err)
	}
}

func TestRightsThatAreNotRightsAreRefused(t *testing.T) {
	svc := account.New(refusingRepo{t: t})

	// Two values, and no more (product.md §8.1). A third would be a role
	// nobody uses and nobody dares remove.
	for _, madeUp := range []access.Rights{"", "admin", "reviewer", "developer", "Member"} {
		if err := svc.Grant(context.Background(), "atlas", "nina", madeUp); !errors.Is(err, account.ErrUnknownRights) {
			t.Errorf("Grant(%q) = %v, want ErrUnknownRights", madeUp, err)
		}
	}
}

func TestBothRightsAreCarriedThrough(t *testing.T) {
	for _, r := range []access.Rights{access.Reader, access.Member} {
		repo := &recordingRepo{}
		if err := account.New(repo).Grant(context.Background(), "atlas", "nina", r); err != nil {
			t.Fatalf("Grant(%q): %v", r, err)
		}
		if repo.rights != r {
			t.Errorf("rights stored = %q, want %q", repo.rights, r)
		}
	}
}

func TestGrantingOnAProjectOrToAPersonNobodyHas(t *testing.T) {
	svc := account.New(absentRepo{})

	// Which of the two is missing is not the caller's business: telling them
	// apart would let somebody map an instance by watching which refusals
	// differ.
	if err := svc.Grant(context.Background(), "nowhere", "nobody", access.Member); !errors.Is(err, account.ErrNotFound) {
		t.Errorf("Grant = %v, want ErrNotFound", err)
	}
}
