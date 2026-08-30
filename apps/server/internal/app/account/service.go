package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
)

// Account is an account as an administrator sees it.
//
// Distinct from Person, which is the projection a signed-in reviewer sees of
// themselves: this one carries when the account was made and whether it still
// works, because that is what administering them is about.
type Account struct {
	Person
	// DeactivatedAt is nil while the account works.
	DeactivatedAt *time.Time
	CreatedAt     time.Time
}

// Membership is one holder's place on one project.
type Membership struct {
	AccountID string
	Name      string
	// Email is empty for a service account, which has no address to reach.
	Email string
	// IsPerson is derived from which kind of account holds the membership,
	// never declared (product.md §8).
	IsPerson bool
	Rights   access.Rights
	AddedAt  time.Time
}

// Repository stores accounts.
type Repository interface {
	CreateAccount(ctx context.Context, name, email string, isAdmin bool) (Account, error)
	ListAccounts(ctx context.Context) ([]Account, error)
	DeactivateAccount(ctx context.Context, id string) (bool, error)

	Members(ctx context.Context, slug string) ([]Membership, error)
	Grant(ctx context.Context, slug, userID string, rights access.Rights) (bool, error)
	Revoke(ctx context.Context, slug, userID string) error
}

// Errors the layers above match on.
var (
	// ErrNameRequired means the account was given no name to be known by.
	ErrNameRequired = errors.New("account: a name is required")
	// ErrEmailRequired means there is no address to send a sign-in link to.
	ErrEmailRequired = errors.New("account: an address is required")
	// ErrEmailTaken means another account already has that address.
	ErrEmailTaken = errors.New("account: that address already has an account")
	// ErrNotFound means no account carries that id.
	ErrNotFound = errors.New("account: no such account")
	// ErrUnknownRights means the grant named something that is not a right.
	ErrUnknownRights = errors.New("account: rights are reader or member")
)

// Service makes and retires accounts.
type Service struct{ repo Repository }

// New returns a Service backed by repo.
func New(repo Repository) *Service { return &Service{repo: repo} }

// Create makes an account for a person.
//
// No password is stored: the address is how they sign in, which is why an
// account without one cannot exist (ADR 0019). The account reaches nothing
// until somebody grants it a membership — being known and being allowed are
// two different things (product.md §8.1).
func (s *Service) Create(ctx context.Context, name, email string, isAdmin bool) (Account, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Account{}, ErrNameRequired
	}
	// Lowercased on the way in, because the unique index is on the lowercased
	// address: storing two casings of one address would let the database refuse
	// a row the caller believed was new.
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return Account{}, ErrEmailRequired
	}
	return s.repo.CreateAccount(ctx, name, email, isAdmin)
}

// List returns every account, deactivated ones included.
//
// They are what a name in the journal resolves to; hiding them would leave a
// reviewer's name pointing at nothing (ADR 0018).
func (s *Service) List(ctx context.Context) ([]Account, error) {
	return s.repo.ListAccounts(ctx)
}

// Deactivate stops an account from signing in, and leaves everything it
// recorded exactly as it is.
//
// Idempotent: an account already deactivated is not deactivated twice, and the
// day it happened does not move.
func (s *Service) Deactivate(ctx context.Context, id string) error {
	found, err := s.repo.DeactivateAccount(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

// Members returns who reaches a project, people and programs alike.
func (s *Service) Members(ctx context.Context, slug string) ([]Membership, error) {
	return s.repo.Members(ctx, slug)
}

// Grant puts a person on a project, or changes what they may do there.
//
// Granting to somebody who already holds a membership changes their rights
// rather than failing: an administrator who meant to demote somebody said so,
// and refusing would make them revoke first and risk forgetting the second
// half.
func (s *Service) Grant(ctx context.Context, slug, userID string, rights access.Rights) error {
	switch rights {
	case access.Reader, access.Member:
	default:
		// Two values, and no more (product.md §8.1). A third would be a role
		// nobody uses and nobody dares remove.
		return ErrUnknownRights
	}

	granted, err := s.repo.Grant(ctx, slug, userID, rights)
	if err != nil {
		return err
	}
	if !granted {
		// Either the project or the person is not there. Which of the two is
		// not the caller's business: telling them apart would let somebody map
		// an instance by watching which refusals differ.
		return ErrNotFound
	}
	return nil
}

// Revoke takes a person off a project.
//
// Idempotent: revoking a membership nobody holds changes nothing. Everything
// they recorded stays and keeps naming them (product.md §8).
func (s *Service) Revoke(ctx context.Context, slug, userID string) error {
	return s.repo.Revoke(ctx, slug, userID)
}
