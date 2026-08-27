package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
)

// accessFixture gives a repository and a project of its own.
//
// These tests read through the repository rather than through a transaction,
// because the method under test is on the repository — and a test-only door in
// production code to avoid that would be a worse trade than an explicit
// cleanup. Accounts are not under a project, so they are removed by hand.
func accessFixture(t *testing.T) (context.Context, *postgres.Repository, sqlcgen.Project) {
	t.Helper()
	ctx, repo, project, _ := intakeFixture(t)
	t.Cleanup(func() {
		mine := fmt.Sprintf("%%@%s.test", t.Name())
		// Service accounts go first: their owner cannot be removed while they
		// exist, which is the point of the constraint rather than an obstacle.
		if _, err := repo.Pool().Exec(ctx,
			`DELETE FROM service_accounts
			 WHERE owner_id IN (SELECT id FROM users WHERE email LIKE $1)`, mine); err != nil {
			t.Errorf("cleaning up the service accounts: %v", err)
		}
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM users WHERE email LIKE $1", mine); err != nil {
			t.Errorf("cleaning up the accounts: %v", err)
		}
	})
	return ctx, repo, project
}

func person(t *testing.T, ctx context.Context, q *sqlcgen.Queries, name string, admin bool) sqlcgen.User {
	t.Helper()
	u, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Name: name, Email: fmt.Sprintf("%s-%d@%s.test", name, time.Now().UnixNano(), t.Name()), IsAdmin: admin,
	})
	if err != nil {
		t.Fatalf("creating a user: %v", err)
	}
	return u
}

func belongs(t *testing.T, ctx context.Context, q *sqlcgen.Queries, projectID, userID string, rights access.Rights) {
	t.Helper()
	if err := q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{
		ProjectID: projectID, UserID: &userID, Rights: string(rights),
	}); err != nil {
		t.Fatalf("adding a member: %v", err)
	}
}

func TestAStrangerHasNoStandingOnAProject(t *testing.T) {
	// Nothing is public (product.md §8.1). Someone with an account but no
	// membership reaches a project the same way someone with no account does.
	ctx, repo, project := accessFixture(t)
	q := repo.Queries()
	outsider := person(t, ctx, q, "outsider", false)

	standing, err := repo.StandingOf(ctx, actor.Actor{ID: outsider.ID, Kind: actor.Human}, project.ID)
	if err != nil {
		t.Fatalf("reading the standing: %v", err)
	}
	if standing.Rights != "" {
		t.Errorf("rights = %q, want none", standing.Rights)
	}
	if access.Allows(standing, access.ReadProject) {
		t.Error("a stranger was allowed to read")
	}
}

func TestAReaderReadsAndDoesNotWrite(t *testing.T) {
	ctx, repo, project := accessFixture(t)
	q := repo.Queries()
	watcher := person(t, ctx, q, "watcher", false)
	belongs(t, ctx, q, project.ID, watcher.ID, access.Reader)

	standing, err := repo.StandingOf(ctx, actor.Actor{ID: watcher.ID, Kind: actor.Human}, project.ID)
	if err != nil {
		t.Fatalf("reading the standing: %v", err)
	}
	if !access.Allows(standing, access.ReadProject) {
		t.Error("a reader could not read")
	}
	if access.Allows(standing, access.WriteProject) {
		t.Error("a reader was allowed to write — a stray click would falsify the book")
	}
}

func TestAnAdministratorReachesNoContent(t *testing.T) {
	// The line the whole section exists for: administration reaches accounts,
	// never the content of a project it is not a member of (product.md §8.2).
	ctx, repo, project := accessFixture(t)
	q := repo.Queries()
	admin := person(t, ctx, q, "admin", true)

	standing, err := repo.StandingOf(ctx, actor.Actor{ID: admin.ID, Kind: actor.Human}, project.ID)
	if err != nil {
		t.Fatalf("reading the standing: %v", err)
	}
	if !access.Allows(standing, access.ManageAccounts) {
		t.Error("an administrator could not manage accounts")
	}
	if access.Allows(standing, access.ReadProject) {
		t.Error("an administrator read a project they are not a member of")
	}
}

func TestADeactivatedAccountStandsForNothing(t *testing.T) {
	ctx, repo, project := accessFixture(t)
	q := repo.Queries()
	gone := person(t, ctx, q, "gone", true)
	belongs(t, ctx, q, project.ID, gone.ID, access.Member)

	if err := q.DeactivateUser(ctx, gone.ID); err != nil {
		t.Fatalf("deactivating: %v", err)
	}

	standing, err := repo.StandingOf(ctx, actor.Actor{ID: gone.ID, Kind: actor.Human}, project.ID)
	if err != nil {
		t.Fatalf("reading the standing: %v", err)
	}
	// Deactivated, not deleted: what they reviewed stays readable and the
	// journal still names them, but the door is shut.
	if standing.Admin || standing.Rights != "" {
		t.Errorf("standing = %+v, want nothing at all", standing)
	}
}

func TestAServiceAccountBelongsToOneProjectAndNeverAdministers(t *testing.T) {
	ctx, repo, first := accessFixture(t)
	q := repo.Queries()
	owner := person(t, ctx, q, "owner", true)

	bot, err := q.CreateServiceAccount(ctx, sqlcgen.CreateServiceAccountParams{
		Name: "atlas-ci", OwnerID: owner.ID,
	})
	if err != nil {
		t.Fatalf("creating a service account: %v", err)
	}
	if err := q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{
		ProjectID: first.ID, ServiceAccountID: &bot.ID, Rights: string(access.Member),
	}); err != nil {
		t.Fatalf("adding the service account: %v", err)
	}

	standing, err := repo.StandingOf(ctx, actor.Actor{ID: bot.ID, Kind: actor.Machine}, first.ID)
	if err != nil {
		t.Fatalf("reading the standing: %v", err)
	}
	// Nothing is forbidden because the caller is a program (ADR 0019).
	if !access.Allows(standing, access.WriteProject) {
		t.Error("a service account holding a membership could not write")
	}
	// But administration reaches accounts, and a program that could make
	// accounts could make itself another.
	if standing.Admin {
		t.Error("a service account administers")
	}
}

func TestOneProjectPerServiceAccountIsEnforcedNotTrusted(t *testing.T) {
	// ADR 0018 asked for it; the partial unique index is what makes it true
	// rather than a rule somebody has to remember.
	ctx, repo, first := accessFixture(t)
	q := repo.Queries()
	owner := person(t, ctx, q, "owner", false)

	bot, err := q.CreateServiceAccount(ctx, sqlcgen.CreateServiceAccountParams{
		Name: "atlas-ci", OwnerID: owner.ID,
	})
	if err != nil {
		t.Fatalf("creating a service account: %v", err)
	}
	add := func(projectID string) error {
		return q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{
			ProjectID: projectID, ServiceAccountID: &bot.ID, Rights: string(access.Member),
		})
	}
	if err := add(first.ID); err != nil {
		t.Fatalf("adding to the first project: %v", err)
	}

	second, err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Slug: fmt.Sprintf("second-%d", time.Now().UnixNano()), Name: "second", IntakePolicy: "per-case",
	})
	if err != nil {
		t.Fatalf("creating a second project: %v", err)
	}
	t.Cleanup(func() {
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM projects WHERE id = $1", second.ID); err != nil {
			t.Errorf("cleaning up the second project: %v", err)
		}
	})

	if err := add(second.ID); err == nil {
		t.Error("a service account reached two projects")
	}
}
