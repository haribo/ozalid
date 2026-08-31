package postgres

import (
	"context"
	"errors"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
)

// CreateAccount stores an account for a person.
func (r *Repository) CreateAccount(ctx context.Context, name, email string, isAdmin bool) (account.Account, error) {
	row, err := r.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Name: name, Email: email, IsAdmin: isAdmin,
	})
	if err != nil {
		err = translate("creating the account", err)
		if errors.Is(err, app.ErrConflict) {
			// The only unique index on this table is the address, so a
			// conflict here is that and nothing else.
			return account.Account{}, account.ErrEmailTaken
		}
		return account.Account{}, err
	}
	return toAccount(row), nil
}

// ListAccounts returns every account, deactivated ones included.
func (r *Repository) ListAccounts(ctx context.Context) ([]account.Account, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, translate("listing the accounts", err)
	}
	out := make([]account.Account, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAccount(row))
	}
	return out, nil
}

// DeactivateAccount stops an account from signing in.
//
// It reports whether the write happened, which is false both for an account
// nobody has and for the last administrator: the query refuses by matching no
// row, so the two are told apart above rather than here (#90).
func (r *Repository) DeactivateAccount(ctx context.Context, id string) (bool, error) {
	rows, err := r.q.DeactivateUser(ctx, id)
	if err != nil {
		return false, translate("deactivating the account", err)
	}
	return rows > 0, nil
}

// SetAdmin promotes or demotes an account, refusing to demote the last
// administrator the same way.
func (r *Repository) SetAdmin(ctx context.Context, id string, isAdmin bool) (bool, error) {
	rows, err := r.q.SetUserAdmin(ctx, sqlcgen.SetUserAdminParams{ID: id, IsAdmin: isAdmin})
	if err != nil {
		return false, translate("changing the role", err)
	}
	return rows > 0, nil
}

// AccountExists answers whether an id names anybody at all.
func (r *Repository) AccountExists(ctx context.Context, id string) (bool, error) {
	found, err := r.q.UserExists(ctx, id)
	if err != nil {
		return false, translate("reading the account", err)
	}
	return found, nil
}

func toAccount(row sqlcgen.User) account.Account {
	out := account.Account{
		Person: account.Person{
			ID: row.ID, Name: row.Name, Email: row.Email, IsAdmin: row.IsAdmin,
		},
		CreatedAt: row.CreatedAt.Time,
	}
	if row.DeactivatedAt.Valid {
		at := row.DeactivatedAt.Time
		out.DeactivatedAt = &at
	}
	return out
}

// Members returns who reaches a project, people first.
func (r *Repository) Members(ctx context.Context, slug string) ([]account.Membership, error) {
	rows, err := r.q.ListProjectMembers(ctx, slug)
	if err != nil {
		return nil, translate("listing the members", err)
	}
	out := make([]account.Membership, 0, len(rows))
	for _, row := range rows {
		m := account.Membership{
			AccountID: row.AccountID, Name: row.Name, IsPerson: row.IsPerson,
			Rights: access.Rights(row.Rights), AddedAt: row.AddedAt.Time,
		}
		if row.Email != nil {
			m.Email = *row.Email
		}
		if !row.IsPerson {
			held := int(row.Tokens)
			m.Tokens = &held
		}
		out = append(out, m)
	}
	return out, nil
}

// Grant puts a person on a project, or changes their rights on it. It reports
// whether both the project and the person exist; neither is named to the
// caller when one does not.
func (r *Repository) Grant(ctx context.Context, slug, userID string, rights access.Rights) (bool, error) {
	rows, err := r.q.GrantMembership(ctx, sqlcgen.GrantMembershipParams{
		Slug: slug, UserID: userID, Rights: string(rights),
	})
	if err != nil {
		return false, translate("granting the membership", err)
	}
	return rows > 0, nil
}

// Revoke takes a person off a project. Revoking one nobody holds is not an
// error: the caller wanted them off, and they are.
func (r *Repository) Revoke(ctx context.Context, slug, userID string) error {
	if _, err := r.q.RevokeMembership(ctx, sqlcgen.RevokeMembershipParams{
		Slug: slug, UserID: &userID,
	}); err != nil {
		return translate("revoking the membership", err)
	}
	return nil
}
