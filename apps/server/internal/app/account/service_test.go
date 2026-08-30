package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/app/account"
)

// recordingRepo remembers what the service handed it.
type recordingRepo struct {
	name, email string
	isAdmin     bool
}

func (r *recordingRepo) CreateAccount(_ context.Context, name, email string, isAdmin bool) (account.Account, error) {
	r.name, r.email, r.isAdmin = name, email, isAdmin
	return account.Account{Person: account.Person{Name: name, Email: email, IsAdmin: isAdmin}}, nil
}

func (r *recordingRepo) ListAccounts(context.Context) ([]account.Account, error) { return nil, nil }

func (r *recordingRepo) DeactivateAccount(context.Context, string) (bool, error) { return true, nil }

// refusingRepo fails the test if anything reaches it.
type refusingRepo struct{ t *testing.T }

func (r refusingRepo) CreateAccount(context.Context, string, string, bool) (account.Account, error) {
	r.t.Error("the account reached the repository, want it refused first")
	return account.Account{}, nil
}

func (r refusingRepo) ListAccounts(context.Context) ([]account.Account, error) { return nil, nil }

func (r refusingRepo) DeactivateAccount(context.Context, string) (bool, error) { return true, nil }

func TestAnAccountWithoutANameIsRefused(t *testing.T) {
	svc := account.New(refusingRepo{t})

	for _, blank := range []string{"", "   ", "\t\n"} {
		if _, err := svc.Create(context.Background(), blank, "nina@example.test", false); !errors.Is(err, account.ErrNameRequired) {
			t.Errorf("Create(%q) = %v, want ErrNameRequired", blank, err)
		}
	}
}

func TestAnAccountWithoutAnAddressIsRefused(t *testing.T) {
	svc := account.New(refusingRepo{t})

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

// absentRepo reports that the row was not there.
type absentRepo struct{}

func (absentRepo) CreateAccount(context.Context, string, string, bool) (account.Account, error) {
	return account.Account{}, nil
}
func (absentRepo) ListAccounts(context.Context) ([]account.Account, error) { return nil, nil }
func (absentRepo) DeactivateAccount(context.Context, string) (bool, error) { return false, nil }
