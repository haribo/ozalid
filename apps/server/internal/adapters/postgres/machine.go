package postgres

import (
	"context"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

// CreateServiceAccount makes a program an account on one project.
//
// The account and its membership are written in one transaction: a service
// account belonging to no project would be a credential reaching nothing that
// nobody would think to remove.
func (r *Repository) CreateServiceAccount(
	ctx context.Context, slug, name, ownerID string, rights access.Rights,
) (account.ServiceAccount, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return account.ServiceAccount{}, fmt.Errorf("beginning the service account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.q.WithTx(tx)

	project, err := q.GetProjectBySlug(ctx, slug)
	if err != nil {
		return account.ServiceAccount{}, translate("reading the project", err)
	}

	// Owned by whoever is creating it: a machine account nobody owns is a
	// machine account nobody revokes (product.md §8).
	made, err := q.CreateServiceAccount(ctx, sqlcgen.CreateServiceAccountParams{
		Name: name, OwnerID: ownerID,
	})
	if err != nil {
		return account.ServiceAccount{}, translate("creating the service account", err)
	}

	if err := q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{
		ProjectID: project.ID, ServiceAccountID: &made.ID, Rights: string(rights),
	}); err != nil {
		return account.ServiceAccount{}, translate("putting the service account on the project", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return account.ServiceAccount{}, fmt.Errorf("committing the service account: %w", err)
	}
	return account.ServiceAccount{ID: made.ID, Name: made.Name, CreatedAt: made.CreatedAt.Time}, nil
}

// DeactivateServiceAccount retires a program on one project.
func (r *Repository) DeactivateServiceAccount(ctx context.Context, slug, id string) (bool, error) {
	rows, err := r.q.DeactivateServiceAccount(ctx, sqlcgen.DeactivateServiceAccountParams{
		ID: id, Slug: slug,
	})
	if err != nil {
		return false, translate("retiring the service account", err)
	}
	return rows > 0, nil
}

// MintToken adds a token to a program, and returns it the one time it can be
// read. Only the hash is stored.
func (r *Repository) MintToken(ctx context.Context, slug, serviceAccountID, label string) (account.MintedToken, error) {
	if _, err := r.q.ServiceAccountInProject(ctx, sqlcgen.ServiceAccountInProjectParams{
		ID: serviceAccountID, Slug: slug,
	}); err != nil {
		return account.MintedToken{}, translate("reading the service account", err)
	}

	token, hash, err := credential.Mint()
	if err != nil {
		return account.MintedToken{}, err
	}
	row, err := r.q.CreateServiceToken(ctx, sqlcgen.CreateServiceTokenParams{
		ServiceAccountID: serviceAccountID, Label: label, TokenHash: hash,
	})
	if err != nil {
		return account.MintedToken{}, translate("minting the token", err)
	}
	return account.MintedToken{
		ID: row.ID, Label: row.Label, Token: token, CreatedAt: row.CreatedAt.Time,
	}, nil
}

// ListTokens returns what a program holds, without the tokens themselves.
func (r *Repository) ListTokens(ctx context.Context, slug, serviceAccountID string) ([]account.ServiceToken, error) {
	if _, err := r.q.ServiceAccountInProject(ctx, sqlcgen.ServiceAccountInProjectParams{
		ID: serviceAccountID, Slug: slug,
	}); err != nil {
		return nil, translate("reading the service account", err)
	}

	rows, err := r.q.ListServiceTokens(ctx, sqlcgen.ListServiceTokensParams{
		ServiceAccountID: serviceAccountID, Slug: slug,
	})
	if err != nil {
		return nil, translate("listing the tokens", err)
	}
	out := make([]account.ServiceToken, 0, len(rows))
	for _, row := range rows {
		t := account.ServiceToken{ID: row.ID, Label: row.Label, CreatedAt: row.CreatedAt.Time}
		if row.LastUsedAt.Valid {
			at := row.LastUsedAt.Time
			t.LastUsedAt = &at
		}
		out = append(out, t)
	}
	return out, nil
}

// RetireToken stops one token working. Retiring one that is already gone is not
// an error: the caller wanted it gone, and it is.
func (r *Repository) RetireToken(ctx context.Context, slug, serviceAccountID, tokenID string) error {
	_, err := r.q.DeleteServiceToken(ctx, sqlcgen.DeleteServiceTokenParams{
		ID: tokenID, ServiceAccountID: serviceAccountID, Slug: slug,
	})
	if err != nil {
		return translate("retiring the token", err)
	}
	return nil
}
