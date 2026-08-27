package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
)

// StandingOf reads what one actor may do on one project.
//
// An actor the database does not know — a deactivated account, an id from
// before identity existed — resolves to nothing rather than to an error.
// Nothing is what the rule then refuses, which is the same answer by a shorter
// road, and it keeps a stale credential from looking like a server fault.
//
// Pass an empty projectID to ask only about the instance: the rights come back
// empty, and only `Admin` carries an answer.
func (r *Repository) StandingOf(
	ctx context.Context, by actor.Actor, projectID string,
) (access.Standing, error) {
	project := &projectID
	if projectID == "" {
		project = nil
	}

	switch by.Kind {
	case actor.Human:
		row, err := r.q.StandingOfUser(ctx, sqlcgen.StandingOfUserParams{
			ID: by.ID, ProjectID: project,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return access.Standing{}, nil
		}
		if err != nil {
			return access.Standing{}, translate("reading a standing", err)
		}
		return access.Standing{Admin: row.IsAdmin, Rights: access.Rights(row.Rights)}, nil

	case actor.Machine:
		row, err := r.q.StandingOfServiceAccount(ctx, sqlcgen.StandingOfServiceAccountParams{
			ID: by.ID, ProjectID: project,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return access.Standing{}, nil
		}
		if err != nil {
			return access.Standing{}, translate("reading a standing", err)
		}
		return access.Standing{Rights: access.Rights(row.Rights)}, nil

	default:
		// A kind nobody defined is nobody. Refusing here rather than guessing
		// keeps an unknown credential from resolving to the nearest thing.
		return access.Standing{}, nil
	}
}
