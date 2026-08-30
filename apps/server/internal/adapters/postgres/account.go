package postgres

import (
	"context"
	"errors"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
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

// DeactivateAccount stops an account from signing in. It reports whether the
// account exists; deactivating one already deactivated is not an error.
func (r *Repository) DeactivateAccount(ctx context.Context, id string) (bool, error) {
	rows, err := r.q.DeactivateUser(ctx, id)
	if err != nil {
		return false, translate("deactivating the account", err)
	}
	return rows > 0, nil
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
